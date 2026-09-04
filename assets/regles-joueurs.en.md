# Game Rules — Crown & Borough

**Friendly version (human players).** This document describes the rules currently
active on the game server. Numeric values come from `assets/balance.yaml`, which
remains the source of playable numbers; when documents disagree, the engine wins.

---

## 1. Overview

Crown & Borough is a turn-based medieval strategy game played on a map of
territories connected by a graph. Each player programs **order chains** for their
armies; all chains are resolved **simultaneously**.

The two pillars of tension are:

- simultaneous resolution of intentions, supports, and combats;
- exponential logistics: large concentrations of troops are costly and
  vulnerable to supply shortages.

A game accepts **2 to 16 players**. The v1 server is a “hotseat” session: the map
and numerical data (owners, army sizes, stocks, infrastructure, and nobles) are
visible to everyone.

Each player starts on a distinct territory: a **castle** is built there for free
(it becomes the **capital**), with 10 R in stock, a one-troop army, and one free
noble.

### Inspirations

The project draws in particular on [Fief](https://boardgamegeek.com/boardgame/107704/fief)
for its feudal setting and territorial stakes, and on
[Diplomacy](https://boardgamegeek.com/boardgame/483/diplomacy) for simultaneous
order programming, supports, and conflict resolution. These games are design
inspirations, not sources of rules applicable to Crown & Borough.

---

## 2. Game Cycle and Seasons

A year has **four turns**: spring, summer, autumn, and **winter**. The turn
counter increases by one at every season, including winter (a new spring follows
winter).

### Action Turns (spring, summer, autumn)

1. Each player prepares and submits order chains for their free nobles and armies.
2. The engine checks submissions: a syntax or reception error prevents the
   affected submission from resolving, without changing the state.
3. The engine simultaneously resolves supply, intentions, supports, combats,
   movement, retreats, joins, dispersals, and chain progression.
4. Territorial control, noble positions, and events are updated, then a **turn
   report** is produced.

An army executes at most **one line of its chain per action season**. An `A` or
`J` order therefore crosses at most one adjacent territory in that resolution.
The chain stays attached to the army between turns: its following lines execute
in later seasons until the chain ends or breaks. For example, `ROS A BOI` then
`BOI A ATL` moves the same army from ROS to BOI this turn, and from BOI to ATL
on the next action turn. A dispersal can create several groups in one
resolution, but it remains peaceful movement to adjacent territories.

### Winter Phase

Winter is a **management truce**: no action chain, movement, combat, or supply is
resolved. The player submits a list of direct investments, processed in the
entered order (see section 5).

| Season | Orders and timing |
|---|---|
| Spring, summer, autumn | Supply is calculated at the **start of resolution**, then intentions, supports, combats, movement, joins, dispersals, and chain progression are resolved together. Each army has only one current line. |
| Winter | No supply or chain order is resolved: direct investments are applied sequentially in the entered list, then stocks are conserved and repatriated. |

Action-season orders are therefore not a queue between players: each one is
evaluated together with the turn's intentions. Winter is instead a sequential
management phase.

---

## 3. Writing a Chain

A chain consists of the **emitting noble's trigram** (header line), followed by
**one line per order**.

> In the web UI, the noble header is **added automatically before submission**:
> you only write the order lines.

Each order line has the form `POSITION SYMBOL [targets...]`. Comments start with
`#`, blank lines are ignored, and case is normalized by the parser.

Example of a complete chain:

```text
HUG              # header: emitting noble (added by the web UI)
ROS A BOI        # attack BOI from ROS
BOI S BRU - FOU  # offensive support
BOI J ROS        # join (must be the last order)
```

### Order Liaison

- **single**: a line without parentheses. The chain stops at the first failure,
  and the suffix is abandoned.
- **loop**: the entire line is enclosed in parentheses, `(…)`. The order is
  retried at each resolution until it succeeds; a hold in loop puts the army on
  standby. A mechanically impossible error always breaks the chain.

An order whose position and target are not adjacent is rejected when the chain
is submitted, with no partial reception of the chain.

A chain is not limited to one season: a successful line advances the chain index,
and the next line waits for the next resolution. A `loop` line deliberately keeps
the same order when it has to wait for an opening.

### Reception

- The chain is attached **immediately and atomically** to the army present at the
  position of its first order; it replaces that army's previous chain.
- A free or hostage noble emits only **one chain per turn**. It may command any
  army belonging to its player; it does not have to be present at the first
  order's position. A chain targeting another player's army, a noble in the
  dungeon, or a noble that has already emitted is rejected.
- If **several chains target the same army in the same turn**, their concurrent
  reception is invalidated: none is received and the army receives no new chain
  for that turn.
- An army without a chain is **No Orders**: it receives no automatic action.

---

## 4. Order Cheat Sheet

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

### Attack (`A`) and Join (`J`)

`YYY` must be **adjacent** to `XXX` through a passable border. The whole army
moves to `YYY`. An attack may fight an enemy army there; a join does not fight and
is repelled if the destination is contested. A join must be the last order in the
chain. A join and a dispersal are never attacks: they are peaceful strength-0
movement and cannot dislodge anyone.

### Support (`S`)

A support strengthens an army of **any nationality**:

- **defensive** (`XXX S YYY`): strengthens the army holding `YYY`, if `YYY` is
  adjacent to `XXX` (an army cannot support itself);
- **offensive** (`XXX S YYY - ZZZ`): strengthens the attack from `YYY` to `ZZZ`.

For offensive support, both `XXX` and `YYY` must be adjacent to the destination
`ZZZ`, and YYY must be the army that actually attacks `ZZZ`. A failed attack
creates no additional penalty: the army follows the normal combat result and its
chain continues or breaks according to its liaison.

It only counts if the supported army performs the announced action. An attack
from a territory different from the supported target can **cut** a support.

### Hold (`H`) and Pillage (`P`)

`H XXX`: the army stays in place and can receive defensive support.
`P XXX`: destroys the infrastructure on the occupied territory; a pillage bonus
(2 R) is credited to the nearest allied source and may reduce famine.

### Dispersal (`D`)

`XXX D DEST1 DEST2 ...` processes destinations in appearance order, with at most
one troop per destination. This is peaceful strength-0 splitting: it does not
fight an army already present; a free, uncontested destination is taken, while a
contested destination repels that assignment and receives no troop.

- a destination is adjacent to `XXX` or equal to `XXX`; destinations may repeat;
- an occupied, contested, or troopless destination does not consume a troop; a
  later destination may still receive one;
- troops that cannot be sent remain at the origin; a list shorter than the army
  therefore leaves a remainder in place;
- troops arriving at the same destination are stacked into one army;
- nobles explicitly assigned follow the produced group: `*` assigns all remaining
  nobles, `*NNN` assigns noble `NNN`; nobles not mentioned remain at the origin
  while a troop remains there;
- if all troops leave the origin and a present noble has no produced group, the
  order is invalid at execution;
- the chain carried by the army follows the **first listed group**. Thus,
  `BRI D ATL NOR` makes the chain follow the ATL group when ATL receives the
  first troop; to keep the chain at the origin while sending troops elsewhere,
  write `BRI D BRI ATL NOR`. Do not skip to NOR after ATL fails while the
  remainder stays at BRI: that would invalidate the rest of the chain;
- in `single`, untreated destinations produce a partial dispersal and the chain
  advances; in `loop`, the remainder retries until an army arrives at every
  destination; if the army is exhausted before all destinations are processed,
  the order is invalid.

Examples:

```text
BRI D ATL ATL              # two troops stacked in the army arriving at ATL
BRI D ATL                  # one troop to ATL, the remainder stays on BRI
BRI D ATL*HUG NOR          # HUG to ATL, the other unit to NOR
BRI D BRI ATL NOR          # BRI keeps the chain; the other groups split away
(BRI D ATL NOR)            # looped dispersal
```

---

## 5. Winter Orders

Winter accepts **no chains or movement**: only direct investments, one order per
line, applied in the entered order.

| Investment | Syntax | Condition | Cost (R) |
|---|---|---|---|
| Recruit a noble | `R N XXX` | `XXX` controlled, with a castle or village and a player army | 2 |
| Recruit a troop | `R T XXX` | `XXX` controlled, and a free player noble on `XXX` or adjacent | 1 |
| Build or upgrade a mill | `C M XXX` | `XXX` controlled; a new mill on an **empty** territory adjacent to a productive castle or village, or an existing mill adjacent to that source | 3 |
| Build a castle | `C C XXX` | `XXX` controlled | 10 |
| Build a supply depot | `C D XXX` | `XXX` controlled | 3 |
| Designate a capital | `E C XXX` | a controlled castle on `XXX` | 0 |
| Place a noble in hostage status | `O N NNN` | `NNN` is an opposing prisoner held by the player | 0 |
| Place a noble in the dungeon | `P N NNN` | `NNN` is an opposing prisoner held by the player | 0 |
| Liberate a noble | `L N NNN` | `NNN` is held by the player; its owner's capital contains one of that owner's armies | 0 |

### Hostage and Dungeon

Orders `O N NNN` and `P N NNN` target an **opposing prisoner** held on the
territory of one of the player's armies. `O` gives the noble `hostage` status and
`P` gives `dungeon` status. Capture normally produces `hostage` status. A hostage
noble may emit a new chain while held; a dungeon noble cannot. The orders can
move a prisoner from one status to the other.

Investments targeting a territory require **control of that territory**. A
construction replaces the existing structure only when the rule says so: a
**castle built on a village replaces the village** and keeps the territory's
stock. An isolated (orphaned) mill produces nothing.

### Resource vocabulary

- `R` means one unit of **stockable resource**: it sits in a territory's stock,
  is produced by a source, and pays for investments;
- a **ration** is one food unit consumed during an action-season supply phase.
  Local rations are produced and distributed on the spot; they do not
  automatically become stock `R`;
- **stock** is therefore the amount of `R` kept on a territory.

Each controlled castle or village is a separate source. Every source produces
`1 R` per turn independently of the others. A second castle is therefore a
second production and supply source, even though only one castle is designated
as the capital. A mill is built only on an empty territory adjacent to a
productive castle or village; it increases the production of **every** neighboring
source, with no owner filter. For example, a level-1 mill between a village and
two castles adds `+1 R` to each of those three production points. Even if the
mill is on a territory controlled by another player, it adds this bonus to a
neighboring source controlled by the relevant player. A noble elsewhere on the
map does not prevent `C M ATL` and is not required to build it. If the build
territory already has another infrastructure, the order is rejected with
`structure_present`: a territory never carries two infrastructures.

**Payment**: the cost is taken first from the stock on the target territory, then
from the nearest controlled source; if the total reserve is insufficient, **no
partial payment** is made and the investment is rejected (reported, with no cost
lost).

Example: a `C M ATL` costing 3 R consumes 1 R from ATL's stock, then 2 R from
the nearest controlled source. If those stocks total only 2 R, the build is
rejected and neither unit is removed.

**End of winter**:

- each remaining stock is kept at `ceil(stock / 2)`;
- stocks outside the capital are brought back to the capital, leaving at most
  **1 R per village** and **2 R per castle**;
- without a capital, stocks remain where they are.

There is no need to spend everything before winter ends: unspent stock is first
conserved, then surplus is repatriated under these caps. A stock of 5 R therefore
becomes 3 R with `ceil(5 / 2)`. Conservation and repatriation happen after
investments, and a territory without a castle or village does not keep stock.
For example, an outlying village keeps at most 1 R after conservation; its
surplus goes to the capital, while an outlying castle may keep 2 R.

### Special cards

The deck contains **{{special_orders.deck_size}} cards**: **{{special_orders.card.plague}} plague**, **{{special_orders.card.bad_weather}} bad weather**, **{{special_orders.card.famine}} famine**, **{{special_orders.card.fair_weather}} fair weather**, **{{special_orders.card.abundant_harvest}} abundant harvest**, and **{{special_orders.card.revolt}} revolt** cards.

A hand is limited to **{{special_orders.hand_limit}} cards**. Each player may use at most **{{special_orders.draw_orders_limit}} draws** per winter. Calamities are programmed into spring (**{{special_orders.calamity_slots.spring}}**), summer (**{{special_orders.calamity_slots.summer}}**), and winter (**{{special_orders.calamity_slots.winter}}**) slots. Plague reduces army sizes by a divisor of **{{special_orders.effects.plague_army_divisor}}**.

---

## 6. Armies, Combat, and Logistics

### Armies and Strength

An army is the sole force entity on a territory: it has an owner and a troop
size. All its troops share the same chain; an army cannot contain mixed orders.

- attack strength is the attacking army's **size**, with **+1** if a free allied
  noble is present on its territory;
- support strength is the supporting army's size, with **+1** if a free allied
  noble is present on its territory;
- an army's defense receives the same **+1** bonus when commanded by a free allied
  noble;
- a castle gives a fixed defensive bonus of **+1**, even without an army;
- the **strictly unique** highest strength wins; a top tie produces a **standoff**,
  including on an empty territory;
- a dislodged army loses its movement and must **retreat**;
- a retreat moves the whole army to a free adjacent territory that was not fought
  over during the turn and differs from the attacker's origin (ties are broken by
  ascending trigram; two armies with no alternative on the same territory are
  destroyed).

Territorial control follows the army that stops there; acquired control remains
after the army leaves until an enemy army stops there.

### Exponential Supply

Supply is resolved **at the start of every action season**, before orders,
combats, and movement. There is no supply phase in winter. A one-troop army
demands `1` ration; it is not automatically free.

An army of `N` troops demands:

```text
cost = 2^(N - 1)  rations
```

| Size | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Ration cost | 1 | 2 | 4 | 8 | 16 |

The food production of the army's own territory is granted to that army alone:
an army only consumes the production of the territory it occupies, at most
**one ration**, and the remainder is its demand to supply. There is only ever
one army per territory, so there is no distribution between armies: an enemy
army on a neighboring territory never takes your territory's ration.

Brigands and other neutral armies also take the ration of the territory they
occupy, but receive no additional supply from a player's controlled source
stocks.

Example: a 2-troop army on a hill with a castle (production 1 + 2 = 3 rations)
receives 1 ration; its remaining demand is 2 − 1 = 1 ration to cover from its
sources. An army on a swamp (production 0) receives nothing and must cover its
full demand.

**Territory food production**: 1 ration on plain, forest, or hill; 0 ration on
mountain or swamp; **+2 rations** when the territory has a castle or village.

**Supply sources**: **controlled castles and villages**. A castle or village
produces **1 R of stock per turn**. The flow crosses allied or neutral
territories and stops before an enemy territory. Base range is **3 territories**;
each controlled supply depot encountered along the route adds **2 territories**.
A neutral village keeps its stock, inaccessible to the player before capture.

Each source calculates its own `R` production: its base production plus the level
of **every adjacent mill**. One mill can therefore feed every neighboring source;
it is not reserved for the owner of its territory. An orphaned mill, with no
adjacent castle or village, produces `0 R`. For example, a village surrounded by
two level-1 mills produces `1 + 1 + 1 = 3 R`; the same mills also add their level
to every neighboring castle. The presence or position of a noble never
conditions `C M XXX` or this production: a noble in NOR does not prevent the
player from building `C M ATL` when ATL is empty, controlled, and adjacent to
the required source.

### Stocks and Famine

When there is a deficit:

1. stocks in controlled castles and villages are emptied (smallest first, with
   the territorial trigram as tie-breaker);
2. remaining armies enter **famine**, starting with those furthest from their
   source, then the largest, then descending trigram.

A famished army **attacks and defends at strength 0** for the turn, even when it
carries a free noble. If it is on infrastructure, it **pills it automatically**;
the pillage bonus, reduced by its residual demand, may end its famine.

If pillage is insufficient or impossible, it loses **1 troop**, never falling
below 1. It nevertheless remains famished and at strength 0 for the whole
current season, even if that loss would make its future demand sustainable. The
loss repeats in every season in which the army is still famished.

Example: a 2-troop army in deficit demands 2 rations. If its stocks and pillage
cannot cover the deficit, it loses one troop and becomes a 1-troop army; it stays
at strength 0 this turn even though a 1-troop army would then demand only one
ration.

The endpoint `GET /api/supply?territory=XXX` previews an army's supply or the area
reached from a controlled source (outside winter only).

### Infrastructure

A territory carries only **one infrastructure**.

| Infrastructure | Condition | v1 effect | Cost |
|---|---|---|---|
| Mill | Build on an empty controlled territory adjacent to a castle or village; upgrade an existing mill adjacent to that source | +1 stockable R per level at **each** adjacent source | 3 |
| Supply depot | None | +2 territories of supply range when controlled | 3 |
| Castle | None | +1 defense, +2 rations, produces 1 stockable R per turn, supply anchor | 10 |
| Village | Generated neutral, **not buildable** | +2 rations, produces 1 stockable R per turn, supply anchor after capture | — |

---

## 7. Nobles: Capture, Movement, and Capacity

Nobles **ride with armies**: they follow movement, attacks, joins, dispersals, and
retreats. A noble counts neither toward supply nor combat losses. A player's free
noble present on its army's territory grants that army **+1 strength** once;
held enemy nobles, hostage nobles, and dungeon nobles do not grant this bonus. A
noble may remain alone on a territory after its army is lost.

**Command capacity**:

- a **free or hostage** noble emits at most **one chain per turn** (a new chain
  means a new turn);
- a **dungeon** noble (`dungeon`) cannot emit a new chain;
- the noble may give the chain to **any army belonging to its player**; it does
  not have to be present at the receiving territory;
- the chain applies to the whole army. The command bonus comes only from a free
  allied noble **present on the army's territory when strength is calculated**:
  issuing a chain remotely does not teleport the noble or give the distant army
  a bonus.

> In this version there is no limit to the number of **nobles carried by an
> army**: an army transports every noble present on its territory.

**Capture**: when an army carrying nobles is **destroyed** on a territory
occupied by an enemy army, the nobles it carried are captured and become
`hostage` by default. A hostage noble may continue to emit a chain; only moving
it to the dungeon removes that ability. The player holding it can read the
chains emitted by that hostage in online games, even when they command an army
that remained with the noble's owner.

**Liberation**: during winter, `L N NNN` is issued by the player holding the
prisoner, not by its owner. If the owner's capital exists and contains one of
their armies, the noble reappears **free in that capital**; otherwise the order is
rejected.

A voluntary noble transfer uses a dispersal. For example,
`BRI D ATL*HUG NOR` sends HUG with the ATL group. Noble HUG grants the `+1`
bonus only if that group actually carries HUG when it fights or defends.

A player who has no free or hostage noble able to emit does not have to submit
chains during an action season.

Nobles assigned during a **dispersal** must all be distributed among the
destinations (`*` or `*NNN`); see section 4.

---

## 8. Victory

The planned victory rule (documented in the specifications) is:

> A player is **eliminated** when they control no territory and own no army.
> Nobles alone do not keep a player in the game. The **last living player wins**.

**Current server state**: this end-of-game rule is **not yet enforced by the
engine**. Games therefore remain open: seasons and resolution continue as long as
players submit orders. Elimination detection and the victory condition will be
enabled in a later version.
