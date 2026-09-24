# Game Rules — Crown & Borough

**Friendly version (human players).** This document describes the rules currently
active on the game server. Numeric values come from `assets/balance.yaml`, which
remains the source of playable numbers; when documents disagree, the engine wins.

---

## 1. The Pitch

Crown & Borough is a turn-based medieval strategy game played on a map of
territories connected by a graph. Players submit their orders in secret; the
engine resolves everyone **at the same time**, season after season.

One constraint shapes everything else in these rules: **orders don't come from
an army, they come from a noble**, and each player only has a handful of them.
A free or hostage noble emits only **one chain per turn** — a chain being a
sequence of orders written ahead of time for an entire army. Since you can't
reprogram every army every turn, you have to plan ahead: decide today what an
army will do two or three turns from now, and live with the uncertainty of what
everyone else is doing in the meantime. That's where all the vocabulary of
"chain," "liaison" (single/loop), and "reception" detailed in section 4 comes
from: it isn't technical decoration, it's the direct consequence of noble
scarcity.

The two pillars of tension in the game:

- simultaneous resolution of intentions, supports, and combats — nobody sees
  what anyone else wrote before resolution;
- exponential logistics — large troop concentrations are expensive to feed and
  become vulnerable the moment their supply is cut (section 7).

An online game accepts **2 to 8 players** (up to 16 in a local game). The map
and numerical data (owners, army sizes, stocks, infrastructure, and noble
positions) are visible to everyone at all times. What stays private are
**intentions**: online, a player only sees the exact detail of a chain or a
combat when they take part in it — section 3 shows this on a worked example.

Each player starts on a distinct territory, where a **castle** is built for
free (their **capital**), with {{starting_resources}} R in stock, an army of
{{starting_troops}} troops, and {{starting_nobles}} free noble(s).

A game lasts a number of years chosen at creation (10 by default); section 10
covers the end of the game and scoring.

### Inspirations

The project draws in particular on [Fief](https://boardgamegeek.com/boardgame/107704/fief)
for its feudal setting and territorial stakes, and on
[Diplomacy](https://boardgamegeek.com/boardgame/483/diplomacy) for simultaneous
order programming, supports, and conflict resolution. These games are design
inspirations, not sources of rules applicable to Crown & Borough.

---

## 2. The Turn at a Glance

A year has **four turns**: spring, summer, autumn, and **winter**. The turn
counter increases by one at every season, including winter.

The three action seasons (spring, summer, autumn) all work the same way:

1. Each player prepares and submits order chains for their free nobles and
   armies.
2. The engine checks each submission: a syntax or reception error rejects the
   affected submission, without touching the rest of the game (section 4).
3. The engine resolves **everyone together**: supply, intentions, supports,
   combats, movement, retreats, joins, dispersals, and chain progression.
4. Territorial control, noble positions, and events are updated, then a **turn
   report** is produced.

An army executes **at most one line of its chain per action season**: an `A`
or `J` order therefore crosses at most one adjacent territory in that
resolution. For example, `ROS A BOI` then `BOI A ATL` moves an army from ROS
to BOI this turn, then from BOI to ATL on the next action turn. A multi-line
chain therefore spans several turns, and stays attached to the army between
resolutions until it ends or breaks — section 3 walks through a full example.

Winter is different: it's a **management truce**, with no chain, movement,
combat, or supply. The player submits a list of direct investments, processed
in the order they were entered (section 8).

| Season | What happens |
|---|---|
| Spring, summer, autumn | Supply is calculated first, then intentions, supports, combats, movement, joins, dispersals, and chain progression are resolved together. Each army has only one current line. |
| Winter | No supply or chain order is resolved: investments are applied one by one, in the entered list, then stocks are conserved and repatriated. |

Spring, summer, and autumn orders are therefore not a queue between players:
each one is evaluated together with the whole turn's intentions. Winter, by
contrast, is a strictly sequential phase.

---

## 3. A Turn, Start to Finish

Here's a complete turn, to put a concrete face on the vocabulary in the
following sections.

**The situation.** Hugues owns ROS (his capital, a castle) with a 2-troop army
and his noble HUG, as well as FOU, a small 1-troop garrison holding his second
noble, ODA. Both ROS and FOU are adjacent to ATL, held by Brune: a 2-troop army
and her noble MIA. ATL is adjacent to NOR, an empty territory Brune controls.

Hugues wants to take ATL. Since HUG and ODA are two separate nobles, he can
have each of them emit a chain this turn — it's precisely because he has two
nobles that he can combine an attack and a support in the same resolution.

**What Hugues writes.** One chain per noble, one line per order (the web UI
adds the noble header automatically):

```text
HUG
ROS A ATL        # attack ATL from ROS
```

```text
ODA
FOU S ROS - ATL  # offensive support for the attack ROS -> ATL
```

**What Brune writes**, without knowing Hugues's plans (simultaneous resolution
means neither sees the other's chains):

```text
MIA
H ATL            # holds her position
```

**The resolution.** The engine adds up the forces engaged on ATL: Hugues's
attack weighs 2 (the ROS army) + 1 (the FOU support) = 3; Brune's defense
weighs 2. To keep this example simple, we ignore the noble bonus detailed in
section 6 here — it would apply identically on both sides of this
calculation. 3 against 2: Hugues wins, and his army occupies ATL. Brune's army
is dislodged and must retreat; NOR is empty and controlled by her, so it's her
destination (section 6 covers the full priority order). Noble MIA follows her
army to NOR.

**What everyone sees afterward.** Both chains involved had only one line: they
are complete, and both of Hugues's armies are now No Orders for the next
season, unless he issues new chains. In the report, Brune took part in the
combat as the defender: she sees the exact forces (3 against 2) and the
support involved. A third player who took no part as attacker, defender, or
supporter would only see that a combat happened on ATL and its general
outcome, without the force details (section 1).

---

## 4. Writing a Chain

A chain consists of the **emitting noble's trigram** (header line), followed by
**one line per order**. Each order line has the form
`POSITION SYMBOL [targets...]`. Comments start with `#`, blank lines are
ignored, and case is normalized by the parser.

> In the web UI, the noble header is **added automatically before submission**:
> you only write the order lines.

### Order Liaison: single or loop

- **single** — a line without parentheses. The chain stops at the first
  failure, and the suffix is abandoned.
- **loop** — the entire line is enclosed in parentheses, `(…)`. The order is
  retried at each resolution until it succeeds; a hold in loop puts the army
  on standby. A mechanically impossible error always breaks the chain, even in
  loop.

A chain is not limited to one season: a successful line advances the chain
index, and the next line waits for the next resolution — that's what would
happen if Hugues's chain in section 3 had a second line, `ATL A NOR`: it would
wait for the following turn to execute. A loop line deliberately keeps the
same order when it has to wait for an opening, and a movement invalidated by
bad weather pauses the chain the same way: the order stays in place and
retries next season.

If a chain contains an error, **the whole submission is rejected with the line
to fix**, and nothing is received until it is corrected: syntax, unknown code,
non-adjacent territories, a join that is not the last order, a support of its
own territory, a transfer to its own territory, or an invalid noble
assignment. The interface reports these errors while you type. The `T`
transfer is the exception to adjacency: it uses the supply network
(section 5).

### Reception: Whose Order Is It

- The chain is attached **immediately and atomically** to the army present at
  the position of its first order; it replaces that army's previous chain.
- A free or hostage noble emits only **one chain per turn** — this is the
  central constraint described in section 1. It may command any army
  belonging to its player; it does not have to be present at the first order's
  position (that's how ODA, staying at FOU, can still act on the FOU army in
  section 3). A chain targeting another player's army, a noble in the dungeon,
  or a noble that has already emitted, is rejected.
- If **several chains target the same army in the same turn**, their
  concurrent reception is invalidated: none is received, and the army
  receives no new chain for that turn. A chain already held stays unchanged.
- An army without a chain is **No Orders**: it receives no automatic action,
  but remains defensible (section 6).
- A player with no free or hostage noble able to emit does not have to submit
  chains during an action season.

---

## 5. Order Cheat Sheet

The orders below are available in spring, summer, and autumn. `XXX`, `YYY`, and
`ZZZ` are territory trigrams; `NNN` is a noble trigram. None costs resources in
an action season.

| Symbol | Syntax | Effect |
|---|---|---|
| `A` | `XXX A YYY` | Attack or move to adjacent `YYY`. |
| `S` | `XXX S YYY` | Defensive support for the army holding `YYY`. |
| `S` | `XXX S YYY - ZZZ` | Offensive support for the attack from `YYY` to `ZZZ`. |
| `H` | `H XXX` | Hold on `XXX`. |
| `J` | `XXX J YYY` | Peaceful join toward adjacent `YYY`; **must be the last order**. |
| `P` | `P XXX` | Pillage the infrastructure on the occupied territory. |
| `D` | `XXX D DEST1 DEST2 ...` | Peaceful dispersal at strength 0: destinations are processed in appearance order, may repeat, and troops arriving on the same territory are stacked. |
| `T` | `XXX T YYY N` | Transfer `N` resources to a castle, village, or opposing army through the supply network. |

### Attack (`A`) and Join (`J`)

**The gist**: `YYY` must be adjacent to `XXX` through a passable border; the
whole army moves there. An attack may fight an enemy army there (section 6
covers the force calculation). A join never fights: it is peaceful
strength-0 movement, and is repelled if the destination is contested.

**Edge cases**:

- a destination is contested as soon as at least one enemy attack takes part
  and no attacking army wins the combat;
- if an allied attack wins the combat on `YYY`, a join targeting the same
  territory may fuse with the winner; enemy attacks that lose that combat do
  not prevent it from arriving;
- an army may attack its own empty castle to garrison it without being
  repelled by the castle's defense (self-capture);
- a join must be the **last order** in the chain.

### Support (`S`)

**The gist**: a support strengthens an army of any nationality, provided the
supported army actually performs the announced action.

- **defensive** (`XXX S YYY`): strengthens the army holding `YYY`, if `YYY` is
  adjacent to `XXX` (an army cannot support itself);
- **offensive** (`XXX S YYY - ZZZ`): strengthens the attack from `YYY` to
  `ZZZ`; both `XXX` and `YYY` must be adjacent to `ZZZ`, and `YYY` must be the
  army that actually attacks `ZZZ` (that's the case for FOU in section 3,
  adjacent to both ATL and ROS).

**Edge cases**: a failed attack creates no additional penalty — the supported
army follows the normal combat result, and its own chain continues or breaks
according to its liaison. An attack from a territory different from the
supported target can **cut** a support.

### Hold (`H`) and Pillage (`P`)

`H XXX`: the army stays in place and can receive defensive support — that's
what Brune writes in section 3.

`P XXX`: destroys the infrastructure on the occupied territory; a pillage
bonus ({{pillage_bonus}} R) is credited to the nearest allied source, and may
reduce famine (section 7).

### Dispersal (`D`)

**The gist**: `XXX D DEST1 DEST2 ...` processes destinations in appearance
order, with at most one troop per destination. This is peaceful strength-0
splitting: it never fights an enemy army. A free, uncontested destination is
taken; an allied destination fuses with the army already there; a contested
destination repels the assignment without consuming a troop.

**Edge cases**:

- a destination is adjacent to `XXX` or equal to `XXX`; destinations may
  repeat;
- a destination occupied by an enemy army, contested, or troopless does not
  consume a troop; a later destination may still receive one;
- several allied dispersals arriving on the same territory are stacked into
  one army;
- troops that cannot be sent remain at the origin; a list shorter than the
  army therefore leaves a remainder in place;
- nobles explicitly assigned follow the produced group: `*` assigns all
  remaining nobles, `*NNN` assigns noble `NNN`; nobles not mentioned remain at
  the origin while a troop remains there. If all troops leave the origin and a
  present noble has no produced group, the order is invalid at execution;
- the chain carried by the army follows the **first listed group**:
  `BRI D ATL NOR` makes the chain follow the ATL group as soon as ATL receives
  its first troop. To keep the chain at the origin while sending troops
  elsewhere, write `BRI D BRI ATL NOR` — do not skip to NOR after ATL fails
  while the remainder stays at BRI, that would invalidate the rest of the
  chain;
- in `single`, untreated destinations produce a partial dispersal and the
  chain advances anyway; in `loop`, the remainder retries until an army
  arrives at every destination — if the army runs out before every
  destination is processed, the order is invalid.

```text
BRI D ATL ATL              # two troops stacked in the army arriving at ATL
BRI D ATL                  # one troop to ATL, the remainder stays on BRI
BRI D ATL*HUG NOR          # HUG to ATL, the other unit to NOR
BRI D BRI ATL NOR          # BRI keeps the chain; the other groups split away
(BRI D ATL NOR)            # looped dispersal
```

### Transfer (`T`)

**The gist**: `XXX T YYY N` is executed after supply, by the army on `XXX`.
`YYY` must be a castle, village, or the territory of an army controlled by
another living player; a bare depot cannot receive. The source territory only
needs to contain stock.

**Edge cases**:

- the route follows the donor's supply range (`{{supply_range}}` territories,
  plus controlled depots); any enemy army on an intermediate territory blocks
  it, but an enemy army at the destination is allowed;
- a famished army cannot transfer; the amount is capped at
  `{{cost_base}}^(N - 1)` for an army of `N` troops, without subtracting local
  rations; the army performs no other order that turn;
- a stock shortage has no effect and does not break a `single` chain; in
  `loop`, the transfer retries, and when the remaining stock is below the
  requested amount, the remainder is sent as a partial final delivery and the
  order completes.

---

## 6. Combat, Strength, and Retreats

### Who Wins a Combat

An army is the sole force entity on a territory: it has an owner and a troop
size, and all its troops share the same chain — an army cannot contain mixed
orders.

- attack strength is the attacking army's **size**, with
  **+{{noble_command_bonus}}** if a free allied noble is present on its
  territory;
- support strength is the supporting army's size, with the same bonus;
- an army's defense receives the same bonus under the same condition;
- a castle gives a fixed defensive bonus of **+{{castle_defense_bonus}}**,
  even without an army — unless all attackers belong to the castle's owner
  (see self-capture, section 5);
- the **strictly unique** highest strength wins; a top tie produces a
  **standoff**, including on an empty territory.

That's exactly the calculation walked through in section 3: 3 against 2, no
tie, Hugues wins.

### Retreating

A dislodged army loses its movement and must retreat as a whole, to an
adjacent destination chosen by descending priority order:

1. an empty territory controlled by the retreating army's owner (with or
   without a castle), even if fought over this turn — that's the case for NOR
   for Brune in section 3;
2. an uncontrolled empty territory (neutral or enemy), without a castle and
   not fought over this turn;
3. an adjacent, non-dislodged friendly army (smallest troop size first), with
   merging: the host gains `N − 1` troops if the retreating army has `N ≥ 2`
   troops, or `1` troop if `N = 1` (no loss). Multiple retreating armies can
   merge sequentially into the same friendly host without destructive
   collision.

Ties within a bucket are broken by distance to the nearest controlled castle
or village, then ascending trigram. For friendly armies, sorting is by troop
size ascending, then distance to the nearest controlled source, then
ascending trigram. The attacker's origin territory is always excluded, and
neutral or enemy empty castles defend against retreat: they are never a valid
destination. Two armies that must retreat to the same empty territory with no
alternative are destroyed. Retreat resolution order follows the ascending
trigram of their origin territory.

Territorial control follows the army that stops there; acquired control
remains after the army leaves, until an enemy army stops there.

### Nobles During a Combat

Nobles ride with armies: they follow movement, attacks, joins, dispersals, and
retreats — that's how MIA ends up at NOR with Brune's army in section 3. A
noble counts neither toward supply nor combat losses; it may remain alone on a
territory after its army is lost. There is no limit, in this version, on the
number of nobles carried by an army: an army transports every noble present on
its territory.

The +{{noble_command_bonus}} bonus comes only from a noble physically present
on the army's territory when strength is calculated: issuing a chain remotely
(section 4) does not teleport the noble or give the distant army a bonus. A
voluntarily transferred noble follows the group that receives it during a
dispersal (`*` or `*NNN`, section 5); it only grants its command bonus if that
group actually carries it when it fights or defends.

When an army carrying nobles is **destroyed** on a territory occupied by an
enemy army, those nobles are captured and become `hostage` by default. A
hostage noble may continue to emit a chain (section 4); only moving it to the
dungeon, covered in section 8, removes that ability.

---

## 7. Logistics and Supply

### The Exponential Cost of an Army

Supply is resolved **at the start of every action season**, before orders,
combats, and movement; there is no supply phase in winter. An army of `N`
troops demands:

```text
cost = {{cost_base}}^(N - 1)  rations
```

| Size | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Ration cost | {{army_cost.1}} | {{army_cost.2}} | {{army_cost.3}} | {{army_cost.4}} | {{army_cost.5}} |

A one-troop army already demands `{{army_cost.1}}` ration: it is not
automatically free. This progression is what makes a large army fragile the
moment its supply route is cut.

### Where the Food Comes From

The production of the territory an army occupies is granted to that army
alone, up to its demand; surplus is lost, and the remainder is its demand to
supply. There is only ever one army per territory, so there is no
distribution between armies: an enemy army on a neighboring territory never
takes your territory's ration.

**Territory food production (rations)**: plain {{ration_terrain.plain}};
forest {{ration_terrain.forest}}; hill {{ration_terrain.hill}}; mountain
{{ration_terrain.mountain}}; swamp {{ration_terrain.swamp}};
**+{{infra_rations_bonus}}** when the territory has a castle or village.

Example: a 2-troop army on a hill with a castle (local production
{{ration_terrain.hill}}, castle bonus {{infra_rations_bonus}}) receives 2
rations, covering its full demand. The same army on a swamp (production
{{ration_terrain.swamp}}) receives only 1 ration and must cover the rest
elsewhere.

**Supply sources**: controlled castles, villages, and caches. A castle or
village produces {{base_production}} R of stock per turn; a bare territory has
no production of its own, but its stock (if any) serves as a cache. The flow
crosses allied, neutral, or enemy-controlled territories, and only stops
before a territory occupied by an enemy army. Base range is
{{supply_range}} territories; each controlled supply depot encountered along
the route adds {{depot_range_bonus}} territories. A neutral village keeps its
stock, inaccessible before capture.

Each source calculates its own production by adding the level of **every
adjacent mill**: one mill can feed every neighboring source, with no owner
filter, and an orphaned mill (with no adjacent castle or village) produces
`0 R`. For example, a village surrounded by two level-1 mills produces
`{{base_production}} + 1 + 1 R`. The presence or position of a noble never
conditions this production.

### Stocks and Famine

When there is a deficit:

1. stocks in controlled castles, villages, and caches are emptied first
   (smallest first, with the territorial trigram as tie-breaker);
2. remaining armies enter **famine**, starting with those furthest from their
   source, then the largest, then descending trigram.

A famished army **attacks and defends at strength 0** for the turn, even when
it carries a free noble. If it occupies infrastructure, it **pillages it
automatically**; the pillage bonus, reduced by its residual demand, may end
its famine. If pillage is insufficient or impossible, it loses **1 troop**,
never falling below 1 — but it stays famished and at strength 0 for the whole
current season, even if that loss would make its future demand sustainable;
the loss repeats in every season the army remains famished.

Example: a 2-troop army in deficit demands 2 rations. If its stocks and
pillage cannot cover the deficit, it loses one troop and becomes a 1-troop
army; it still stays at strength 0 this turn, even though a 1-troop army would
then only demand 1 ration.

In the interface, selecting an army or a controlled source shows its supply or
the area it reaches (outside winter only). A transfer being drafted also shows
its route and blockers.

### What Infrastructure Provides

A territory carries only **one infrastructure**; their cost and build
conditions are detailed in section 8.

| Infrastructure | v1 effect |
|---|---|
| Mill | +1 stockable R per level at each adjacent source |
| Supply depot | +{{depot_range_bonus}} territories of supply range when controlled |
| Castle | +{{castle_defense_bonus}} defense, +{{infra_rations_bonus}} rations, produces {{base_production}} stockable R per turn, supply anchor |
| Village | +{{infra_rations_bonus}} rations, produces {{base_production}} stockable R per turn, supply anchor after capture |

---

## 8. Winter Orders

Winter accepts **no chains or movement**: only direct investments, one order
per line, applied in the entered order.

| Investment | Syntax | Condition | Cost (R) |
|---|---|---|---|
| Recruit a noble | `R N XXX` | `XXX` controlled, with a castle or village and a player army | {{costs.noble}} |
| Recruit a troop | `R T XXX` | `XXX` controlled, and a free player noble on `XXX` or adjacent | {{costs.troop}} |
| Build or upgrade a mill | `C M XXX` | `XXX` controlled; a new mill on an **empty** territory adjacent to a productive castle or village, or an existing mill adjacent to that source | {{costs.mill_levels.0}} (L1), {{costs.mill_levels.1}} (L2), {{costs.mill_levels.2}} (L3) |
| Build a castle | `C C XXX` | `XXX` controlled | {{costs.castle}} |
| Build a supply depot | `C D XXX` | `XXX` controlled | {{costs.supply_depot}} |
| Designate a capital | `E C XXX` | a controlled castle on `XXX` | 0 |
| Place a noble in hostage status | `O N NNN` | `NNN` is an opposing prisoner held by the player | 0 |
| Place a noble in the dungeon | `P N NNN` | `NNN` is an opposing prisoner held by the player | 0 |
| Liberate a noble | `L N NNN` | `NNN` is held by the player; its owner's capital contains one of that owner's armies | {{costs.liberation}} |
| Transfer resources | `G XXX YYY N` | `XXX` is a castle or village controlled by the donor; `YYY` is a castle or village controlled by another player | 0 |

This is where, in winter, the fate of enemy nobles captured in combat
(section 6) is decided: `O`/`P` moves a prisoner between `hostage` and
`dungeon`; a hostage noble may still emit a chain for its original owner while
it remains a hostage, but no longer once in the dungeon — the holder can in
fact read those chains in online games, even when they command an army that
stayed with the noble's owner. Capture normally produces `hostage` status.
`L N NNN` is issued by the **holder**, not the owner: if the owner's capital
exists and contains one of their armies, the noble reappears there free;
otherwise the order is rejected.

A winter transfer is therefore not limited to the donor's own castles and
villages: `G` can directly supply a structure controlled by the recipient. The
debit still follows the usual rules and can use only the donor's payment
reserves.

A mill starts at level 1 and can reach level 3 inclusive. Construction costs
{{costs.mill_levels.0}} R; upgrades to levels 2 and 3 cost
{{costs.mill_levels.1}} R and {{costs.mill_levels.2}} R respectively. `C M` on
a level-3 mill is rejected with reason `mill_max_level_reached`, with no stock
deducted. Mills above level 3 already present in a game are preserved and
remain productive; only new upgrades are blocked.

Investments targeting a territory require **control of that territory**. A
construction replaces the existing structure only when the rule says so: a
**castle built on a village replaces the village** and keeps the territory's
stock. An orphaned mill produces nothing.

### Resource Vocabulary

- `R` means one unit of **stockable resource**: it sits in a territory's
  stock, is produced by a source, and pays for investments when held by a
  controlled castle or village;
- a **ration** is one food unit consumed during an action-season supply phase
  (section 7); local rations do not automatically become stock `R`;
- **stock** is therefore the amount of `R` kept on a territory.

Each controlled castle or village is a separate source, and any controlled
territory with positive stock is an action-season cache source. Every castle
or village produces {{base_production}} R per turn independently of the
others: a second castle is therefore a second source, even though only one
castle is designated as the capital. A mill adds its level to every adjacent
source, even across owner boundaries — see section 7 for the details of this
production.

**Payment**: the cost is taken first from the stock on the target territory,
then from the nearest controlled source; if the total reserve is
insufficient, **no partial payment** is made and the investment is rejected
(reported, with no cost lost). Example: a `C M ATL` costing
{{costs.mill_levels.0}} R first consumes ATL's stock, then the remainder from
the nearest controlled source; if those stocks do not total the required
cost, the build is rejected with no partial payment made.

**End of winter**:

- each remaining castle or village stock is kept at
  `ceil(stock / {{winter_stock_divisor}})` — a stock of 5 R therefore becomes
  3 R;
- a supply depot keeps its stock in full; stock outside a castle, village, or
  depot is lost;
- castle and village stocks outside the capital are brought back to the
  capital, leaving at most {{village_stock_cap}} R per village and
  {{castle_stock_cap}} R per castle;
- without a capital, those stocks remain where they are; depot stock remains
  on its territory.

There is no need to spend everything before winter ends: unspent stock is
first conserved, then surplus is repatriated under these caps. Conservation
and repatriation happen after investments.

---

## 9. Special Cards and Calamities

Playable card orders are submitted in a separate `special` field, distinct
from noble chains and requiring no noble. Winter discards are written in the
`winter` sheet.

- `P FW ROS`: play Fair weather on the region seeded by ROS;
- `P AH ROS`: play Abundant harvest on that region;
- `P RV BRU`: play Revolt on the BRU territory, only when an active bad
  harvest affects its region;
- `D C FW` or `D C AH`: discard a card, winter only.

The hand is replenished automatically in winter after discards; no draw order
is needed.

Fair weather, Abundant harvest, and Revolt can be played in spring, summer,
and autumn, but not winter. Played cards are consumed before army-order
resolution. Fair weather cancels only bad weather, and Abundant harvest
cancels only bad harvest; a card that cancels a calamity does not provide its
regional bonus. Duplicate cards of the same kind are consumed, but only one is
effective: with an active calamity the first card cancels and a second one
applies the regional bonus; without a calamity the first card applies it
directly. The bonus stays capped at one unit per category and region; further
cards are consumed without effect.

The deck contains **{{special_orders.deck_size}} cards**:
**{{special_orders.card.plague}}** plague, **{{special_orders.card.bad_weather}}**
bad weather, **{{special_orders.card.famine}}** bad harvest,
**{{special_orders.card.fair_weather}}** fair weather,
**{{special_orders.card.abundant_harvest}}** abundant harvest, and
**{{special_orders.card.revolt}}** revolt cards. A hand is limited to
**{{special_orders.hand_limit}} cards**, and each player automatically
receives up to **{{special_orders.draw_orders_limit}} bonus cards per
winter**, after discards.

A drawn calamity is programmed into the first free slot of the following
year: spring (**{{special_orders.calamity_slots.spring}}**), summer
(**{{special_orders.calamity_slots.summer}}**), or autumn
(**{{special_orders.calamity_slots.autumn}}**). Its region is selected
deterministically when programmed. The spring augury reveals the kind, season,
and region of every calamity in that year; future auguries remain hidden. As
soon as a calamity is drawn, the interface announces it in the special-cards
panel, and the announcement stays visible until the calamity applies or is
countered. No calamity resolves in winter.

- plague reduces armies by a divisor of
  **{{special_orders.effects.plague_army_divisor}}** and may remove a noble;
- bad weather blocks movements originating from or targeting its region,
  except holds and defensive support;
- bad harvest disables mills and infrastructure ration bonuses in its region;
- Revolt is played on a territory (`P RV TER`) during action seasons,
  provided its region suffers a bad harvest. Each card adds a roll between
  **{{special_orders.effects.revolt_army_min_size}}** and
  **{{special_orders.effects.revolt_army_max_size}}** troops to the
  territory's common neutral army; the territory may be neutral (mere
  brigandage), and an army is raised there when the territory is empty. If the
  territory is occupied, the revolt resolves as a battle between the rebel
  army and the holder: the loser retreats or is destroyed. When an Abundant
  harvest cancels the region's bad harvest, pending revolts in that region are
  canceled and their players take their cards back. A crushed rebellion
  retreats like any defeated army instead of vanishing. Neutral armies never
  lose strength to a famine, but lose one troop at the end of the turn when
  the local production of their territory cannot feed them.

Public rumors are recalculated in every report from the current bonus hands of
all players. They appear when at least two players hold a card, without
revealing the player or the internal card identifier. Several cards of the
same kind are grouped into one graduated sentence: level 1 for a few cards,
level 2 for a stronger presence, and level 3 for exceptional abundance. The
scale is recalibrated to the game's hand capacity (players multiplied by the
hand limit), so the same number of cards does not produce the same level in a
small and a large game.

---

## 10. End of Game, Score, and Victory

A game's duration is chosen at creation, between 1 and 50 years (10 by
default). A year has four turns; the interface shows the historical year
`1000 + year`, so “Year 1001” on the first turn.

**Elimination**: a player is eliminated when they no longer control any
territory and no longer own any army. Nobles alone do not keep a player in the
game. An eliminated player no longer submits orders.

**End of game**: the game ends immediately when only one player remains (that
player wins), or otherwise after the final winter of the chosen duration is
resolved — the player with the highest score wins. An exact tie at the top has
no winner.

**Score**, recomputed after every turn and visible to everyone:

| Element | Points |
|---|---:|
| Controlled territory | 1 |
| Controlled village | 2 |
| Controlled mill | 1 |
| Controlled castle | 5 |
| Held noble | 2 |
| Troop | 1 per unit in their armies |
| Resource `R` | 1 per unit in stock on their controlled territories |

Infrastructure and resources only score on a controlled territory. A free
noble counts for its owner. A captured noble, hostage or in the dungeon,
counts for the player who controls the territory where it stands, not for its
original owner.
