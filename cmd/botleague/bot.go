package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Bot is one house played by one language model.
type Bot struct {
	cfg     PlayerConfig
	house   string
	persona Persona
	client  *gameClient
	llm     LLM
	dir     string
	league  *League

	mu      sync.Mutex
	system  string
	memory  string
	history []string
	seen    map[string]bool
	notes   []string
	usage   Usage
	callSeq int
}

const (
	historyKept        = 8
	chatPublicWindow   = 15
	chatPrivateWindow  = 10
	maxMessagesPerCall = 4
)

func (b *Bot) path(name string) string { return filepath.Join(b.dir, name) }

// load restores the private memory, history and read cursor of a previous run.
func (b *Bot) load() {
	if raw, err := os.ReadFile(b.path("memory.md")); err == nil {
		b.memory = string(raw)
	}
	if b.memory == "" {
		b.memory = b.persona.Anchor(b.cfg.Name+" ("+b.house+")") + "\n\n(no notes yet)"
	}
	if raw, err := os.ReadFile(b.path("history.md")); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			if line != "" {
				b.history = append(b.history, line)
			}
		}
	}
	if entries, err := os.ReadDir(b.path("calls")); err == nil {
		b.callSeq = len(entries)
	}
	b.seen = map[string]bool{}
	if raw, err := os.ReadFile(b.path("seen.json")); err == nil {
		var ids []string
		if json.Unmarshal(raw, &ids) == nil {
			for _, id := range ids {
				b.seen[id] = true
			}
		}
	}
}

func (b *Bot) saveMemory() {
	_ = os.MkdirAll(b.dir, 0o755)
	_ = os.WriteFile(b.path("memory.md"), []byte(b.memory), 0o600)
}

func (b *Bot) saveSeen() {
	ids := make([]string, 0, len(b.seen))
	for id := range b.seen {
		ids = append(ids, id)
	}
	raw, _ := json.Marshal(ids)
	_ = os.WriteFile(b.path("seen.json"), raw, 0o600)
}

// Remember appends a line to the orchestrator-written history of the bot.
func (b *Bot) Remember(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.history = append(b.history, line)
	if len(b.history) > historyKept {
		b.history = b.history[len(b.history)-historyKept:]
	}
	_ = os.WriteFile(b.path("history.md"), []byte(strings.Join(b.history, "\n")+"\n"), 0o600)
}

// updateMemory replaces the memory by the model's version, keeping the
// identity anchor and the size bound.
func (b *Bot) updateMemory(memory string) {
	memory = strings.TrimSpace(memory)
	if memory == "" {
		return
	}
	anchor := b.persona.Anchor(b.cfg.Name + " (" + b.house + ")")
	lines := strings.SplitN(memory, "\n", 2)
	body := memory
	if strings.HasPrefix(lines[0], "I am ") {
		body = ""
		if len(lines) > 1 {
			body = strings.TrimSpace(lines[1])
		}
	}
	limit := b.league.cfg.MemoryChars
	if utf8.RuneCountInString(body) > limit {
		body = string([]rune(body)[:limit]) + "…"
	}
	b.mu.Lock()
	b.memory = anchor + "\n\n" + body
	b.saveMemory()
	b.mu.Unlock()
}

// chatView fetches the recent chat of the bot: the table and every private
// conversation that exists.
func (b *Bot) chatView(ctx context.Context) (string, []string) {
	g := b.league
	var out strings.Builder
	var ids []string
	names := g.houseNames()
	channels := []string{"public"}
	for _, slot := range g.slots {
		if slot.ID != b.house {
			channels = append(channels, slot.ID)
		}
	}
	for _, channel := range channels {
		limit := chatPrivateWindow
		title := "Private with " + channel + " " + names[channel]
		if channel == "public" {
			limit = chatPublicWindow
			title = "Public table"
		}
		messages, err := b.client.Messages(ctx, g.gameID, channel, limit)
		if err != nil {
			if channel == "public" {
				fmt.Fprintf(&out, "%s: (unreadable: %v)\n", title, err)
			}
			continue
		}
		if channel != "public" && len(messages) == 0 {
			continue
		}
		out.WriteString(renderChannel(title, messages, b.seen, b.house, names))
		for _, message := range messages {
			ids = append(ids, message.ID)
		}
	}
	if out.Len() == 0 {
		return "(no message)\n", nil
	}
	return out.String(), ids
}

// userPrompt assembles the situation of one call.
func (b *Bot) userPrompt(ctx context.Context, phase string, state map[string]any, report string) (string, []string) {
	b.mu.Lock()
	memory := b.memory
	history := strings.Join(b.history, "\n")
	notes := strings.Join(b.notes, "\n")
	b.notes = nil
	b.mu.Unlock()

	chat, ids := b.chatView(ctx)
	var p strings.Builder
	fmt.Fprintf(&p, "PHASE: %s\n\n", phase)
	p.WriteString("## Your private memory\n" + memory + "\n\n")
	if history != "" {
		p.WriteString("## What happened to you recently (written by the game master)\n" + history + "\n\n")
	}
	if report != "" {
		p.WriteString("## Report of the last resolved turn\n" + report + "\n\n")
	}
	p.WriteString("## Current state, as seen by house " + b.house + "\n" + renderState(state, b.house) + "\n")
	p.WriteString("## Chat (messages marked [NEW] arrived since your last decision)\n" + chat + "\n")
	if notes != "" {
		p.WriteString("## Notices\n" + notes + "\n\n")
	}
	switch {
	case strings.HasPrefix(phase, "negotiation"):
		p.WriteString("## Your task\nThis is a negotiation round. Read the chat, answer proposals, open the talks your plan needs. Send messages only: orders sent now are ignored. The other houses answer before the order round. Update your memory with deals and impressions. Answer with the JSON object.")
	default:
		season := asString(state["season"])
		if season == "winter" {
			p.WriteString("## Your task\nIt is winter: write your winter investment lines (orders.winter) and any card orders (orders.special). You may send last messages. Answer with the JSON object.")
		} else {
			p.WriteString("## Your task\nWrite this season's orders: one chain per noble you want to use (orders.chains), card orders in orders.special. Re-read the newest chat messages first: a counter-offer may change your plan. You may send messages. Answer with the JSON object.")
		}
	}
	return p.String(), ids
}

// ask runs one model call, retrying once when the answer is not JSON.
func (b *Bot) ask(ctx context.Context, conversation []Message, label string) (Decision, string, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		answer, usage, err := b.llm.Complete(ctx, conversation)
		b.mu.Lock()
		b.usage.add(usage)
		b.callSeq++
		seq := b.callSeq
		b.mu.Unlock()
		b.logCall(seq, label, conversation, answer, err)
		if err != nil {
			return Decision{}, "", err
		}
		decision, parseErr := parseDecision(answer)
		if parseErr == nil {
			return decision, answer, nil
		}
		lastErr = parseErr
		conversation = append(conversation,
			Message{Role: "assistant", Content: answer},
			Message{Role: "user", Content: "Your answer could not be read (" + parseErr.Error() + "). Answer again with ONE JSON object only, no prose, no code fence."})
	}
	return Decision{}, "", lastErr
}

func (b *Bot) logCall(seq int, label string, conversation []Message, answer string, callErr error) {
	dir := b.path("calls")
	_ = os.MkdirAll(dir, 0o755)
	entry := map[string]any{"label": label, "at": time.Now().Format(time.RFC3339), "answer": answer}
	if callErr != nil {
		entry["error"] = callErr.Error()
	}
	// The system prompt is identical on every call: keep only the dialogue.
	if len(conversation) > 1 {
		entry["prompt"] = conversation[1:]
	}
	raw, _ := json.MarshalIndent(entry, "", " ")
	_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%03d-%s.json", seq, label)), raw, 0o600)
}

func (b *Bot) markSeen(ids []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	changed := false
	for _, id := range ids {
		if !b.seen[id] {
			b.seen[id] = true
			changed = true
		}
	}
	if changed {
		b.saveSeen()
	}
}

// Negotiate runs a message-only round.
func (b *Bot) Negotiate(ctx context.Context, round, rounds int, state map[string]any, report string) error {
	phase := fmt.Sprintf("negotiation %d/%d", round, rounds)
	prompt, ids := b.userPrompt(ctx, phase, state, report)
	label := fmt.Sprintf("t%02d-%s-n%d", asInt(state["turn"]), asString(state["season"]), round)
	decision, _, err := b.ask(ctx, []Message{{Role: "system", Content: b.system}, {Role: "user", Content: prompt}}, label)
	if err != nil {
		return err
	}
	b.markSeen(ids)
	b.updateMemory(decision.Memory)
	b.send(ctx, decision.Messages)
	return nil
}

// PlayOrders runs the order round and submits, repairing rejected orders.
// It returns the one-line summary of what was submitted.
func (b *Bot) PlayOrders(ctx context.Context, state map[string]any, report string) (string, error) {
	g := b.league
	prompt, ids := b.userPrompt(ctx, "orders", state, report)
	label := fmt.Sprintf("t%02d-%s-orders", asInt(state["turn"]), asString(state["season"]))
	conversation := []Message{{Role: "system", Content: b.system}, {Role: "user", Content: prompt}}
	decision, answer, err := b.ask(ctx, conversation, label)
	if err != nil {
		return "", err
	}
	b.markSeen(ids)
	b.updateMemory(decision.Memory)

	for attempt := 0; ; attempt++ {
		submitErr := b.client.Submit(ctx, g.gameID, decision.Orders.Submission())
		if submitErr == nil {
			// Chat throttling must not delay the submission.
			b.send(ctx, decision.Messages)
			return summarizeOrders(decision.Orders), nil
		}
		var apiErr *APIError
		if !errors.As(submitErr, &apiErr) || apiErr.Status != 400 || attempt >= g.cfg.MaxRepairs {
			return "", submitErr
		}
		conversation = append(conversation,
			Message{Role: "assistant", Content: answer},
			Message{Role: "user", Content: repairPrompt(apiErr)})
		repaired, repairedAnswer, askErr := b.ask(ctx, conversation, fmt.Sprintf("%s-fix%d", label, attempt+1))
		if askErr != nil {
			return "", askErr
		}
		decision.Orders = repaired.Orders
		answer = repairedAnswer
	}
}

// Pass submits empty orders so the turn is not blocked by a failing bot.
func (b *Bot) Pass(ctx context.Context) error {
	err := b.client.Submit(ctx, b.league.gameID, Orders{})
	return err
}

func repairPrompt(apiErr *APIError) string {
	var p strings.Builder
	p.WriteString("The server rejected your orders: " + apiErr.Message + "\n")
	for _, issue := range apiErr.Details {
		fmt.Fprintf(&p, "- noble %s line %d (%s): %s\n", issue.Noble, issue.Line, issue.Code, issue.Message)
	}
	p.WriteString("Correct them and answer again with the full JSON object (orders replaced by the corrected set; leave messages empty).")
	return p.String()
}

func summarizeOrders(orders OrdersDecision) string {
	var parts []string
	for _, chain := range orders.Chains {
		parts = append(parts, strings.ReplaceAll(chain.Text, "\n", " ; "))
	}
	for _, winter := range orders.Winter {
		parts = append(parts, "winter: "+strings.ReplaceAll(winter.Lines, "\n", " ; "))
	}
	for _, special := range orders.Special {
		parts = append(parts, "card: "+special.Text)
	}
	if len(parts) == 0 {
		return "no orders"
	}
	return strings.Join(parts, " | ")
}

// send posts the messages of a decision, respecting the chat rate limit.
func (b *Bot) send(ctx context.Context, messages []OutMessage) {
	g := b.league
	public := 0
	sent := 0
	for _, message := range messages {
		body := strings.TrimSpace(message.Body)
		to := normalizeRecipient(message.To)
		if body == "" || to == "" || sent >= maxMessagesPerCall {
			continue
		}
		if strings.EqualFold(to, "public") || strings.EqualFold(to, "table") {
			to = "public"
			if public++; public > 1 {
				continue
			}
		}
		if to == b.house {
			continue
		}
		if utf8.RuneCountInString(body) > 500 {
			body = string([]rune(body)[:499]) + "…"
		}
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			if err = b.client.PostMessage(ctx, g.gameID, to, body); !isAPIError(err, "rate_limited") {
				break
			}
			if sleepCtx(ctx, 1200*time.Millisecond) != nil {
				return
			}
		}
		if err != nil {
			b.mu.Lock()
			b.notes = append(b.notes, fmt.Sprintf("Your message to %s was not delivered: %v", to, err))
			b.mu.Unlock()
			g.logf("[%s] message to %s failed: %v", b.cfg.Name, to, err)
			continue
		}
		sent++
		if sleepCtx(ctx, 1100*time.Millisecond) != nil {
			return
		}
	}
}

// sleepCtx waits for d or until the context ends.
func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

var houseID = regexp.MustCompile(`^(?i)p(\d+)\b`)

// normalizeRecipient reduces what a model wrote in "to" ("P2 Maison Bleue",
// "p2") to the house id the chat expects ("P2"); "public" and unknown values
// are left as they are.
func normalizeRecipient(to string) string {
	to = strings.TrimSpace(to)
	if match := houseID.FindStringSubmatch(to); match != nil {
		return "P" + match[1]
	}
	return to
}
