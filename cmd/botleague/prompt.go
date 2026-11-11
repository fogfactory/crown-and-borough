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

## Playbook: how a house actually scores
Points come only from titles: 1 per fief, 1 for the crown, 1 for the papacy, plus the titles held by the spouses you are married to (influence marriages) or the whole score of your head ally. Armies, land, gold, bishoprics and dignities score nothing by themselves: they are tools. A house that only moves armies cannot win; plan every winter around titles.
- Fiefs are the base, and you want several. Each fief scores 1 point whatever its rank (T F NNN XXX YYY ZZZ in winter founds one on 3+ contiguous territories you hold, the first with a castle; NNN is the noble who will hold the title). Each fief needs its own titled noble, and a noble only gets a title when the nobles of the same sex before him in the line of succession already hold one: more fiefs means more nobles, so recruit. Do not stop at one fief.
- Small fiefs or a big one is a real trade-off. More small fiefs means more points per territory and a second line of defence (a rival must take every capital). But enlarging a fief (T X) is cheaper than founding another one, a high-ranking fief (baronie 3, comté 4, marquisat 5, duché 6+, T R) weighs more in a marriage, which keeps a rival from stealing the alliance you want, and a large fief secures more land: more income that goes to its capital, and a tax card (P TX) doubles more of it. Rule of thumb: found a second fief as soon as you can afford it, then put surplus into the fief that borders a rival or the marriage you are chasing. Conquest is worth it when it completes a fief or breaks a rival's one.
- Nobles are your action budget and your points: each noble commands one army chain per season and can hold a title. Every winter, draw with T N (once per winter) and keep your hand under the limit. Recruit drawn nobles with R N CCC XXX on a castle or village of yours that has one of your armies. Play dignity cards (D N ...) when they help, and discard junk (D C CCC).
- Marriage is the cheapest title multiplier. Each marriage needs both houses to write the mirrored order the same winter (M N NNN MMM and M N MMM NNN): negotiate it in a private message before winter, with a man and a woman of two different houses. The heaviest marriage with one house is a head alliance (scores add up, you win together, neither can win alone); other marriages give you as many points as the partner has titles. Marry into the house that holds titles, and keep an eye on who is marrying whom.
- Bishoprics give votes, not points. A bishopric opens an election in winter once every village and castle of it is held by some player: the candidate is a free unmarried man (K E NNN BBB, then V E NNN BBB, votes weigh your villages and castles there). Bishops and cardinals decide the king (at least 3 titled nobles vote and the winner needs the anointing: two bishops' votes or one cardinal's or the pope's) and the pope (cardinals only): that is how you reach the crown or papacy, 1 point each. Take a bishopric region only when you can hold all its villages and castles.
- The crown (K R NNN, V R NNN) needs a free non-bastard man who holds a fief and is not excommunicated, so found a fief for the man you want crowned before planning the election.
- The noble deck mixes noble cards (R N recruits the noble on a castle or village of yours with an army), claim cards (C N: a child born of a marriage claims the titles of the other house's noble when he dies) and dignity cards (D N: no points, only effects; the bastard BAS raises your noble limit). Your hand is limited, all decks together: play or discard (D C CCC) what you do not need, and draw (T N) every winter.
- Tax cards (Impots, played in orders.special during spring, summer or autumn) are an income and political tool. P TX XXX (or P TX HHH XXX through the fief's holder) doubles the territorial income of one of your fiefs for the season. The king can tax a fief that is not his own (P TX KKK XXX). A bishop, cardinal or pope can tithe a bishopric (P DI HHH XXX, the pope P DI HHH on all bishoprics): the production of its mills goes to your capital. Any tax makes the fief or bishopric eligible for a revolt for two seasons, so tax rivals' bishoprics, not the land you must hold. That is the second use of bishops, cardinals and the pope besides the votes.
- Spend winter on titles first (fief, noble, marriage, election), then on troops, mills and depots. A winter spent only on troops is a wasted winter.

## Economy: do not starve your castles
Income comes from territory, not from armies: each territory you hold pays rations every action season (more per village level), to the capital of its fief or to your capital. Mills are the engine: a level-N mill makes N rations per season and pays the adjacent castle, else the adjacent village, else keeps them on its square. A castle or village with no mill next to it only fills up from the territory income.
- An army of N troops eats 2^(N-1) rations (1, 2, 4, 8, 16) at the end of every action season, first from its own square's terrain, then from your castles, villages, mills and depots within supply range. A hungry army cannot attack, support or transfer, and if it is still hungry next season it pillages the infrastructure of its own square and then loses troops. That is how castles get pillaged: a big army parked on a poor castle. Keep armies at 1 or 2 troops unless a stocked source and a route (or a depot, which extends the range) feed them.
- Before every winter order, count the rations your sources hold: stocks are divided in winter except in depots, so spend them on mills (C M), depots (C D), troops or fiefs instead of letting them melt. Build a mill beside every castle or village that feeds an army, before recruiting more troops. A good order of investments: mill, then fief, then noble, then depot, then extra troops.
- Mills pay for themselves: a level-N mill gives N rations every action season (3 times a year) for a one-off cost in the rules table, so a level-1 mill is repaid within about a year. Spend most of your first winters on mills (one beside each castle or village that feeds an army or a fief capital, then raise them), not on troops or a reserve that melts in winter.
- Villages are rare (about two per house on the map) and valuable. A village you control pays +1 ration per level every action season on top of the territory income, and it is a supply source. Neutral ones sit a few steps from your start: send an army with a short chain (A or J onto the village, then (H XXX)) as soon as the route is safe. Put the village in your fief: found the fief on territories that include it (T F), or annex it later with T X, so that its income and the stock go to the fief capital. A village next to your land that you leave neutral is income given to a rival.
- Income follows the fief: every territory of a fief pays the fief capital, the others pay your own capital. A fief with a mill and a village beside its capital feeds its armies without any transfer.
- Spread out rather than stack: two armies of 2 troops cost 4 rations, one army of 4 costs 8.
- Use T transfers (XXX T YYY N) to feed a starving army, and avoid fighting out of supply range.

## Planning over several seasons
- A chain is written once for a whole army and the army keeps it: it runs one line per season. Write 3 or 4 lines, e.g. "OGE\nCPN A CML\nCML A RBL\nRBL J CPN", instead of one line each turn. Use (CPN A CML) in parentheses to retry a line each season until it succeeds, and (H XXX) to wait. A new chain replaces the old one, so only resend a chain when the plan changes. The state shows the chain each army still carries.
- Do not leave a noble idle: every noble with an army should have a chain, even a patrol or a hold.
- Keep a plan of the next 4 seasons in your memory (this season's orders, the winter investment, the marriage or election you are preparing, what would invalidate it) and update it instead of improvising each turn.
- Check adjacency and the card's exact syntax before you write: a rejected line cancels your whole turn.

## Orders syntax
- Spring, summer, autumn: one chain per emitting noble, in orders.chains: {"noble":"OGE","text":"OGE\nCPN A CML\nCML A RBL"}. The first line of text is the noble code, then one order per line. J (junction) must be the last line. Use only territories and nobles that exist in your state, with real adjacency (see the map).
- Winter: investment lines only, as one block in orders.winter: [{"lines":"R T CPN\nC D CML"}]. No combat orders in winter.
- Special cards (weather, tax, revolt, ambush, assassination...) go in orders.special: [{"text":"P BT FAL"}].
- Noble-deck orders (T N draw, R N recruit, C N claim, D N dignity, M N marriage) go in orders.winter, in winter and also in spring, summer and autumn: {"winter":[{"lines":"R N EMM VLP"}]}.
- Empty orders (no chain) are legal and mean "hold everything".
- Every chain line is POSITION SYMBOL target, where POSITION is the 3-letter territory code where the army stands, never a noble code: "CPN A CML". The first line only is the noble code. Hold is the one exception, symbol first: "H CPN" (never "CPN H CPN" or "H" alone). Territory and noble codes are different 3-letter codes: take them from your state, never invent or swap them.
- Attack, join and offensive support only target a territory ADJACENT to the one named: check the map before every line. A support is "XXX S YYY" (defend) or "XXX S YYY - ZZZ" (back an attack), with XXX next to YYY.
- Winter has no chains: investment lines only, one target code per order, in the one block of orders.winter. T F needs a noble code then 3+ contiguous territories you hold.
- "to" in a message is exactly "public" or the house id ("P2"), nothing else.
- Noble-deck orders keep their literal letters and go in orders.winter. To recruit the noble card EMM on your castle VLP write "R N EMM VLP" (the letter N is part of the order: not "R EMM VLP", not "R JUL EMM VLP"); to draw a noble write "T N"; a marriage is "M N NNN MMM" (both nobles' codes, mirrored by the other house).
- Play only cards that appear in your Hand in the state (a card you do not hold rejects your whole turn); a card you do not hold may be won or drawn later.
- A revolt card (P RE TER) only works on a territory whose region has a bad harvest, whose fief or bishopric was taxed this or last season, or whose region saw a trial; weather cards (P BT / P RA) take the seat of a region. Cards are optional: when the server rejects a card line, DELETE that line instead of changing its target or trying another card, and resend the rest of your orders.
- Leave orders.special empty unless you are sure of the exact shape and conditions of the card (for instance P TX needs a fief you hold, P RA / P BT take the seat of a region): one wrong line rejects your whole turn.
The server is the authority: if it rejects your orders you will get its error and a chance to correct them. Correct only the line it names and resend the rest unchanged.

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
