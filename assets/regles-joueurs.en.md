# Game Rules — Crown & Borough

**Friendly version (human players).** This document describes the rules currently
active on the game server. Numeric values come from `assets/balance.yaml`, which
remains the source of playable numbers; when documents disagree, the engine wins.

---

## 1. The Pitch

Crown & Borough is a turn-based medieval strategy game played on a map of
territories. Players submit their orders in secret; the engine resolves
everyone **at the same time**, season after season.

One constraint shapes everything else in these rules: **orders come from a
noble, not an army**, and each player only has a handful of them. A free or
hostage noble emits only **one chain per turn** — a chain being a sequence of
orders written ahead of time for an entire army. Since you can't reprogram
every army every turn, the game is as much about strategic planning as
execution: decide today what an army will do two or three turns from now,
and live with the uncertainty of what everyone else is doing in the
meantime. That's where all the vocabulary of "chain," "liaison"
(single/loop), and "reception" detailed in section 4 comes from: it isn't
technical decoration, it's the direct consequence of noble scarcity.

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

Each player starts on a distinct territory, never a mountain and always
bordered by at least two non-mountain territories, where a **castle** is
built for free (their **capital**), with {{starting_resources}} R in stock, a
garrison of {{starting_troops}} troops, and {{starting_nobles}} free
noble(s). The player also receives {{starting_outposts}} one-troop outposts,
placed from the start on that many distinct non-mountain territories
neighbouring their capital, ready to expand their territory from turn one.

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
3. The engine resolves **everyone together**: intentions, supports, combats,
   movement, retreats, joins, dispersals, resource transfers, and chain
   progression.
4. Territorial control and noble positions are updated from the positions and
   outcomes of that resolution.
5. The engine resolves **supply** on these final positions and control —
   territories just captured included: territory income, mill production,
   rations, and famine (section 7) — then a **turn report** is produced.

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
| Spring, summer, autumn | Intentions, supports, combats, movement, joins, dispersals, transfers, and chain progression are resolved together, then territorial control is updated, then supply is calculated last on these final positions and control. Each army has only one current line. |
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
and her noble MIA. ATL is adjacent to NOR, an empty territory with no
controller.

<svg viewBox="0 0 540 280" width="100%" role="img" aria-label="ROS and FOU (Hugues) are adjacent to ATL (Brune), itself adjacent to NOR (no controller, empty)" style="max-width:480px;margin:16px auto;display:block;font-family:system-ui,sans-serif">
  <line x1="90" y1="70" x2="300" y2="130" stroke="#b7a786" stroke-width="2"/>
  <line x1="90" y1="190" x2="300" y2="130" stroke="#b7a786" stroke-width="2"/>
  <line x1="300" y1="130" x2="460" y2="130" stroke="#b7a786" stroke-width="2"/>
  <circle cx="90" cy="70" r="30" fill="#f3ead9" stroke="#3f6b52" stroke-width="3"/>
  <text x="90" y="76" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">ROS</text>
  <text x="90" y="114" text-anchor="middle" font-size="11" fill="#3f6b52">Hugues's capital</text>
  <text x="90" y="128" text-anchor="middle" font-size="11" fill="#594b3c">2 troops · HUG</text>
  <circle cx="90" cy="190" r="30" fill="#f3ead9" stroke="#3f6b52" stroke-width="3"/>
  <text x="90" y="196" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">FOU</text>
  <text x="90" y="234" text-anchor="middle" font-size="11" fill="#3f6b52">Hugues's garrison</text>
  <text x="90" y="248" text-anchor="middle" font-size="11" fill="#594b3c">1 troop · ODA</text>
  <circle cx="300" cy="130" r="30" fill="#f3ead9" stroke="#3a5a8c" stroke-width="3"/>
  <text x="300" y="136" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">ATL</text>
  <text x="300" y="174" text-anchor="middle" font-size="11" fill="#3a5a8c">held by Brune</text>
  <text x="300" y="188" text-anchor="middle" font-size="11" fill="#594b3c">2 troops · MIA</text>
  <circle cx="460" cy="130" r="30" fill="#f8f0e2" stroke="#3a5a8c" stroke-width="2" stroke-dasharray="4 3"/>
  <text x="460" y="136" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">NOR</text>
  <text x="460" y="174" text-anchor="middle" font-size="11" fill="#3a5a8c">no controller</text>
  <text x="460" y="188" text-anchor="middle" font-size="11" fill="#594b3c">empty</text>
</svg>

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

**What Brune writes**, without knowing Hugues's plans (simultaneous
resolution means neither sees the other's chains): nothing for ATL. An army
without a chain is **No Orders** (section 4) but still defends normally — an
`H ATL` order wouldn't change anything here: it only exists to occupy a
chain line while waiting, for example in a loop. Noble MIA, still on the
territory, simply has nothing to emit this turn.

**The resolution.** The engine adds up the forces engaged on ATL: Hugues's
attack weighs 2 (the ROS army) + 1 (the FOU support) = 3; Brune's defense
weighs 2. To keep this example simple, we ignore the noble bonus detailed in
section 6 here — it would apply identically on both sides of this
calculation. 3 against 2: Hugues wins, and his army occupies ATL. Brune's army
is dislodged and must retreat; NOR is empty, has no controller, and wasn't
fought over this turn, so it's her retreat destination (section 6 covers the
full priority order). Noble MIA follows her army to NOR.

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

**Noble card orders.** Noble-deck cards can also be played in spring, summer
and autumn, on a sheet separate from the chains (see section 8 for the
conditions). They are free and apply before the army orders; a recruited noble
acts from the next turn.

| Syntax | Effect |
|---|---|
| `R N CCC XXX` | Recruit the noble of card `CCC` on `XXX`. |
| `C N HHH NNN` | Claim: heir `HHH` claims the titles of another player's noble `NNN` (consumes a `CLM` card). |
| `D N NNN CCC` | Give `NNN` the dignity of card `CCC` (`BAS`: bastard; `ARC`, `CTL`, `ABB`, `HRB`, `AST`, `EON`, `COR`, `ESP`, `EMP`, `SOR`: dignities of the ladies). |

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
- if `XXX`, the join's **origin**, is itself under attack — allied or enemy,
  whatever the outcome — the join is cancelled outright and the army stays on
  `XXX` (section 6, "Who Wins a Combat");
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
according to its liaison. An attack by another player on the supporting army,
from a territory different from the supported target, **cuts** the support,
even if that attack fails. A support is also cut if the supporting army is
dislodged.

### Hold (`H`) and Pillage (`P`)

`H XXX`: the army stays in place and can receive defensive support; mostly
useful to occupy a chain line while waiting, especially in a loop
(`(H XXX)`).

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
- as with a join, if one of your attacks wins the combat on a destination,
  the troop arrives there and fuses with the winning army;
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
  destination is processed, the order is invalid;
- if `XXX`, the dispersal's **origin**, is itself under attack — allied or
  enemy, whatever the outcome — the whole dispersal is cancelled outright,
  even a partial one: no troop leaves for any destination (section 6, "Who
  Wins a Combat").

```text
BRI D ATL ATL              # two troops stacked in the army arriving at ATL
BRI D ATL                  # one troop to ATL, the remainder stays on BRI
BRI D ATL*HUG NOR          # HUG to ATL, the other unit to NOR
BRI D BRI ATL NOR          # BRI keeps the chain; the other groups split away
(BRI D ATL NOR)            # looped dispersal
```

### Transfer (`T`)

**The gist**: `XXX T YYY N` is executed during order resolution, before the
end-of-turn supply phase (section 7), by the army on `XXX`. `YYY` must be a
castle, village, or the territory of an army that **controls** its own
territory and belongs to another living player (an army that merely occupies
`YYY`, for instance on a fief member it does not control, cannot receive); a
bare depot cannot receive. The source territory only needs to contain stock,
and must not be occupied against its controller.

**Edge cases**:

- the route follows the donor's supply range (`{{supply_range}}` territories,
  plus controlled depots); any enemy army on an intermediate territory blocks
  it, but an enemy army at the destination is allowed;
- a famished army (section 7) cannot send a transfer; the amount is capped at `{{cost_base}}^(N - 1)` for an army of `N`
  troops, without subtracting local rations; the army performs no other order
  that turn;
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
  even without an army, **as long as it stays anchored** — a fief member or a
  player's own capital — unless all attackers belong to the castle's owner
  (see self-capture, section 5); an empty castle that is neither a fief
  member nor a player's capital is **inert** and gives no bonus (see "Fiefs",
  section 8); a fief capital's castle is a **city** and gives
  **+{{city_defense_bonus}}** instead;
- the **strictly unique** highest strength wins; a top tie produces a
  **standoff**, including on an empty territory;
- you never dislodge your own army: an attack on a territory held by one of
  your armies that stays there fails, but it still blocks the other attacks
  from entering;
- the supports you give an opponent do not help them dislodge your own army:
  they only count to block the other attacks;
- **head-to-head**: when two armies attack each other, the stronger one
  (without the opposing player's supports) wins and dislodges the other; on a
  tie, or between two armies of the same player, neither moves;
- a repelled attack still blocks its destination for the other attacks, even
  if its army is dislodged, unless it lost a head-to-head;
- attacks in a circle (A to B, B to C, C to A) all succeed if nothing else
  opposes them;
- a join or a dispersal whose **origin** comes under attack — allied, enemy,
  or even a starving attack at zero strength — is **cancelled outright**: none
  of its troops leaves, whether that attack wins or loses the combat there.
  There is no fleeing through a join or a dispersal: leaving an attacked
  territory means surviving the combat fought over it.

That's exactly the calculation walked through in section 3: 3 against 2, no
tie, Hugues wins.

### Retreating

A dislodged army loses its movement and must retreat as a whole, to an
adjacent destination chosen by descending priority order:

1. an empty territory **anchored** to the retreating army's owner — a member
   of one of their fiefs, or their own capital (section 8) — even if fought
   over this turn;
2. any other empty territory that isn't anchored, has no **anchored** castle
   (anyone's), and wasn't fought over this turn — that's the case for NOR for
   Brune in section 3, an empty territory she only controls positionally and
   that wasn't fought over this turn; a territory merely controlled
   positionally, including the one the retreating army just left this same
   turn, falls back to this second priority like any other empty territory —
   only an anchor earns the first;
3. an adjacent, non-dislodged friendly army (smallest troop size first), with
   merging: the host gains `N − 1` troops if the retreating army has `N ≥ 2`
   troops, or `1` troop if `N = 1` (no loss). Multiple retreating armies can
   merge sequentially into the same friendly host without destructive
   collision.

Ties within a bucket are broken by distance to the nearest controlled castle
or village, then ascending trigram. For friendly armies, sorting is by troop
size ascending, then distance to the nearest controlled source, then
ascending trigram. The attacker's origin territory is always excluded. An
**anchored** castle — a member of a fief or a player's capital, even neutral
or enemy to the retreating army — defends against retreat and is never a
valid destination; an empty **inert** castle (neither fief, nor capital, nor
army) becomes a valid second-priority destination again, just like no castle
at all. Two armies that must retreat to the same empty territory with no
alternative are destroyed. Retreat resolution order follows the ascending
trigram of their origin territory.

Taking control of a territory follows the army that stops there; keeping it
depends on its **anchor**: a member of a fief, or a player's own capital — a
permanent exception, even without an army on it. Outside an anchor, control
is **ephemeral**: a territory stays "someone's" only while one of that
player's armies is currently stationed there; as soon as that stops being
true, it reverts to neutral (no controller) at the next control update,
until any army, whoever owns it, stops there again and retakes it
positionally. Within a fief (section 8), control is instead **transitive**: a
member other than its capital stays controlled by the fief's owner even when
an enemy army — or a revolt — stops there; it **occupies** the member without
controlling it. Only capturing the **capital** transfers control of every
member to the conqueror at once; a `NEUTRAL` revolt never takes control of a
territory, fief or not — so it never hands a release back to a former
non-fief controller. A cell occupied against its controller (fief or not) is
no longer a usable supply source or depot for anyone, and rejects any winter
investment aimed at it (see sections 7 and 8); it still keeps its defensive
bonus for the occupant, and its territory income keeps flowing to its normal
destination, never intercepted.

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
enemy army, those nobles are captured and become `hostage` by default (a bastard goes straight to the dungeon). A
hostage noble may continue to emit a chain (section 4); only moving it to the
dungeon, covered in section 8, removes that ability.

---

## 7. Logistics and Supply

### The Exponential Cost of an Army

Supply is resolved **at the end of every action season**, after orders,
combats, and movement, on the turn's final positions and territorial control
— territories just captured included; there is no supply phase in winter. An
army of `N` troops demands:

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
{{ration_terrain.mountain}}; swamp {{ration_terrain.swamp}}. A castle or
village adds no rations: only the terrain feeds an army locally. A bad harvest
removes every local ration of its region; an abundant harvest doubles them.

Example: a 2-troop army on a plain (local production
{{ration_terrain.plain}}) receives 2 rations, covering its full demand, castle
or not. The same army in a forest (production {{ration_terrain.forest}})
receives only 1 ration and must cover the rest elsewhere; in the mountains
(production {{ration_terrain.mountain}}), it depends entirely on supply.

**Supply sources**: controlled castles, villages, and caches, plus an isolated
mill (see below). A castle or village produces no stockable R by itself: its
contribution comes from the mills adjacent to it and from the territory
income it receives (see "Territory Income" below); a bare territory has no
production of its own, but its stock (if any) serves as a cache. The
flow crosses allied, neutral, or enemy-controlled territories, and only stops
before a territory occupied by an enemy army. A cell **occupied against its
controller** (section 6) — for instance a fief member held by an opponent who
never took control of it — is however no longer a usable source or depot for
anyone, controller or occupant. Outside any fief and outside a capital, a
castle or depot with no army on it is likewise **inert**: it then has no
controller left at all (section 6), and is no longer a usable source or depot
for anyone. Base range is {{supply_range}} territories; each controlled
supply depot, not occupied, encountered along the route adds
{{depot_range_bonus}} territories. A neutral village keeps its stock,
inaccessible before capture.

A mill outside any fief and outside a capital, with no army on it, is itself
**inert**: it produces nothing at all while it stays in that state, whether
it was never held or was just abandoned. An active mill — a member of a fief,
on a player's capital, or currently held by an army — of level `N` produces
`N` R and credits exactly **one** infrastructure: the adjacent castle **under
the same control as the mill's own territory**,
else the adjacent village under the same control, else the mill's own
territory. A castle or village adjacent to the mill but controlled by another
player is ignored. "Neutral" control is a controller like any other: a
neutral mill never feeds a player, only a neutral village adjacent to it,
else its own territory. A mill therefore never credits two infrastructures at
once. An isolated mill (no eligible castle or village of its own control)
produces on its own territory, which then becomes a source in its own right;
that production isn't automatically routed elsewhere — it still needs a
transfer order (`T`). The presence or position of a noble never conditions
this production.

### Territory Income

Each action season (never in winter), every territory you control yields
{{territory_income}} R, plus {{village_income}} R more if it carries a
village. This income is credited **at the end of the turn**, with the rest of
supply, on the turn's final territorial control: a territory captured during
the turn credits its new controller, not the one from the start of the turn.
It never travels through the supply network and can never be intercepted,
even when the producing territory — or its destination — is occupied by an
enemy army.

A territory that belongs to a **fief** (section 8) credits the **fief's
capital** instead of your own. Outside any fief, it goes straight to your
**capital**'s stock.

Without a designated capital (or right after it falls), each territory's
income goes to the closest controlled castle over crossable borders
(trigram tie-break), else the closest controlled village, else it is lost —
distinct territories can therefore feed different destinations the same turn
while no capital exists. A bad harvest suppresses this income in the region
of the territory producing it; an abundant harvest doubles it, exactly like
terrain rations. A **neutral** village keeps producing {{village_income}} R
per turn into its own stock, recovered on capture.

### Stocks and Famine

When there is a deficit:

1. stocks in controlled castles, villages, and caches are emptied first
   (smallest first, with the territorial trigram as tie-breaker);
2. remaining armies enter **famine**, starting with those furthest from their
   source, then the largest, then descending trigram.

An army that lacks rations at this end-of-turn resolution is marked
**famished**, a status that persists through the entire following turn: it
**attacks and defends at strength 0**, even when it carries a free noble — the
noble bonus does not apply — and it cannot send a resource transfer (it can
still receive one, see section 5). This first turn in deficit costs it
nothing else: no pillage of the infrastructure on its territory, no troop
loss. You therefore have the entire following turn to pull it out of deficit,
either by moving it away from the affected area or by sending it resources
through a transfer.

Its status is only recalculated at the next supply resolution, at the end of
that following turn. If it reached a sufficient source in the meantime, it
becomes valid again from that turn on. If it is still in deficit at that
point, while already famished, it **pillages the infrastructure on its
territory automatically**, if it occupies one: the pillage bonus, reduced by
its residual demand, may cover the deficit. If pillage is insufficient or
impossible, it loses **1 troop**, never falling below 1, and stays famished
for the turn after that.

Example: a 2-troop army in deficit demands 2 rations. Lacking sufficient
stocks, it ends the turn in deficit and is marked famished: it will attack and
defend at strength 0 for the entire following turn, but loses nothing for
now. If, at the following turn's resolution, its stocks and pillage still
cannot cover its deficit, it loses one troop, becomes a 1-troop army, and
stays famished for the turn after that — its fate depends on what it reaches
as a supply source by that resolution, not on its new demand.

In the interface, selecting an army or a controlled source shows its supply or
the area it reaches (outside winter only). A transfer being drafted also shows
its route and blockers.

### What Infrastructure Provides

A territory carries only **one infrastructure**; their cost and build
conditions are detailed in section 8.

| Infrastructure | v1 effect |
|---|---|
| Mill | Inert (no production) outside a fief/capital with no army; otherwise `N` stockable R per level, credited to a single adjacent infrastructure (castle, else village, else itself) |
| Supply depot | +{{depot_range_bonus}} territories of supply range while anchored or occupied; inert otherwise |
| Castle | +{{castle_defense_bonus}} defense while it stays anchored or occupied, inert otherwise; supply anchor; receives territory income (section 7); becomes a city (+{{city_defense_bonus}}, not stacked) on a fief's capital (section 8) |
| Village | Supply anchor after capture, receives territory income once controlled (produces {{village_income}} R per turn into its own stock while neutral, never held, or just abandoned — the only infrastructure that never goes inert) |
| Fortified village | Identical to a village (stock, production, income), and additionally gains +{{castle_defense_bonus}} defense while it stays anchored or occupied, inert otherwise (section 8) |

---

## 8. Winter Orders

Winter accepts **no chains or movement**: only direct investments, one order
per line, applied in the entered order.

| Investment | Syntax | Condition | Cost (R) |
|---|---|---|---|
| Draw a noble card | `T N` | once per winter and per player; the noble deck, shared by all players, must not be empty | 0 |
| Recruit a noble | `R N CCC XXX` | `CCC` is a noble card in your hand; `XXX` controlled, with a castle or village and a player army, and fewer living nobles owned than your cap, {{noble_limit}} at the base (see below) | 0 |
| Claim titles | `C N HHH CCC` | you hold a claim card (`CLM`); `HHH` is one of your nobles, placed during a marriage of `CCC` with one of your nobles, not a bastard and without a current claim; `CCC` is a noble of another player | 0 |
| Grant a dignity | `D N NNN CCC` | `NNN` is any noble, yours or an opponent's; `CCC` is a dignity card in your hand (`BAS`: bastard; `ARC`, `CTL`, `ABB`, `HRB`, `AST`, `EON`, `COR`, `ESP`, `EMP`, `SOR`: ladies; Blocked ones: unmarried only); a noble carries a dignity only once, and a lady carries a single lady dignity (`dignity_exclusive`) | 0 |
| Discard a noble card | `D C CCC` | `CCC` is a card in your noble hand (noble trigram or `BAS`); no limit per winter; the card goes to the deck's discard pile | 0 |
| Recruit a troop | `R T XXX` | `XXX` controlled, and a free player noble on `XXX` or adjacent | {{costs.troop}} |
| Build or upgrade a mill | `C M XXX` | `XXX` controlled; a **new** mill requires an **empty** territory adjacent to a castle or village, or itself carrying one; an **existing** mill can always be upgraded, even in isolation | {{costs.mill_levels.0}} (L1), {{costs.mill_levels.1}} (L2), {{costs.mill_levels.2}} (L3) |
| Build a castle, or fortify a village | `C C XXX` | `XXX` controlled; on a village, fortifies it instead of building a castle there; rejected with no stock deducted if the village is already fortified | {{costs.castle}} |
| Build a supply depot | `C D XXX` | `XXX` controlled | {{costs.supply_depot}} |
| Designate a capital | `E C XXX` | a controlled castle on `XXX` | 0 |
| Place a noble in hostage status | `O N NNN` | `NNN` is an opposing prisoner held by the player | 0 |
| Place a noble in the dungeon | `P N NNN` | `NNN` is an opposing prisoner held by the player | 0 |
| Hand a noble to another player | `H N NNN XXX [O\|P]` | `NNN` is a free noble of the player, or a hostage or prisoner they hold; `XXX` holds another player's army. The noble keeps its status with that player (a free noble becomes a hostage), which `O` (hostage) or `P` (dungeon) can set; if that player is its owner, it is freed on `XXX` | 0 |
| Transfer resources | `G XXX YYY N` | `XXX` is a castle or village controlled by the donor; `YYY` is a castle or village controlled by another player | 0 |
| Found a fief | `T F NNN XXX YYY ZZZ …` | `NNN` is a player noble, even a hostage or prisoner; `XXX` (capital) and the rest of the group are controlled, contiguous, and need no castle outside the capital; no territory already in a fief; no enemy or revolt army on the group | {{costs.fief_per_territory}} per territory |
| Assign a vacant fief | `T A NNN XXX` | `NNN` is a player noble, even a hostage or prisoner; `XXX` is the capital of a vacant fief the player holds | 0 |
| Marry two nobles | `M N NNN MMM` | `NNN` is a free player noble, `MMM` a free noble of another player, of the opposite sex, both unmarried; the other player must submit `M N MMM NNN` the same winter, otherwise the marriage is refused | 0 |

**The noble deck.** Nobles are not bought: they are drawn from a single
**noble deck**, shared by all players, shuffled when the game is
created. It holds **noble cards** (a first name, its trigram and its sex, as
many men as women) and **dignity cards**, at least one per game. Each winter,
`T N` adds the top card of the deck to your hand, once per player (beyond
that, or when the draw pile and the discard pile are both empty, the order is
rejected). Your hand is limited to **{{special_orders.hand_limit}} cards**,
noble, dignity and special-order cards together: `T N` is rejected when it is
full. You draw at most **{{special_orders.draw_orders_limit}} cards per
winter**, across all decks, of which only one from the noble deck. `R N CCC XXX` plays
the noble card `CCC` from your hand: the noble appears on `XXX` at no cost, as
long as you own fewer living nobles (free, hostage or in the dungeon) than
your cap, {{noble_limit}} at the base. A card that is not in your hand is
rejected. `D C CCC` discards a card from your noble hand without playing it: it joins the deck's
discard pile, frees a slot in your hand (a `T N` placed lower in the same sheet can use it) and is
not named in the public report.

A played card stays on the noble it brought into play or that carries its
dignity. When that noble dies, its noble card leaves the game and a new noble
card of the same sex, with a first name not yet used, joins the discard pile
(none if no first name is left); the dead noble's trigram is never reused. A
dignity card returns to the discard pile when its carrier dies or loses the
dignity. When the draw pile is empty, the discard pile is shuffled to rebuild
it.

Noble (`R N`), claim (`C N`) and dignity (`D N`) cards can be played in any season, action or
winter; `T N` and `D C` remain winter orders.

A **claim**, which consumes a claim card (`CLM`), lets one of your nobles, the heir `HHH`, claim the titles of a noble
`CCC` of another player with `C N HHH CCC`, provided `HHH` was placed during a
marriage between `CCC` and one of your nobles. An heir claims a single noble and
cannot be a bastard. Claims stack:
they rank from oldest to newest, and within the same winter the wife's
family comes first. When `CCC` dies, the fiefs `CCC` holds pass to the first
living heir in the ranking, with their territories, whatever its rank in your line of
succession; with no fief or no living heir, the claim lapses. A claim is public: every player sees it in the report. A bastard card played on the heir
cancels its claim; played on a parent noble, it does not.

A dignity card is played on any noble, yours or an opponent's, secret ones
included, with `D N NNN CCC`, including a
noble you just recruited with an `R N` placed earlier in the same sheet. A noble
carries a given dignity only once. The deck holds the **bastard** (`BAS`), open
to any noble, and one card for each dignity of the ladies (see below). A bastard:

- raises your noble cap by 1 while alive, even when married, hostage or in
  the dungeon; each bastard adds 1, never beyond {{noble_limit_max}};
- is always last in your line of succession and receives a new fief only if it
  is the last of your lineage; a fief it already holds stays with it;
- cannot be king;
- may marry, but its marriage is not an alliance (no weight, no category, no
  shared score);
- goes straight to the dungeon, never as a hostage, when captured in combat;
- counts as a title in your score.

The dignities of the ladies are permanent and each counts as a title in your
score. They can only be played on a lady (`dignity_female_only`), and a lady carries only one (`dignity_exclusive`); the bastard stacks with it. A **Blocked**
dignity cannot be played on a married lady (`noble_married`) and forbids
marriage afterwards; a **Free** dignity can be played on a married or unmarried
lady and leaves marriage open. Their bonuses stop while the lady is in the
dungeon; a hostage lady keeps them and her captor profits too (passive bonuses only: only her owner gives her orders). **Hidden**
dignities are known to your player only: other players see a lady without a
dignity, in neither the state view nor the report.

| Dignity | Code | Marriage | Effect |
|---|---|---|---|
| D'Arc | `ARC` | Blocked | `+1` force to the army she commands, on top of the noble command bonus of `+1`. |
| Castellan | `CTL` | Free | In a castle, you learn the orders issued this turn on her fief, including those targeting enemy armies. An army coming from outside that enters without changing its chain stays hidden. |
| Abbess | `ABB` | Blocked | Played with the village seed of a region (`D N NNN ABB TER`), for good. While she is in it, you know the order chains issued in that region. |
| Herbalist | `HRB` | Blocked | Your troops and nobles on her territory and the adjacent ones are immune to plague; your army on her territory consumes 2 fewer rations per turn, never below zero. |
| Astrologer | `AST` | Free | In winter, you privately see the next 3 calamities of the special-orders draw pile. `V C NNN I` strikes one (position 1 to 3), once per winter per astrologer, by her owner only; the following calamities take their place. |
| Chevalier d'Éon (hidden) | `EON` | Free, unmarried lady | The lady is replaced by a male noble, with a first name and code drawn from the unused names and all the prerogatives of a man (so he marries a woman). Her identity as a lady stays secret: only you know it, until she is unmasked — captured in combat, imprisoned, or married and targeted by a Claim (the targeted lord then becomes a bastard, the marriage is annulled, and the Claim is lost). Once unmasked, her identity and dignity are visible to everyone. |
| Correspondent (hidden) | `COR` | Free | Hostage of a player, she lets you see every order that player issues. |
| Spy (hidden) | `ESP` | Free | Hostage of a player, she reveals their whole hand (special orders, nobles and dignities). |
| Poisoner (hidden) | `EMP` | Free | Every army of another player in her region consumes 1 more ration per turn, with no explanation for its owner. |
| Witch (hidden) | `SOR` | Free | In winter, `S R NNN CAL` calls a calamity (`PE`, `MT` or `FA`): if it is drawn for next year it strikes the region where the Witch stands, otherwise the ritual fails. `S R NNN N` fixes the season (`1` spring, `2` summer, `3` autumn) of the first calamity drawn, which then strikes the Witch's region. One ritual per winter, either one; other players see a slightly unusual calamity announcement, without knowing why. No protection against plague. Once revealed, she is excommunicated ex officio. |

This is where, in winter, the fate of enemy nobles captured in combat
(section 6) is decided: `O`/`P` moves a prisoner between `hostage` and
`dungeon`; a hostage noble may still emit a chain for its original owner while
it remains a hostage, but no longer once in the dungeon — the holder can in
fact read those chains in online games, even when they command an army that
stayed with the noble's owner. Capture normally produces `hostage` status,
except for a bastard (see "The noble deck" above), who goes straight to the dungeon.
`H N NNN XXX` hands a noble you control to another player's army standing on
`XXX`. It lets you voluntarily send one of your free nobles as a hostage (its
host then sees its chains and benefits from the passive bonuses of a lady),
free a prisoner by sending it back to its owner's army (it is then free on
`XXX`, at no cost), or pass a hostage on to
another player. Only the holder of a hostage can send it back or pass it on.

A winter transfer is therefore not limited to the donor's own castles and
villages: `G` can directly supply a structure controlled by the recipient. The
debit still follows the usual rules and can use only the donor's payment
reserves.

A mill starts at level 1 and can reach level 3 inclusive. Construction costs
{{costs.mill_levels.0}} R; upgrades to levels 2 and 3 cost
{{costs.mill_levels.1}} R and {{costs.mill_levels.2}} R respectively. `C M` on
a level-3 mill is rejected with reason `mill_max_level_reached`, with no stock
deducted. Mills above level 3 already present in a game are preserved and
remain productive; only new upgrades are blocked. The adjacency requirement
(an empty territory next to a castle or village) only applies to
**building** a new mill; an existing mill can always be upgraded, even in
isolation, paying from its own stock (see "Resource Vocabulary" below).

Investments targeting a territory require **control of that territory** and
that it not be **occupied against its controller** (section 6): an enemy
army — or a revolt — stationed there rejects the order with no stock
deducted. `C C` on a village **fortifies** it for the cost of a castle,
instead of replacing it: the fortified village keeps its stock, production,
and income, and additionally gains a castle's defensive bonus (see "What Infrastructure
Provides" above). `C C` on an already-fortified village is
rejected with no stock deducted. An isolated mill (no castle or village of
its own control adjacent) produces on its own territory (see section 7) and
can always be upgraded.

### Fiefs

`T F` founds a fief: a group of at least 3 controlled, contiguous
territories, whose first entry is the **capital** (only it needs a castle).
The title depends on the group's size: barony (3), county (4), marquisate
(5), duchy (6 or more). The capital's castle becomes a **city** and provides
**+{{city_defense_bonus}}** defense in total (replacing the usual castle
bonus, not stacking with it). The title belongs to the designated titulaire
noble, who must be free at the time of founding; a single noble may hold
several titles, and a player may hold several fiefs.

A fief's titulaire is addressed by form of address (shown in French) —
Baron/Baronne (barony), Comte/Comtesse (county), Marquis/Marquise
(marquisate), Duc/Duchesse (duchy) — followed by their first name and the fief's capital;
a noble without a fief is a "Sieur" or a "Dame". Their spouse (marriage in
force) bears the matching courtesy title, whatever their sex, for as long as
the titulaire keeps the fief. It is display only: it has no effect on score,
alliances or succession.

Control of a fief is **transitive** (section 6): a member other than the
capital stays yours even when an enemy army stops there; it **occupies** the
member without taking it from you. Only capturing the **capital** costs you
the entire fief, every member at once.

If the titulaire dies (plague) or the capital changes hands, the fief becomes
**vacant**: it keeps producing and scoring its point, but has no titulaire.
`T A` then assigns it to a noble of the player who holds it, provided
every noble ahead of them in your line of succession (the order in which you
recruited them, bastards always counting last) already holds a fief of an equal or higher title; founding a
fief follows the same rule. At the end of winter, a fief still vacant at that
point is **automatically assigned** to the first noble in your line of
succession, even a hostage or prisoner, with a warning in the report telling
you to take back manual assignment next turn; with no living noble at all, it
simply stays vacant — it is never dissolved for lack of assignment.
If the capital's castle is destroyed (pillage, including
automatic famine pillage), the fief is dissolved **immediately**, regardless
of the season: this is the only way a fief is dissolved. Capturing the
titulaire (hostage or dungeon), by contrast, has no effect on the fief.

### Resource Vocabulary

- `R` means one unit of **stockable resource**: it sits in a territory's
  stock, is produced by a source, and pays for investments when held by a
  controlled castle or village;
- a **ration** is one food unit consumed during an action-season supply phase
  (section 7); local rations do not automatically become stock `R`;
- **stock** is therefore the amount of `R` kept on a territory.

Each controlled castle or village is a separate source, and any controlled
territory with positive stock is an action-season cache source, as is an
isolated mill (see section 7): a second castle is therefore a second source,
even though only one castle is designated as the capital. Its stock depends
on the territory income it receives (section 7, if it is the capital or its
fallback) and on the mills adjacent to it under the same control — see
section 7 for the details of this production.

**Payment**: the cost is taken first from the stock on the target territory,
then from the nearest controlled source; if the total reserve is
insufficient, **no partial payment** is made and the investment is rejected
(reported, with no cost lost). A settlement or mill occupied against its
controller is never part of these reserves. Upgrading a mill (`C M ATL`) is
the exception
to this order: it first consumes the mill's own stock, then the stock of the
infrastructure that would receive its production (castle first, else
village), before falling back to the usual payment network; if those stocks
do not total the required cost, the upgrade is rejected with no partial
payment made.

**End of winter**:

- each remaining castle, village, or mill stock is kept at
  `ceil(stock / {{winter_stock_divisor}})` — a stock of 5 R therefore becomes
  3 R;
- a supply depot keeps its stock in full; stock outside a castle, village,
  mill, or depot is lost;
- castle and village stocks outside the capital are brought back to the
  capital, leaving at most {{village_stock_cap}} R per village and
  {{castle_stock_cap}} R per castle; a mill's stock is **never** repatriated,
  and neither is a settlement's stock while it is occupied against its
  controller (it stays there and follows normal conservation instead);
- without a capital, those stocks remain where they are; depot stock remains
  on its territory.

There is no need to spend everything before winter ends: unspent stock is
first conserved, then surplus is repatriated under these caps. Conservation
and repatriation happen after investments.

**Prosperity and exodus**: right after conservation, the total stock loss
that just occurred — summed across the whole map, every player combined —
can found new villages: every full {{prosperity_loss_threshold}} R lost
triggers one founding. Territories that lost stock this winter are ranked by
loss descending (trigram ascending on ties), and each triggered founding
comes from the next territory in that ranking, without ever degrading it or
taking more stock from it than normal conservation already did. The founding
lands on the closest free tile to that origin territory, not adjacent to an
existing village or castle, in priority order: inside a fief of the player
who controls the origin territory, else land that player controls, else
anywhere free at all, including neutral or another player's; if no tile
qualifies at any level, a supply depot becomes a village instead, or failing
that a mill. The founded village belongs to the controller of the arrival
tile, with no necessary link to the player whose origin territory triggered
the founding.

---

## 9. Special Cards and Calamities

Playable card orders are submitted in a separate `special` field, distinct
from noble chains and requiring no noble. Winter discards are written in the
`winter` sheet.

- `P FW ROS`: play Fair weather on the region seeded by ROS;
- `P AH ROS`: play Abundant harvest on that region;
- `P RV BRU`: play Revolt on the BRU territory, when an active bad harvest
  affects its region, or when a seigneurial tax was played on the capital of
  BRU's fief this turn or the previous one — every territory of the fief is
  then eligible, not only its taxed capital, or when a trial executed a lady
  in BRU's region during the previous action season;
- `P TR NNN`: play Trial on the noble with code NNN — an opponent's or not —,
  judged at the end of the turn (see below);
- `P TX BRU`: play the Seigneurial tax on BRU, provided BRU is the capital of
  a fief the player controls (a vacant fief included) — the one exception
  where the target is not a region's seed village;
- `D C FW` or `D C AH`: discard a card, winter only.

The hand is replenished automatically in winter after winter orders and discards; no draw order
is needed.

Fair weather, Abundant harvest, Revolt, the Seigneurial tax, and Trial can be played
in spring, summer, and autumn, but not winter. Played cards are consumed
before army-order resolution. Fair weather cancels only bad weather, and
Abundant harvest cancels only bad harvest; a card that cancels a calamity does
not provide its regional bonus. Duplicate cards of the same kind are
consumed, but only one is effective: with an active calamity the first card
cancels and a second one applies the regional bonus; without a calamity the
first card applies it directly. The bonus applies only once per kind and
region; further cards are consumed without effect. The Seigneurial tax
follows a separate rule, per fief rather than per region: two cards played on
the same fief the same turn never stack, the second is simply consumed
without effect.

Regional bonuses:

- Fair weather **doubles** the production of the region's mills;
- Abundant harvest **doubles** the terrain rations of every territory of the
  region and the region's territory income;
- the Seigneurial tax **doubles** the targeted fief's territorial income,
  village included, for the turn — it never touches mill production.

The deck contains **{{special_orders.deck_size}} cards**:
**{{special_orders.card.plague}}** plague, **{{special_orders.card.bad_weather}}**
bad weather, **{{special_orders.card.famine}}** bad harvest,
**{{special_orders.card.fair_weather}}** fair weather,
**{{special_orders.card.abundant_harvest}}** abundant harvest,
**{{special_orders.card.revolt}}** revolt,
**{{special_orders.card.seigneurial_tax}}** seigneurial tax, and
**{{special_orders.card.trial}}** trial cards. A hand is limited to
**{{special_orders.hand_limit}} cards**, noble and dignity cards included.
After their winter orders and discards, each player automatically receives
bonus cards, up to **{{special_orders.draw_orders_limit}} per winter** across
all decks (a noble card drawn counts as one) and within the free slots of
their hand.

A drawn calamity is programmed into the following year, on a season drawn at
random among those that still have a free slot: spring
(**{{special_orders.calamity_slots.spring}}**), summer
(**{{special_orders.calamity_slots.summer}}**), or autumn
(**{{special_orders.calamity_slots.autumn}}**). Its season and region are
selected deterministically when programmed. The spring augury reveals the kind, season,
and region of every calamity in that year; future auguries remain hidden. As
soon as a calamity is drawn, the interface announces it in the special-cards
panel, and the announcement stays visible until the calamity applies or is
countered. No calamity resolves in winter.

- plague reduces armies by a divisor of
  **{{special_orders.effects.plague_army_divisor}}** and may remove a noble;
- bad weather blocks movements originating from or targeting its region,
  except holds and defensive support, and the region's mills produce nothing;
- bad harvest removes the terrain rations of every territory of its region and
  the region's territory income;
- Revolt is played on a territory (`P RV TER`) during action seasons,
  provided its region suffers a bad harvest, or a Seigneurial tax was played
  on the capital of the territory's fief this turn or the previous one —
  every territory of the fief is then eligible, not only its taxed capital —
  or a Trial executed a lady in its region: every territory of that region is
  then eligible during the action season that follows the execution. Each card adds a roll between
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
  the local production of their territory cannot feed them;
- the Seigneurial tax is played on a fief's capital (`P TX XXX`) the player
  controls, a vacant fief included. It doubles the fief's territorial income
  for the turn, village included, and never touches mill production.
  Rejected if the player does not control the targeted fief, or if XXX is
  not its capital. Two cards played on the same fief the same turn do not
  stack: the second is consumed with no effect. If the taxed fief's capital is
  captured during that same turn, the tax is canceled: nobody receives the
  doubling for that transition turn;
- Trial is played on a noble (`P TR NNN`), yours or an opponent's, in spring,
  summer, or autumn. It is judged at the very end of the turn, once every
  other effect has resolved. Only an **unmarried lady carrying a dignity**,
  hidden or not, other than the Abbess, can be tried: the trial **reveals** her
  dignity and she is **executed**, gone for good, and Revolt becomes playable on any territory of the region
  where she stood during the following action season. Any other trial is
  unfounded: the card is consumed and nothing else happens.

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
| Held fief | 1, vacant included, until dissolved |

Infrastructure and resources only score on a controlled territory. A free
noble counts for its owner. A captured noble, hostage or in the dungeon,
counts for the player whose army physically holds it — the one stationed on
its territory —, not for that territory's controller nor for its original
owner: outside a fief, territorial control is ephemeral (section 6) and may
have vanished while the capturing army still stands there.
