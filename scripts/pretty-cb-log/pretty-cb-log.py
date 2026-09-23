#!/usr/bin/env python3
"""Display a Crown & Borough chat log in a readable format.

The log format is intentionally simple, but the raw channel and player IDs are
not especially pleasant to read. This script resolves player display names
from the persona.json files stored next to a run and can follow a growing log
without requiring third-party packages.

Usage
-----

Invoke the script directly; run lookups are resolved from
``~/.crown-borough/run/`` regardless of the current directory.

Follow a run live. Existing messages are displayed first, then new messages
are formatted as they arrive. Press Ctrl-C to stop:

    scripts/pretty-cb-log/pretty-cb-log.py \\
        --follow \\
        --run afdaf864-e94e-48ba-9b14-6c1890df4ad1

Display a log once using its complete path:

    scripts/pretty-cb-log/pretty-cb-log.py \\
        ~/.crown-borough/run/afdaf864-e94e-48ba-9b14-6c1890df4ad1/chat.log

Display only public messages:

    scripts/pretty-cb-log/pretty-cb-log.py \\
        --public-only \\
        --run afdaf864-e94e-48ba-9b14-6c1890df4ad1

Display private messages involving one player:

    scripts/pretty-cb-log/pretty-cb-log.py \\
        --dm cb-08ce6364 \\
        --run afdaf864-e94e-48ba-9b14-6c1890df4ad1

Use a narrower message width:

    scripts/pretty-cb-log/pretty-cb-log.py \\
        --width 80 \\
        --run afdaf864-e94e-48ba-9b14-6c1890df4ad1

Alternatively, pipe a log through ``tail -F``. The ``--run`` option supplies
the persona directory used to resolve player names:

    tail -F ~/.crown-borough/run/afdaf864-e94e-48ba-9b14-6c1890df4ad1/chat.log \\
        | scripts/pretty-cb-log/pretty-cb-log.py \\
            --pipe \\
            --run afdaf864-e94e-48ba-9b14-6c1890df4ad1

Colors are enabled automatically for interactive terminals and disabled when
output is redirected. Use ``--color`` to force colors or ``--no-color`` to
disable them.
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import re
import shutil
import sys
import textwrap
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Optional, TextIO


LOG_LINE = re.compile(
    r"^\[(?P<timestamp>[^\]]+)\]"
    r" \[ch=(?P<channel>[^\]]+)\]"
    r" \[from=(?P<sender>[^\]]+)\]"
    r" \[to=(?P<recipient>[^\]]+)\]"
    r" (?P<message>.*)$"
)


@dataclass(frozen=True)
class Message:
    timestamp: str
    channel: str
    sender: str
    recipient: str
    text: str


class Styles:
    """Small ANSI style wrapper that can be disabled for redirected output."""

    COLORS = (
        "\033[38;5;75m",
        "\033[38;5;178m",
        "\033[38;5;141m",
        "\033[38;5;114m",
        "\033[38;5;215m",
        "\033[38;5;80m",
    )

    def __init__(self, enabled: bool) -> None:
        self.enabled = enabled
        self.reset = "\033[0m" if enabled else ""
        self.dim = "\033[2m" if enabled else ""
        self.bold = "\033[1m" if enabled else ""
        self.cyan = "\033[36m" if enabled else ""
        self.green = "\033[32m" if enabled else ""

    def sender(self, player_id: str) -> str:
        if not self.enabled:
            return ""
        digest = hashlib.sha256(player_id.encode("utf-8")).digest()
        return self.COLORS[digest[0] % len(self.COLORS)]

    def wrap(self, prefix: str, value: str) -> str:
        return f"{prefix}{value}{self.reset}"


class NameResolver:
    """Resolve IDs using persona files in one run directory."""

    def __init__(self, run_dir: Optional[Path]) -> None:
        self.names: dict[str, str] = {}
        self.duplicates: set[str] = set()
        if run_dir is None:
            return

        try:
            entries = sorted(run_dir.iterdir())
        except OSError:
            return

        for entry in entries:
            if not entry.is_dir():
                continue
            persona_path = entry / "persona.json"
            try:
                with persona_path.open("r", encoding="utf-8") as persona_file:
                    persona = json.load(persona_file)
            except (OSError, json.JSONDecodeError):
                continue

            display_name = persona.get("display_name")
            if isinstance(display_name, str) and display_name:
                self.names[entry.name] = display_name

        counts: dict[str, int] = {}
        for display_name in self.names.values():
            counts[display_name] = counts.get(display_name, 0) + 1
        self.duplicates = {
            display_name for display_name, count in counts.items() if count > 1
        }

    def label(self, player_id: str) -> str:
        if player_id == "table":
            return "Everyone"

        display_name = self.names.get(player_id)
        if display_name is None:
            return player_id
        if display_name in self.duplicates:
            return f"{display_name} [{player_id}]"
        return display_name


def parse_line(line: str) -> Optional[Message]:
    match = LOG_LINE.fullmatch(line.rstrip("\r\n"))
    if match is None:
        return None
    return Message(
        timestamp=match.group("timestamp"),
        channel=match.group("channel"),
        sender=match.group("sender"),
        recipient=match.group("recipient"),
        text=match.group("message"),
    )


def parse_timestamp(value: str) -> Optional[dt.datetime]:
    try:
        return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def format_timestamp(value: str) -> str:
    parsed = parse_timestamp(value)
    if parsed is None:
        return value
    if parsed.tzinfo is not None:
        parsed = parsed.astimezone(dt.timezone.utc)
        return parsed.strftime("%Y-%m-%d %H:%M:%S UTC")
    return parsed.strftime("%Y-%m-%d %H:%M:%S")


def date_label(value: str) -> str:
    parsed = parse_timestamp(value)
    if parsed is None:
        return "unknown date"
    return parsed.date().isoformat()


class Formatter:
    def __init__(
        self,
        resolver: NameResolver,
        styles: Styles,
        output: TextIO,
        width: int,
        public_only: bool,
        dm_player: Optional[str],
    ) -> None:
        self.resolver = resolver
        self.styles = styles
        self.output = output
        self.width = max(40, width)
        self.public_only = public_only
        self.dm_player = dm_player
        self.last_date: Optional[str] = None

    def accepts(self, message: Message) -> bool:
        is_public = message.channel == "table"
        if self.public_only and not is_public:
            return False
        if self.dm_player is not None:
            if is_public:
                return False
            participants = {message.sender, message.recipient}
            if self.dm_player not in participants:
                return False
        return True

    def print_message(self, message: Message) -> None:
        if not self.accepts(message):
            return

        current_date = date_label(message.timestamp)
        if current_date != self.last_date:
            if self.last_date is not None:
                self.output.write("\n")
            divider = f"--- {current_date} "
            divider += "-" * max(0, self.width - len(divider))
            self.output.write(self.styles.wrap(self.styles.dim, divider) + "\n")
            self.last_date = current_date

        is_public = message.channel == "table"
        kind = "PUBLIC" if is_public else "DM"
        sender = self.resolver.label(message.sender)
        recipient = "Everyone" if is_public else self.resolver.label(message.recipient)

        timestamp = self.styles.wrap(self.styles.dim, format_timestamp(message.timestamp))
        kind_style = self.styles.green if is_public else self.styles.cyan
        kind_value = self.styles.wrap(kind_style, f"{kind:<7}")
        sender_value = self.styles.wrap(
            self.styles.bold + self.styles.sender(message.sender), sender
        )
        route = f"{timestamp}  {kind_value} {sender_value} -> {recipient}"
        self.output.write(route + "\n")

        body_width = max(20, self.width - 4)
        wrapped = textwrap.wrap(
            message.text,
            width=body_width,
            break_long_words=True,
            break_on_hyphens=False,
        ) or [""]
        for body_line in wrapped:
            self.output.write(f"    {body_line}\n")
        self.output.flush()

    def print_unparsed(self, line: str) -> None:
        if self.public_only or self.dm_player is not None:
            return
        value = line.rstrip("\r\n")
        if not value:
            return
        label = self.styles.wrap(self.styles.dim, "[unparsed]")
        self.output.write(f"{label} {value}\n")
        self.output.flush()

    def process_line(self, line: str) -> None:
        message = parse_line(line)
        if message is None:
            self.print_unparsed(line)
        else:
            self.print_message(message)


def follow_file(path: Path, stream: TextIO, formatter: Formatter) -> None:
    """Follow appends and reopen the file if it is replaced or truncated."""

    current_inode = os.fstat(stream.fileno()).st_ino
    try:
        while True:
            line = stream.readline()
            if line:
                formatter.process_line(line)
                continue

            time.sleep(0.2)
            try:
                status = path.stat()
            except FileNotFoundError:
                continue

            position = stream.tell()
            if status.st_ino == current_inode and status.st_size >= position:
                continue

            stream.close()
            stream = path.open("r", encoding="utf-8", errors="replace")
            current_inode = os.fstat(stream.fileno()).st_ino
            for line in stream:
                formatter.process_line(line)
    finally:
        stream.close()


def read_file(path: Path, formatter: Formatter, follow: bool) -> None:
    with path.open("r", encoding="utf-8", errors="replace") as stream:
        for line in stream:
            formatter.process_line(line)
        if follow:
            follow_file(path, stream, formatter)


def read_pipe(formatter: Formatter) -> None:
    for line in sys.stdin:
        formatter.process_line(line)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Display a Crown & Borough chat log with readable names and formatting."
    )
    parser.add_argument(
        "log_file",
        nargs="?",
        help="path to chat.log (omit when using --run or --pipe)",
    )
    parser.add_argument(
        "--run",
        metavar="RUN_ID",
        help="read ~/.crown-borough/run/RUN_ID/chat.log and its persona files",
    )
    parser.add_argument(
        "--pipe",
        action="store_true",
        help="format log lines from stdin; combine with --run to resolve persona names",
    )
    parser.add_argument(
        "--follow",
        "-f",
        action="store_true",
        help="keep watching for new lines and file replacement",
    )
    filters = parser.add_mutually_exclusive_group()
    filters.add_argument(
        "--public-only",
        action="store_true",
        help="show only messages sent to the public table",
    )
    filters.add_argument(
        "--dm",
        metavar="PLAYER_ID",
        help="show only private messages involving PLAYER_ID",
    )
    colors = parser.add_mutually_exclusive_group()
    colors.add_argument(
        "--color",
        action="store_true",
        help="force ANSI colors even when output is redirected",
    )
    colors.add_argument(
        "--no-color",
        action="store_true",
        help="disable ANSI colors",
    )
    parser.add_argument(
        "--width",
        type=int,
        help="message width in columns (default: terminal width or 100)",
    )
    return parser


def resolve_inputs(
    parser: argparse.ArgumentParser, args: argparse.Namespace
) -> tuple[Optional[Path], Optional[Path]]:
    if args.pipe and args.log_file is not None:
        parser.error("--pipe cannot be used with a log file path")
    if args.run is not None and args.log_file is not None:
        parser.error("choose either --run or a log file path, not both")
    if args.follow and args.pipe:
        parser.error("--follow is for files; stdin already remains open while it is followed")
    if args.width is not None and args.width < 40:
        parser.error("--width must be at least 40 columns")

    if args.run is not None:
        run_dir = Path.home() / ".crown-borough" / "run" / args.run
        return (None if args.pipe else run_dir / "chat.log"), run_dir
    if args.log_file is not None:
        log_path = Path(args.log_file).expanduser()
        return log_path, log_path.parent
    if args.pipe:
        return None, None
    parser.error("provide a log file path, --run RUN_ID, or --pipe")
    return None, None


def display_width(explicit_width: Optional[int]) -> int:
    if explicit_width is not None:
        return explicit_width
    if not sys.stdout.isatty():
        return 100
    return max(60, shutil.get_terminal_size((100, 24)).columns)


def print_header(
    output: TextIO,
    styles: Styles,
    source: str,
    follow: bool,
    public_only: bool,
    dm_player: Optional[str],
) -> None:
    title = styles.wrap(styles.bold, "Crown & Borough chat")
    output.write(f"{title}\n")
    output.write(f"  {source}\n")
    modes: list[str] = []
    if public_only:
        modes.append("public only")
    if dm_player is not None:
        modes.append(f"DMs involving {dm_player}")
    if follow:
        modes.append("following; Ctrl-C to stop")
    if modes:
        output.write(f"  {' | '.join(modes)}\n")
    output.write("\n")
    output.flush()


def main(argv: Optional[Iterable[str]] = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    log_path, run_dir = resolve_inputs(parser, args)

    use_color = args.color or (not args.no_color and sys.stdout.isatty())
    styles = Styles(use_color)
    resolver = NameResolver(run_dir)
    formatter = Formatter(
        resolver=resolver,
        styles=styles,
        output=sys.stdout,
        width=display_width(args.width),
        public_only=args.public_only,
        dm_player=args.dm,
    )

    source = "stdin" if args.pipe else str(log_path)
    print_header(
        output=sys.stdout,
        styles=styles,
        source=source,
        follow=args.follow,
        public_only=args.public_only,
        dm_player=args.dm,
    )

    try:
        if args.pipe:
            read_pipe(formatter)
        else:
            assert log_path is not None
            read_file(log_path, formatter, args.follow)
    except FileNotFoundError as error:
        parser.error(f"cannot open {error.filename}: file does not exist")
    except PermissionError as error:
        parser.error(f"cannot open {error.filename}: permission denied")
    except KeyboardInterrupt:
        sys.stderr.write("\nStopped.\n")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
