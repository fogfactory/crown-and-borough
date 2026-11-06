package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Decision is what a model answers: private reasoning, chat messages, the
// orders of the phase and the new private memory.
type Decision struct {
	Thoughts string         `json:"thoughts,omitempty"`
	Messages []OutMessage   `json:"messages,omitempty"`
	Orders   OrdersDecision `json:"orders"`
	Memory   string         `json:"memory,omitempty"`
}

// OutMessage is a chat message to send: To is "public" or a house id.
type OutMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

// OrdersDecision is lenient about the shape models give orders.
type OrdersDecision struct {
	Chains  []ChainOrder  `json:"chains,omitempty"`
	Winter  []WinterOrder `json:"winter,omitempty"`
	Special []DeckOrder   `json:"special,omitempty"`
}

func (o OrdersDecision) Submission() Orders {
	return Orders{Chains: o.Chains, Winter: o.Winter, Special: o.Special}
}

func (o *OrdersDecision) UnmarshalJSON(data []byte) error {
	var raw struct {
		Chains  json.RawMessage `json:"chains"`
		Winter  json.RawMessage `json:"winter"`
		Special json.RawMessage `json:"special"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	chains, err := lenientList(raw.Chains)
	if err != nil {
		return fmt.Errorf("orders.chains: %w", err)
	}
	for _, item := range chains {
		var chain ChainOrder
		switch value := item.(type) {
		case map[string]any:
			chain.Noble, _ = value["noble"].(string)
			chain.Text = firstString(value, "text", "lines", "orders")
		case string:
			chain.Text = value
		}
		chain = normalizeChain(chain)
		if chain.Noble != "" && chain.Text != "" {
			o.Chains = append(o.Chains, chain)
		}
	}
	for _, item := range mustList(raw.Winter) {
		switch value := item.(type) {
		case map[string]any:
			if lines := firstString(value, "lines", "text"); lines != "" {
				o.Winter = append(o.Winter, WinterOrder{Lines: lines})
			}
		case string:
			o.Winter = append(o.Winter, WinterOrder{Lines: value})
		}
	}
	o.Winter = mergeWinter(o.Winter)
	for _, item := range mustList(raw.Special) {
		switch value := item.(type) {
		case map[string]any:
			if text := firstString(value, "text", "lines"); text != "" {
				o.Special = append(o.Special, DeckOrder{Text: text})
			}
		case string:
			o.Special = append(o.Special, DeckOrder{Text: value})
		}
	}
	return nil
}

// mergeWinter keeps the winter sheet in one block: a model that answers one
// line per entry still gives one sheet.
func mergeWinter(list []WinterOrder) []WinterOrder {
	if len(list) <= 1 {
		return list
	}
	lines := make([]string, 0, len(list))
	for _, entry := range list {
		lines = append(lines, strings.TrimSpace(entry.Lines))
	}
	return []WinterOrder{{Lines: strings.Join(lines, "\n")}}
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		switch value := values[key].(type) {
		case string:
			return value
		case []any:
			parts := make([]string, 0, len(value))
			for _, part := range value {
				if text, ok := part.(string); ok {
					parts = append(parts, text)
				}
			}
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func lenientList(raw json.RawMessage) ([]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var list []any
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var single any
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, err
	}
	return []any{single}, nil
}

func mustList(raw json.RawMessage) []any {
	list, _ := lenientList(raw)
	return list
}

var noblePrefix = regexp.MustCompile(`^[A-Z]{3}$`)

// positionHold matches a hold written like an attack ("PVL H PVL"): the rules
// put the symbol first ("H PVL").
var positionHold = regexp.MustCompile(`^(\(?)[A-Z]{3} H ([A-Z]{3})(\)?)$`)

// normalizeChain makes sure the text opens with the emitting noble's code,
// the way the server reads a chain.
func normalizeChain(chain ChainOrder) ChainOrder {
	chain.Noble = strings.ToUpper(strings.TrimSpace(chain.Noble))
	lines := make([]string, 0)
	for _, line := range strings.Split(strings.ReplaceAll(chain.Text, "\r", ""), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, positionHold.ReplaceAllString(line, "${1}H ${2}${3}"))
		}
	}
	if len(lines) == 0 {
		return chain
	}
	if noblePrefix.MatchString(lines[0]) {
		if chain.Noble == "" {
			chain.Noble = lines[0]
		}
	} else if chain.Noble != "" {
		lines = append([]string{chain.Noble}, lines...)
	}
	chain.Text = strings.Join(lines, "\n")
	return chain
}

var thinkBlock = regexp.MustCompile(`(?s)<think>.*?</think>`)

// parseDecision pulls the JSON object out of a model answer, which may carry
// reasoning blocks, markdown fences or prose around it.
func parseDecision(answer string) (Decision, error) {
	answer = thinkBlock.ReplaceAllString(answer, "")
	var lastErr error
	for _, candidate := range jsonCandidates(answer) {
		var decision Decision
		err := json.Unmarshal([]byte(candidate), &decision)
		if err != nil {
			// Models often leave raw line breaks inside strings (orders, memory).
			decision = Decision{}
			if retry := json.Unmarshal([]byte(escapeStringControls(candidate)), &decision); retry == nil {
				return decision, nil
			}
			lastErr = err
			continue
		}
		return decision, nil
	}
	if lastErr != nil {
		return Decision{}, fmt.Errorf("the answer is not valid JSON: %w", lastErr)
	}
	return Decision{}, errors.New("the answer holds no JSON object")
}

// escapeStringControls escapes the raw control characters (line breaks, tabs)
// found inside JSON strings, which the format forbids.
func escapeStringControls(text string) string {
	var out strings.Builder
	inString := false
	for index := 0; index < len(text); index++ {
		character := text[index]
		switch {
		case inString && character == '\\' && index+1 < len(text):
			out.WriteByte(character)
			index++
			out.WriteByte(text[index])
			continue
		case character == '"':
			inString = !inString
		case inString && character == '\n':
			out.WriteString(`\n`)
			continue
		case inString && character == '\r':
			continue
		case inString && character == '\t':
			out.WriteString(`\t`)
			continue
		}
		out.WriteByte(character)
	}
	return out.String()
}

// jsonCandidates lists the balanced {...} objects of a text, longest first.
func jsonCandidates(text string) []string {
	var found []string
	for start := 0; start < len(text); start++ {
		if text[start] != '{' {
			continue
		}
		if end := matchBrace(text, start); end > start {
			found = append(found, text[start:end+1])
			start = end
		}
	}
	// Prefer the largest object: a decision nests orders and messages.
	for left := 1; left < len(found); left++ {
		for right := left; right > 0 && len(found[right]) > len(found[right-1]); right-- {
			found[right], found[right-1] = found[right-1], found[right]
		}
	}
	return found
}

func matchBrace(text string, start int) int {
	depth := 0
	inString := false
	for index := start; index < len(text); index++ {
		character := text[index]
		switch {
		case inString:
			if character == '\\' {
				index++
			} else if character == '"' {
				inString = false
			}
		case character == '"':
			inString = true
		case character == '{':
			depth++
		case character == '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}
