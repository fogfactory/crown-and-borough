package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// systemPrompt is the stable part of every call of a bot: its role, its
// persona, the protocol, the output format, the map and the rules.
func systemPrompt(house, houseName string, persona Persona, players []PlayerSlot, mapData MapData, rules string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are %s, house %s in **Crown & Borough**, an asynchronous turn-based strategy game on a map. You are one player, not an assistant and not a moderator. Maximize your own chance to win. You may bluff, conceal plans, make conditional promises and break a deal when your persona makes that rational. You may only use the information available to your house.

`, houseName, house)
	b.WriteString(persona.Brief())
	b.WriteString("\n## Table\n")
	for _, slot := range players {
		marker := ""
		if slot.ID == house {
			marker = " (you)"
		}
		fmt.Fprintf(&b, "- %s: %s%s\n", slot.ID, slot.Name, marker)
	}
	b.WriteString(`
Everyone at the table is a language-model agent with its own persona. Nobody can be trusted by default. A human observer may read every private conversation.

## Diplomacy
You talk through the in-game chat: "public" is the table, a house id ("P2") opens a private conversation with that house.
- A proposal states its scope and expiry (territories, turn, what you expect in return). Vague friendship is not a pact.
- Answer a proposal with accept, counter, reject or delay, not only a narration of your intent.
- Keep messages short, in character, 500 characters at most, in the language of the table (French by default). Never paste raw state, JSON, your memory or your exact order list.
- At most one public message per turn unless a public warning is useful. You may omit, distort or delay information.
- A promise is a strategic input, not an obligation: follow your persona's trust and betrayal rules, and record every deal and its expiry in your memory.

## Priorities each turn
1. Can famine, a forced retreat or an exposed capital eliminate you next turn?
2. Which army or source is the most valuable safe target, counting support, castle defense, noble command and likely counter-support?
3. Does every army stay supplied after the move? An army of N troops needs 2^(N-1) rations. Action orders execute one line per army per season.
4. Which deal changes the board rather than merely exchanging information?
5. Which winter investment improves survival or creates a decisive advantage? You win with titles (fiefs, crown, papacy), not with land or gold.
Do not attack just because an enemy is adjacent. Do not build a large army without a source or depot. Keep a retreat and a route to your capital.

## Orders syntax
- Spring, summer, autumn: one chain per emitting noble, in orders.chains: {"noble":"OGE","text":"OGE\nCPN A CML\nCML A RBL"}. The first line of text is the noble code, then one order per line. J (junction) must be the last line. Use only territories and nobles that exist in your state, with real adjacency (see the map).
- Winter: investment lines only, as one block in orders.winter: [{"lines":"R T CPN\nC D CML"}]. No combat orders in winter.
- Card orders (special cards, noble cards, assassination...) go in orders.special: [{"text":"P BT FAL"}].
- Empty orders (no chain) are legal and mean "hold everything".
The server is the authority: if it rejects your orders you will get its error and a chance to correct them.

## Answer format
Answer with ONE JSON object and nothing else (no prose, no markdown fence):
{
  "thoughts": "2-4 sentences of private reasoning",
  "messages": [{"to": "public", "body": "..."}, {"to": "P2", "body": "..."}],
  "orders": {"chains": [], "winter": [], "special": []},
  "memory": "your complete updated private memory (replaces the previous one)"
}
- "messages" and "orders" may be empty. "memory" is private; keep it under 3000 characters. Its first line is your identity anchor: keep it unchanged.
- Keep in memory: active pacts (peer, scope, expiry, break condition), a note per rival, suspicions, lessons from the last outcomes and your plan for the next turn with what would invalidate it. Replace guesses with observed outcomes.
`)
	b.WriteString("\n## Map (static)\nTerritory (terrain, V = village): adjacent territories [impassable links].\n")
	b.WriteString(renderMap(mapData))
	b.WriteString("\n## Official player rules\n")
	b.WriteString(rules)
	return b.String()
}

func renderMap(data MapData) string {
	var b strings.Builder
	for _, territory := range data.Territories {
		flag := ""
		if territory.Village {
			flag = " V"
		}
		fmt.Fprintf(&b, "%s (%s%s): %s", territory.ID, territory.Terrain, flag, strings.Join(territory.Adjacencies, " "))
		if len(territory.Impassable) > 0 {
			fmt.Fprintf(&b, " [no passage: %s]", strings.Join(territory.Impassable, " "))
		}
		b.WriteString("\n")
	}
	if len(data.Regions) > 0 {
		b.WriteString("Bishoprics (regions): ")
		parts := make([]string, 0, len(data.Regions))
		for _, region := range data.Regions {
			parts = append(parts, fmt.Sprintf("%s=%s", region.ID, region.Name))
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n")
	}
	return b.String()
}

// compactRules keeps the sections a bot needs on every turn.
func compactRules(rules string) string {
	keep := []string{"L'essentiel", "1. Une année", "2. Commander", "4. Nourrir", "5. L'hiver", "9. Fin de partie", "Annexe A", "Annexe B"}
	var out []string
	include := false
	for _, line := range strings.Split(rules, "\n") {
		if strings.HasPrefix(line, "## ") {
			include = false
			for _, prefix := range keep {
				if strings.Contains(line, prefix) {
					include = true
				}
			}
		}
		if include || strings.HasPrefix(line, "# ") {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// prune drops the noise of a JSON value: nulls, empty containers, false and
// zero numbers, and the keys listed in drop.
func prune(value any, drop map[string]bool) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if drop[key] {
				continue
			}
			if pruned := prune(item, drop); pruned != nil {
				out[key] = pruned
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			if pruned := prune(item, drop); pruned != nil {
				out = append(out, pruned)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case nil:
		return nil
	case bool:
		if !typed {
			return nil
		}
	case float64:
		if typed == 0 {
			return nil
		}
	case string:
		if typed == "" {
			return nil
		}
	}
	return value
}

var stateNoise = map[string]bool{
	"arms": true, "color": true, "projectedMillIncome": true, "yearCount": true,
	"nobleDeckSize": true, "nobleDiscardSize": true, "handLimit": true, "revision": true,
}

func compactJSON(value any, drop map[string]bool, limit int) string {
	pruned := prune(value, drop)
	if pruned == nil {
		return "none"
	}
	raw, _ := json.Marshal(pruned)
	text := string(raw)
	if limit > 0 && len(text) > limit {
		text = text[:limit] + "…(truncated)"
	}
	return text
}

func asMap(value any) map[string]any {
	typed, _ := value.(map[string]any)
	return typed
}

func asList(value any) []any {
	typed, _ := value.([]any)
	return typed
}

func asString(value any) string {
	typed, _ := value.(string)
	return typed
}

func asInt(value any) int {
	typed, _ := value.(float64)
	return int(typed)
}

// renderState writes a player's view of the game as compact text.
func renderState(state map[string]any, house string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Turn %d · year %d · Season: %s\n", asInt(state["turn"]), asInt(state["year"]), asString(state["season"]))
	if scores := asMap(state["scores"]); scores != nil {
		ids := sortedKeys(scores)
		parts := make([]string, 0, len(ids))
		for _, id := range ids {
			parts = append(parts, fmt.Sprintf("%s=%d", id, asInt(asMap(scores[id])["total"])))
		}
		fmt.Fprintf(&b, "Influence: %s\n", strings.Join(parts, " "))
	}
	if victory := asMap(state["victory"]); victory != nil {
		fmt.Fprintf(&b, "Victory thresholds: %s\n", compactJSON(victory, nil, 600))
	}

	b.WriteString("\n### Houses\n")
	for _, item := range asList(state["players"]) {
		player := asMap(item)
		line := fmt.Sprintf("- %s %s", asString(player["id"]), asString(player["name"]))
		if capital := asString(player["capitalTerritory"]); capital != "" {
			line += " capital " + capital
		}
		if income := asInt(player["projectedIncome"]); income != 0 {
			line += fmt.Sprintf(" income+%d", income)
		}
		if upkeep := asInt(player["projectedConsumption"]); upkeep != 0 {
			line += fmt.Sprintf(" upkeep-%d", upkeep)
		}
		if risk := asList(player["armiesAtRisk"]); len(risk) > 0 {
			line += " AT RISK " + compactJSON(risk, nil, 400)
		}
		if succession := asList(player["succession"]); len(succession) > 0 {
			codes := make([]string, 0, len(succession))
			for _, code := range succession {
				codes = append(codes, asString(code))
			}
			line += " heirs " + strings.Join(codes, ">")
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n### Territories (stock = rations kept there)\n")
	var empty []string
	for _, item := range asList(state["territories"]) {
		territory := asMap(item)
		id := asString(territory["id"])
		owner := asString(territory["owner"])
		army := asMap(territory["army"])
		infra := asList(territory["infrastructures"])
		stock := asInt(territory["resources"])
		if owner == "" && army == nil && len(infra) == 0 && stock == 0 {
			empty = append(empty, id)
			continue
		}
		line := id
		if owner != "" {
			line += " " + owner
		} else {
			line += " neutral"
		}
		if stock > 0 {
			line += fmt.Sprintf(" stock%d", stock)
		}
		if army != nil {
			line += fmt.Sprintf(" army%d(%s)", asInt(army["size"]), asString(army["owner"]))
			if army["starving"] == true {
				line += " STARVING"
			}
			if chain := asMap(army["chain"]); chain != nil {
				line += " chain:" + compactJSON(chain, nil, 200)
			}
		}
		for _, entry := range infra {
			infraMap := asMap(entry)
			line += fmt.Sprintf(" %s%d", asString(infraMap["type"]), asInt(infraMap["level"]))
			if infraMap["pillaged"] == true {
				line += "(pillaged)"
			}
		}
		if income := asInt(territory["projectedIncome"]); income > 0 && owner == house {
			line += fmt.Sprintf(" income%d→%s", income, asString(territory["incomeDestination"]))
		}
		if mill := asInt(territory["millProduction"]); mill > 0 && owner == house {
			line += fmt.Sprintf(" mill+%d→%s", mill, asString(territory["millDestination"]))
		}
		b.WriteString(line + "\n")
	}
	if len(empty) > 0 {
		fmt.Fprintf(&b, "Neutral and empty: %s\n", strings.Join(empty, " "))
	}

	b.WriteString("\n### Nobles\n")
	var mine []string
	for _, item := range asList(state["nobles"]) {
		noble := asMap(item)
		line := fmt.Sprintf("- %s at %s (%s) — %s, house %s", asString(noble["code"]), asString(noble["location"]), asString(noble["status"]), asString(noble["name"]), asString(noble["owner"]))
		if spouse := asString(noble["spouse"]); spouse != "" {
			line += " married to " + spouse
		}
		if title := asString(noble["religiousTitle"]); title != "" {
			line += " " + title
		}
		if dignities := asList(noble["dignities"]); len(dignities) > 0 {
			line += " " + compactJSON(dignities, nil, 120)
		}
		if asString(noble["owner"]) == house {
			mine = append(mine, line)
		} else {
			b.WriteString(line + "\n")
		}
	}
	if len(mine) > 0 {
		b.WriteString("Yours:\n" + strings.Join(mine, "\n") + "\n")
	}

	rest := map[string]any{}
	for key, value := range state {
		switch key {
		case "turn", "year", "season", "scores", "victory", "players", "territories", "nobles", "yearCount", "revision":
		default:
			rest[key] = value
		}
	}
	drop := map[string]bool{"territories": true}
	for key := range stateNoise {
		drop[key] = true
	}
	if len(rest) > 0 {
		b.WriteString("\n### Other state (JSON, empty fields omitted)\n")
		keys := sortedKeys(rest)
		for _, key := range keys {
			if text := compactJSON(rest[key], drop, 2500); text != "none" {
				fmt.Fprintf(&b, "%s: %s\n", key, text)
			}
		}
	}
	return b.String()
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// renderReport shortens a turn report to what is worth remembering.
func renderReport(report map[string]any) string {
	if report == nil {
		return "(no report yet)"
	}
	drop := map[string]bool{"arms": true, "color": true, "name": true, "infrastructures": true, "nobles": true}
	return compactJSON(report, drop, 7000)
}

// renderChannel prints the recent messages of a channel, flagging the ones
// the bot has not seen yet.
func renderChannel(title string, messages []ChatMessage, seen map[string]bool, house string, names map[string]string) string {
	if len(messages) == 0 {
		return fmt.Sprintf("%s: (no message yet)\n", title)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s:\n", title)
	for _, message := range messages {
		marker := ""
		if !seen[message.ID] && message.Author != house {
			marker = " [NEW]"
		}
		author := message.Author
		if message.Author == house {
			author += " (you)"
		} else if name := names[message.Author]; name != "" {
			author += " " + name
		}
		fmt.Fprintf(&b, "  [t%d %s] %s%s: %s\n", message.Turn.Number, message.Turn.Season, author, marker, strings.ReplaceAll(message.Body, "\n", " "))
	}
	return b.String()
}
