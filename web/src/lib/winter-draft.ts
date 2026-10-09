/** Appends one order line to a draft, on its own line. */
export function appendDraftLine(draft: string, line: string): string {
  const trimmed = draft.replace(/\s+$/, '')
  return trimmed === '' ? `${line}\n` : `${trimmed}\n${line}\n`
}

/** Order lines of a draft, upper-cased and stripped of their `# comment`. */
export function draftLines(draft: string): string[] {
  return draft
    .split('\n')
    .map((line) => line.replace(/#.*$/, '').trim().toUpperCase())
    .filter((line) => line !== '')
}
