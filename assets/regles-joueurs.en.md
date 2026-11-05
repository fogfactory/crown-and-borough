# Game Rules — Crown & Borough

Crown & Borough is a turn-based medieval strategy game played on a map of
territories. Everyone writes their orders in secret, then all of them are
resolved **at the same time**, season after season. The numbers quoted here
come from the server configuration; when in doubt, the engine is the
reference.

---

## Everything you need to win

![The map and the command post in spring](/rules/en/overview.webp)

- **You win with titles.** Each fief, the crown and the papacy is worth
  1 influence point. Territories, armies and gold score nothing by
  themselves: they are how you found and hold fiefs.
- **Reach the goal before the end.** The game stops as soon as a player, or
  an allied couple, reaches their influence goal. Otherwise the best score
  in the last year wins.
- **Your nobles give the orders.** Each noble writes a single order chain per
  season for one of your armies. The more nobles you have, the more you can
  do.
- **A big army is very expensive.** Its food cost doubles with every extra
  troop, and an army cut off from its supply goes famished.
- **Winter is for building.** There is no fighting in winter: you build,
  found fiefs, marry your nobles and elect bishops, popes and kings.
- **Marriage is a weapon.** A head marriage joins two houses for a shared
  victory; your other marriages bring you your spouse's influence.

---

## 1. A year in four seasons

A game lasts 1 to 50 years (10 by default). Each year has four turns:
**spring, summer, autumn** (the action seasons) and **winter**. The interface
shows the historical year `1000 + year`.

**Setup.** An online game seats 2 to 8 players (up to 16 locally). Each
player starts with:

- a **capital**: a castle on a non-mountain territory, with
  {{starting_resources}} R in stock and an army of {{starting_troops}}
  troops;
- {{starting_nobles}} free nobles;
- {{starting_outposts}} one-troop outposts on neighboring territories.

**An action season** goes like this:

1. you write one order chain per noble, and play your cards;
2. the engine resolves everyone together: supports, combats, moves, retreats
   and transfers;
3. territory control is updated;
4. supply is computed on these final positions: income, production, rations
   and famine;
5. everyone receives their turn report.

**Winter** is a truce: no chains, no moves, no combat. You invest, and
manage your nobles and titles (section 5).

**What is public, what is secret.** The map, armies, stocks, infrastructure
and noble positions are visible to everyone. Your **intentions** stay
secret: online, only the participants in a chain or a combat see its
details. Everyone else sees only the outcome.

> Appendix D walks through a full turn on an example.

---

## 2. Commanding: nobles and chains

### The order chain

A **chain** is a sequence of orders written in advance for **a whole army**.
Each free or hostage noble issues **one per season**, for any army of their
player, even a distant one. The interface adds the noble's name at the top
for you: you only write the order lines, one per line, in the form
`POSITION SYMBOL [targets]`.

![Two chains being written, and their arrows on the map](/rules/en/chain.webp)

- **One line per season.** An army carries out a single line each action
  season: with `ROS A BOI` then `BOI A ATL`, it reaches BOI this season and
  ATL the next. The chain stays attached to the army until it ends.
- **single or loop.** A plain line breaks the chain if it fails. A line in
  parentheses, `(…)`, is retried every season until it succeeds.
- **A new chain replaces the old one.** If two of your nobles target the
  same army in the same season, neither chain is received.
- **Without a chain, the army has "No Orders".** It stays put and defends
  normally.
- **Errors are refused up front.** A faulty submission (syntax, territories
  that are not adjacent, and so on) is rejected whole, with the line to fix.
  The interface flags these while you type.

### The orders

| Order | Syntax | Effect |
|---|---|---|
| Attack | `XXX A YYY` | Enters adjacent `YYY`, fighting if needed. |
| Defensive support | `XXX S YYY` | Strengthens the army that stays on `YYY`. |
| Offensive support | `XXX S YYY - ZZZ` | Strengthens the attack from `YYY` to `ZZZ`. |
| Hold | `H XXX` | Stays put. Useful for waiting: `(H XXX)`. |
| Join | `XXX J YYY` | Peacefully joins `YYY`; last order of the chain. |
| Pillage | `P XXX` | Degrades the territory's infrastructure and yields {{pillage_bonus}} R. |
| Dispersal | `XXX D DEST1 DEST2 …` | Sends one troop to each destination, in order. |
| Transfer | `XXX T YYY N` | Sends `N` R through the supply network. |

What to remember:

- **Attack and join** target a neighboring territory across a passable
  border. A join never fights: it fails if the territory is contested.
- **A support** strengthens an army of any player. A defensive support only
  counts if the supported army stays on its territory and does not attack.
  An attack on the supporting army, coming from anywhere but the target,
  **cuts** the support.
- **A dispersal** splits an army without fighting. Troops that cannot leave
  stay behind, and the chain follows the first group listed.
- **A transfer** sends resources to a castle, a village or another player's
  army within supply range. An enemy army on the way blocks it. An army that
  transfers does nothing else that turn.
- **A join or a dispersal is cancelled** if its starting territory is
  attacked. You cannot flee a combat.

The details (nobles in a dispersal, transfer caps, edge cases) are in
appendix C.

---

## 3. Combat

### Who wins

A territory never holds more than one army. Its **strength** is:

- its **size** in troops;
- **+{{noble_command_bonus}}** if one of your free nobles stands on its
  territory;
- **+ each support**, worth the size of the supporting army (with the same
  noble bonus);
- in defense only, **+{{castle_defense_bonus}}** for an active castle or
  fortified village, and **+{{city_defense_bonus}}** for a city, the castle
  of a fief capital.

**The strongest wins, provided it is alone at the top.** A tie at the top is
a standoff: nobody gets in.

- Two armies attacking each other: the stronger dislodges the other.
- Attacks in a circle (A→B, B→C, C→A) all succeed if nothing stands in the
  way.
- You never dislodge your own army, and supports you give an opponent never
  help them against you.

### Retreating

The beaten army falls back whole to a neighboring territory, in this order
of preference:

1. an empty territory **anchored** to you: a member of one of your fiefs, or
   your capital;
2. another empty territory, with no active castle, that was not attacked
   this turn;
3. one of your neighboring armies, which absorbs it. Armies that are not
   famished come first, then the smallest.

With nowhere to go, the army is **destroyed**. Its nobles are then captured
by the enemy army there, and become its hostages.

> **Tip.** A fief always offers a safe fallback, even on a territory fought
> over this turn. The full tie-breaks are in appendix C.

### Control and occupation

- **Outside a fief, control is ephemeral.** A territory is yours as long as
  one of your armies stands on it. As soon as it leaves, the territory turns
  neutral again.
- **A fief is solid.** Its members stay yours even if an enemy army stops
  there: it **occupies** them without taking them. Only taking the **fief's
  capital** brings down the whole fief at once.
- **A territory occupied by the enemy** serves neither as a source nor as a
  depot, and accepts no winter investment. Its income still comes to you.
- **Your capital** stays yours, even when empty.

### Nobles in combat

Nobles follow their army in all its moves. They count neither in losses nor
in supply. The command bonus only comes from a noble **present** on the
territory: commanding from afar gives none.

---

## 4. Feeding your armies

### The cost of an army

Supply is computed **at the end of every action season**, never in winter.
An army of `N` troops needs `{{cost_base}}^(N − 1)` rations:

| Size | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Rations | {{army_cost.1}} | {{army_cost.2}} | {{army_cost.3}} | {{army_cost.4}} | {{army_cost.5}} |

### Where the food comes from

1. **The territory's terrain** feeds the army standing on it: plain
   {{ration_terrain.plain}}, forest {{ration_terrain.forest}}, hill
   {{ration_terrain.hill}}, mountain {{ration_terrain.mountain}}, swamp
   {{ration_terrain.swamp}}.
2. **The rest comes from your sources**: your castles, your villages, your
   caches (a territory that keeps a stock) and your isolated mills. They feed
   armies within **{{supply_range}} territories**. Each of your supply depots
   on the way adds {{depot_range_bonus}} territories. Only an enemy army
   stops the flow.

![A territory card: resources, army, nobles present](/rules/en/territory.webp)

Select an army or a source to see its supply line or zone. The treasury
summary shows the projected famine risk.

### Income and production

- **Territory income.** Each controlled territory yields
  {{territory_income}} R per action season, plus {{village_income}} R per
  village level. This income goes to its fief's capital, otherwise to your
  capital. It is never intercepted.
- **Mill.** A level `N` mill produces `N` R per action season. It pours its
  output into your neighboring castle, otherwise your neighboring village,
  otherwise keeps it on its own territory.
- **Village.** A village has a level from 0 (ruin) to
  {{village_max_level}}. While neutral, it fills its own stock, which you
  collect by taking it.

### Famine

An army short of rations is **famished** until the next supply. Its
**supplied value** `s` is the largest size it could have fed. A famished
army:

- defends at strength `s − 1`, without noble bonus;
- **cannot attack, support or transfer** (it can receive a transfer);
- can be targeted by a Revolt card (section 8).

If it is still hungry at the next supply, it first pillages the
infrastructure on its territory. If that is not enough, it **loses troops**
until it is back down to `s`. At `s = 0`, it disappears and its nobles die
with it.

> **Good to know.** You always have one full season to react: move the army
> closer to a source, split it, or send it food. The groups a dispersal
> creates stay famished until the next supply, but each one, being smaller,
> is easier to feed.

### Infrastructure and pillage

A territory carries a single infrastructure. A castle, a fortified village
and a supply depot only work while **active**: members of a fief, a player's
capital, or held by an army. Otherwise they are **inert**.

| Infrastructure | Role | If pillaged |
|---|---|---|
| Castle | Defense +{{castle_defense_bonus}}, source, receives income | Falls into ruin |
| Village | Source, income +{{village_income}} R per level | Loses 1 level (0 = ruin) |
| Fortified village | Like a village, plus a castle's defense | Its wall falls |
| Mill | `N` R per season | Loses 1 level, disappears at level 1 |
| Supply depot | Range +{{depot_range_bonus}} | Disappears |

A ruin cannot be pillaged and produces nothing. The stock of a pillaged
infrastructure is added to the pillager's loot.

---

## 5. Winter: build and invest

![The winter orders panel](/rules/en/winter.webp)

In winter, you fill in a sheet of direct orders, without chains. The main
investments:

| To… | Order | Cost |
|---|---|---|
| Recruit a troop | `R T XXX` | {{costs.troop}} R |
| Build or upgrade a mill | `C M XXX` | {{costs.mill_levels.0}} / {{costs.mill_levels.1}} / {{costs.mill_levels.2}} R |
| Build a castle, or fortify a village | `C C XXX` | {{costs.castle}} R |
| Build a supply depot | `C D XXX` | {{costs.supply_depot}} R |
| Found a fief | `T F NNN XXX YYY ZZZ…` | {{costs.fief_per_territory}} R per territory |
| Annex a territory to a fief | `T X NNN XXX YYY` | {{costs.annex_per_territory}} R |
| Raise a fief by one rank | `T R NNN XXX` | {{costs.rank_per_territory}} R per territory |
| Designate your capital | `E C XXX` | free |
| Send R to another player | `G XXX YYY N` | free |

The full list is in appendix B. What you need to know:

- **You pay on the spot, then from the nearest.** The cost is taken from the
  stock of the target territory, then from your nearest sources. If the total
  falls short, the order is refused and nothing is taken.
- **You only build at home.** The territory must be controlled by you, and
  not occupied by the enemy.
- **Stocks melt in winter.** Each castle, village and mill keeps only its
  stock divided by {{winter_stock_divisor}}, rounded up (5 R becomes 3 R).
  Depots keep everything. What remains is then brought back to your capital,
  leaving at most {{village_stock_cap}} R in each village and
  {{castle_stock_cap}} R in each castle.
- **Prosperity is born from losses.** Every {{prosperity_loss_threshold}} R
  lost over the winter, across the whole map, founds or grows a village near
  where the stock was lost.

Winter orders resolve in steps, in a fixed order: the pope's sanctions,
royal orders, management, inquiries and filed trials, dissolutions,
marriages, elections, investiture, end of winter, then judgments and
assassinations. The details are in appendix B.

---

## 6. Nobles, fiefs and the crown

### The noble deck

Nobles are drawn from a **deck shared** by all players, which mixes noble
cards (a first name, a sex) and dignity cards.

- **Draw**: `T N`, once per winter.
- **Recruit**: `R N CCC XXX` brings the noble of card `CCC` onto `XXX`, which
  must be a castle or village of yours, with one of your armies. The noble
  acts from the next season on.
- **Cap**: you own at most {{noble_limit}} living nobles. Each bastard raises
  this cap by 1, up to {{noble_limit_max}}.
- **Hand**: {{special_orders.hand_limit}} cards at most, all decks combined.
  `D C CCC` discards a card.

Noble cards (`R N`), claim cards (`C N`) and dignity cards (`D N`) can be
played in any season.

### Hostages and prisoners

A captured noble becomes a **hostage** of the army that took them. A bastard
goes straight to the **dungeon**. In winter, the captor can switch a
prisoner between hostage and dungeon (`O N` / `P N`), hand them over or
return them (`H N`).

- A **hostage** can still issue a chain for their player, but their captor
  reads it.
- In the **dungeon**, they command nothing, and their dignities and religious
  titles are suspended.

### Fiefs

A **fief** is founded in winter with `T F`: at least 3 controlled,
contiguous territories, the first of which, the **capital**, carries a
castle. Its title follows its size: **barony** (3), **county** (4),
**marquisate** (5), **duchy** (6 or more). The capital's castle becomes a
**city**.

- A fief is worth **1 influence point**, whatever its rank.
- Its members stay yours even when occupied. Only taking its capital brings
  it down (section 3).
- You expand it with `T X` and raise it one rank with `T R`, once per
  winter.
- If its holder dies or the capital changes hands, the fief becomes
  **vacant**. It still produces and still counts. Name a new holder with
  `T A`, otherwise the first noble of your line receives it at the end of
  winter.

**The line of succession** is the order in which you recruited your nobles,
bastards always last. A noble only receives a title if the nobles **of the
same sex** ahead of them already hold a title at least as high. Clergy block
no one.

### Marriages and alliances

![The lineage and alliances tree](/rules/en/lineage.webp)

Two free, unmarried nobles of different sexes and of two different players
marry in winter if **each player** writes the order: `M N NNN MMM` on one
side, `M N MMM NNN` on the other. Without the matching order, the marriage
is refused.

Each marriage has a **weight**: that of the weaker spouse, based on their
rank in the line and their title, plus 1 for each other marriage between the
same two houses.

- **The head marriage** is the heaviest marriage for both houses. It forms a
  **full alliance**: the two scores add up toward a **shared victory**, but
  neither house can win alone.
- **The other marriages** are **influence marriages**: each house gains as
  many points as the other house has titles.
- A bastard's marriage is not an alliance.

The **Lineage and alliances** view shows the families, marriages and claims,
and offers a victory simulator.

### Claims

One of your nobles recruited during a marriage between one of yours and
another player's noble `CCC` can claim `CCC`'s titles: `C N HHH CCC`, with a
claim card (`CLM`). When `CCC` dies, their fiefs pass to the first living
heir, from the oldest Claim to the most recent. A Claim is public.

### Dignities

A dignity card is played with `D N NNN CCC` on any noble, yours or someone
else's. A dignity is worth no influence point: its value lies in its effects.

- **The bastard** (`BAS`) is played on any noble. It raises your noble cap,
  closes the line of succession, cannot be king, and its marriage is not an
  alliance.
- **Ladies' dignities** are only played on a lady, who carries at most one
  visible and one hidden. A **blocked** dignity forbids marriage; a **free**
  dignity allows it.

| Dignity | Code | Marriage | Effect |
|---|---|---|---|
| D'Arc | `ARC` | Blocked | +1 strength to the army she commands. |
| Castellan | `CTL` | Free | In a castle, reveals the orders issued on her fief. |
| Abbess | `ABB` | Blocked | Tied to a region: reveals the chains issued there. |
| Herbalist | `HRB` | Blocked | Shields her territory and its neighbors from plague; her army eats 2 rations less. |
| Astrologer | `AST` | Free | Shows the next 3 calamities; `V C NNN I` strikes one off per winter. |
| Chevalier d'Éon | `EON` | Free (unmarried) | Hidden: the lady presents as a man. |
| Correspondent | `COR` | Free | Hidden: as a player's hostage, reveals all their orders. |
| Spy | `ESP` | Free | Hidden: as a player's hostage, reveals their whole hand. |
| Poisoner | `EMP` | Free | Hidden: enemy armies in her region eat 1 ration more. |
| Witch | `SOR` | Free | Hidden: `S R` steers a calamity toward her region. |

Hidden dignities are known only to their player, until an inquiry, a capture
or a trial reveals them. The finer rules of each dignity are in appendix C.

### The crown

**The king** is a man, not a bastard, who cannot be pope. The crown is worth
**1 influence point**, on top of his fiefs. The king reigns for life and
keeps his powers as a hostage. If he is in the dungeon, the **queen** rules
as regent in his place.

- **Royal orders** (`W X`, `W R`, `W F`, `W G`): authorize annexing a
  territory of another fief, make a rank raise or a fief foundation free,
  grant a crown fief. Each type can be used at most
  {{royal.orders_per_winter}} time(s) per winter.
- **Royal rent**: 1 R per fief on the map each action season; the queen
  receives half.
- **Royal tax**: the Tax card played by the king on a fief that is not his
  (section 8).

**Electing a king.** If the throne is vacant, at least 3 titled nobles can
vote and the clergy can anoint, an election opens. Put forward a candidate
with `K R NNN` (a free man holding a fief, not excommunicated), then vote with
`V R NNN`. Each titled noble gives one vote to their player. The winner must
be alone at the top, with **at least 3 votes**, including the
**anointing**: two bishops' votes, or one vote from a cardinal or the pope.

**When the king dies**, his first eligible Claim heir receives the crown.
Otherwise the queen rules as regent and the throne is vacant.

---

## 7. The Church

![The Realm bookmark: pope, cardinals and bishops](/rules/en/realm.webp)

The map is divided into **bishoprics**, the same regions as the calamities.
The **papacy** is worth **1 influence point**; bishops and cardinals score no
points, but they bring votes and powers.

### Bishops

A bishopric without a bishop opens an election in winter as soon as **each
of its territories** is held by a player.

- **Candidacy**: `K E NNN BBB`, where `BBB` is the bishopric's seed village.
  The candidate is a free, unmarried, non-excommunicated man.
- **Vote**: `V E NNN BBB`. Your player weighs 1 vote per territory held in
  the bishopric (2 for the seed village), plus 1 per bishop, 2 per cardinal,
  3 for the pope and 1 per abbess of that bishopric.
- The best-placed candidate wins, if alone at the top. Otherwise the seat
  stays vacant until the next winter.

### Cardinals and the pope

A **cardinal** is a promoted bishop, in one of two ways: through the `CAR`
card (`D N NNN CAR`), effective at once, or through the purchase
`N C NNN` for {{religion.cardinal_cost}} R, effective at the end of winter.
The number of cardinals is capped according to the number of players.

As soon as 2 cardinals are in play and the papal throne is empty, a
**conclave** opens. Candidates are bishops or cardinals (`K P NNN`). Only
cardinals vote, one vote each (`V P NNN`). The winner needs **more than half**
of all cardinals.

A title won in winter only serves from the next winter on. A religious title
is lost on death or through excommunication, never through capture.

### The Church's powers

| Power | Who | Order | Effect |
|---|---|---|---|
| Excommunicate / lift | Pope | `X E NNN` / `X L NNN` | Strips the noble of votes and candidacy; a bishop or cardinal loses the title. One per winter, one excommunicated noble at a time per player. |
| Inquiry | Cardinal or pope | `Q HHH NNN` | Reveals `NNN`'s hidden dignity to everyone, and to you alone the commissioner of an assassination. Costs 1 to 5 R depending on the target's rank. |
| Trial | Two cardinals | `J HHH NNN` each | Executes an unmarried lady who carries a dignity (except the Abbess), an excommunicated noble, or an assassination commissioner. The king escapes it. |
| Dissolve a marriage | Pope, at a spouse's request | `X D NNN` from both sides | The spouses are single again. |
| Tithe | Bishop, cardinal or pope | `P DI HHH XXX` | Tax card: diverts a bishopric's mill output (section 8). |
| Appease a revolt | Cleric or abbess | `P FQ HHH TER` / `P PQ HHH TER` | Removes a rebel army (section 8). |

---

## 8. Special cards and calamities

Cards are played in spring, summer or autumn, in the **special cards**
field, without a noble. They resolve before army orders. In winter, your hand
refills by itself, up to {{special_orders.draw_orders_limit}} cards per
winter across all decks.

The deck holds {{special_orders.deck_size}} cards.

### Calamities

| | Calamity | Effect on its region |
|---|---|---|
| <img src="/icons/reaper-scythe.svg" width="22" alt=""> | Plague ({{special_orders.card.plague}}) | Divides armies by {{special_orders.effects.plague_army_divisor}} and can kill nobles. |
| <img src="/icons/raining.svg" width="22" alt=""> | Bad weather ({{special_orders.card.bad_weather}}) | Blocks moves leaving or entering the region (except hold and defensive support); mills produce nothing. |
| <img src="/icons/desert-skull.svg" width="22" alt=""> | Bad harvest ({{special_orders.card.famine}}) | Removes terrain rations and territory income. |

A drawn calamity is scheduled for the following year, at most one per
season. It is announced as soon as it is drawn, and the **spring omen**
reveals the season and region of every calamity of the year.

### Bonus cards

| | Card | Order | Effect |
|---|---|---|---|
| <img src="/icons/sun.svg" width="22" alt=""> | Fair weather ({{special_orders.card.fair_weather}}) | `P FW XXX` | Cancels bad weather; otherwise doubles the region's mill output. |
| <img src="/icons/wheat.svg" width="22" alt=""> | Abundant harvest ({{special_orders.card.abundant_harvest}}) | `P AH XXX` | Cancels a bad harvest; otherwise doubles the region's rations and income. |
| <img src="/icons/uprising.svg" width="22" alt=""> | Revolt ({{special_orders.card.revolt}}) | `P RV TER` | Raises rebels, or turns a famished army (see below). |
| | Tax ({{special_orders.card.seigneurial_tax}}) | `P TX …` / `P DI …` | Seigneurial tax, royal tax or tithe. |
| | Tunnel ({{special_orders.card.tunnel}}) | `P SO …` | Frees a noble, or lets an attack ignore walls. |
| | Ambush ({{special_orders.card.ambush}}) | `P EM AAA BBB` | If the attack wins, captures a noble of the beaten army. |
| | Assassination ({{special_orders.card.assassination}}) | `P AS HHH NNN` | Kills `NNN`, in secret. Also playable in winter. |
| | Justice ({{special_orders.card.justice}}) | `P JU HHH` | Executes `HHH` if they commissioned an assassination. Also playable in winter. |

For Fair weather and Abundant harvest, `XXX` is the region's seed village.
The bonus applies only once per region and per season.

**Revolt.** It is played on a territory:

- whose region suffers a bad harvest;
- or whose fief (or bishopric) was taxed this season or the previous one;
- or whose region saw a lady executed by trial in the previous season.

Each card raises {{special_orders.effects.revolt_army_min_size}} to
{{special_orders.effects.revolt_army_max_size}} neutral rebel troops, who
fight the occupier. Played on a **famished army**, the card raises no
rebels: if the rolls add up to the army's size, it turns neutral and its free
nobles die.

**Tax.** The same card has three uses:

- **Seigneurial tax** (`P TX XXX`, or `P TX HHH XXX` by the holder): doubles
  the territory income of one of your fiefs for the season.
- **Royal tax** (`P TX KKK XXX`, by the king): creates for you the same amount
  as the income of a fief that is not yours.
- **Tithe** (`P DI HHH XXX`, or `P DI HHH` for the pope over every
  bishopric): diverts the bishopric's mill output to your capital.

Any tax makes the fief or bishopric eligible for Revolt for two seasons.

**Appeasement.** A cleric removes a rebel army, without a card, once per
season:

- `P FQ HHH TER`, the free rite, in their region: a d6 succeeds on 1 to 3,
  fails on 4 or 5, and kills the cleric on 6;
- `P PQ HHH TER`, the paid, risk-free appeasement:
  {{religion.appeasement_cost_base}}^size R.

**Assassination and Justice.** An assassination is unsigned: only the
commissioner and those who investigate the victim or the commissioner know
the culprit. Justice only strikes the guilty: played on an innocent, it is
lost.

**Rumors.** The report hints, without naming anyone, at the cards going
around in players' hands.

---

## 9. End of game and victory

![The scores panel and each player's goal](/rules/en/scores.webp)

**Influence.** Your influence score is the sum of:

- your **titles**: 1 point per fief (vacant ones included), 1 for the crown
  and 1 for the papacy;
- your **influence marriages**: the titles of each house allied to you by a
  secondary marriage;
- your **head ally's** influence, if you have one.

Dignities, bishoprics, cardinalates, territories, armies and resources score
nothing.

**Goal.** It depends on the number of players, and is shown in the scores
panel.

| Players | 2 | 3 | 4 | 5 | 6 | 7 | 8 |
|---|---:|---:|---:|---:|---:|---:|---:|
| Alone | 3 | 5 | 6 | 8 | 9 | 11 | 12 |
| In a head alliance | — | 6 | 8 | 9 | 11 | 13 | 15 |

**End of game.** The game stops after the turn in which:

- a player without a head alliance reaches the solo goal;
- or an allied couple together reaches the alliance goal;
- or only one player is left;
- or the last year has been played. The best score then wins.

**Victory.** The winner, or the winning couple, earns a **major victory**.
Among the houses linked to the winner by marriages, the one with the
strongest link earns a **minor victory**, unless there is a tie. The others
fail. Ties are broken by solo victory first, then score, the crown and the
number of territories. Players still tied win together. A game where nobody
holds a title has no winner.

**Elimination.** A player with no fief, no capital and no army left is
eliminated. Nobles alone do not keep a player in the game.

---

## Appendix A — Order cheat sheet

`XXX`, `YYY`, `ZZZ` are territories; `NNN`, `HHH` are nobles; `CCC` is a
card. Army orders are free.

**Army chains** (spring, summer, autumn)

| Syntax | Order |
|---|---|
| `XXX A YYY` | Attack or move to an adjacent territory |
| `XXX S YYY` | Defensive support of the army on `YYY` |
| `XXX S YYY - ZZZ` | Offensive support of the attack `YYY` → `ZZZ` |
| `H XXX` | Hold |
| `XXX J YYY` | Peaceful join (last order) |
| `P XXX` | Pillage |
| `XXX D DEST1 DEST2 …` | Dispersal (`*` or `*NNN` after a destination sends nobles there) |
| `XXX T YYY N` | Transfer of `N` R |
| `( … )` | Loop line, retried until it succeeds |

**Noble cards** (any season)

| Syntax | Order |
|---|---|
| `R N CCC XXX` | Recruit the noble of card `CCC` on `XXX` |
| `C N HHH CCC` | `HHH` claims `CCC`'s titles (`CLM` card) |
| `D N NNN CCC` | Grant dignity `CCC` to `NNN` (`ABB` also takes a region) |

**Special cards** (spring, summer, autumn)

| Syntax | Card |
|---|---|
| `P FW XXX` · `P AH XXX` | Fair weather · Abundant harvest, on a region's seed village |
| `P RV TER` | Revolt |
| `P TX XXX` · `P TX HHH XXX` · `P TX KKK XXX` | Seigneurial tax · by the holder · royal tax |
| `P DI HHH XXX` · `P DI HHH` | Tithe on a bishopric · the pope's tithe |
| `P FQ HHH TER` · `P PQ HHH TER` | Appeasement: free rite · paid |
| `P SO NNN` · `P SO AAA - BBB` | Tunnel: escape · attack under the walls |
| `P EM AAA BBB` | Ambush |
| `P AS HHH NNN` | Assassination (also in winter) |
| `P JU HHH` | Justice (also in winter) |

---

## Appendix B — Winter orders

| Syntax | Order | Main condition | Cost |
|---|---|---|---|
| `T N` | Draw a noble card | once per winter, hand not full | 0 |
| `D C CCC` | Discard a card from your hand | — | 0 |
| `R N CCC XXX` | Recruit a noble | castle or village of yours, with one of your armies | 0 |
| `C N HHH CCC` | Claim | `CLM` card; `HHH` recruited during a marriage with `CCC` | 0 |
| `D N NNN CCC` | Grant a dignity | card in hand | 0 |
| `R T XXX` | Recruit a troop | one of your free nobles on `XXX` or next to it | {{costs.troop}} |
| `C M XXX` | Mill: build or upgrade | new: empty territory next to a castle or village; level 3 at most | {{costs.mill_levels.0}} / {{costs.mill_levels.1}} / {{costs.mill_levels.2}} |
| `C C XXX` | Castle, or fortify a village | — | {{costs.castle}} |
| `C D XXX` | Supply depot | — | {{costs.supply_depot}} |
| `E C XXX` | Designate your capital | one of your castles | 0 |
| `G XXX YYY N` | Send R to another player | from your castle or village to theirs | 0 |
| `O N NNN` · `P N NNN` | Prisoner as hostage · in the dungeon | a prisoner you hold | 0 |
| `H N NNN XXX [O\|P]` | Hand a noble to another player's army | your noble, or a prisoner you hold | 0 |
| `T F NNN XXX YYY ZZZ…` | Found a fief | 3 or more contiguous territories of yours, capital with a castle | {{costs.fief_per_territory}} / territory |
| `T A NNN XXX` | Assign a vacant fief | — | 0 |
| `T X NNN XXX YYY` | Annex `YYY` to fief `XXX` | `YYY` adjacent, held by your army; the king's consent if it belongs to another fief | {{costs.annex_per_territory}} |
| `T R NNN XXX` | Raise the fief one rank | once per winter, no higher than duchy | {{costs.rank_per_territory}} / territory |
| `M N NNN MMM` | Marriage | matching order from the other player | 0 |
| `W X` · `W R` · `W F` · `W G` | Royal orders | king or regent | 0 |
| `K R NNN` · `V R NNN` | Candidacy and vote for the throne | vacant throne | 0 |
| `K E NNN BBB` · `V E NNN BBB` | Candidacy and vote for a bishopric | vacant bishopric | 0 |
| `K P NNN` · `V P NNN` | Candidacy and vote in the conclave | at least 2 cardinals | 0 |
| `N C NNN` | Promote your bishop to cardinal | — | {{religion.cardinal_cost}} |
| `X E NNN` · `X L NNN` | Excommunicate · lift the excommunication | pope | 0 |
| `X D NNN` | Dissolve a marriage | pope and a spouse's request | 0 |
| `Q HHH NNN` | Inquiry | cardinal or pope, one per winter | 1 to 5 |
| `J HHH NNN` | Trial | two cardinals on the same target | 0 |
| `V C NNN I` | Strike off an announced calamity | Astrologer | 0 |
| `S R NNN CAL` · `S R NNN N` | Witch's ritual | a single ritual per winter, all players combined | 0 |

**Resolution order.** Each step sees the outcome of the previous ones:

1. the state is frozen: a seat freed this winter is only elected next
   winter;
2. excommunications and lifts;
3. royal orders, then management (investments, cardinal purchases, noble
   cards);
4. inquiries and filing of trials;
5. marriage dissolutions;
6. marriages;
7. elections: bishoprics, conclave, then the throne;
8. investiture of the elected;
9. end of winter: hands, vacant fiefs, stock conservation and
   repatriation, prosperity;
10. trial judgments, then Assassination and Justice.

Within a step, players go in the order of their number, and your orders in
the order of your sheet.

---

## Appendix C — Edge cases

### Chains and orders

- **Offensive support of one of your armies.** If it opens the chain, your
  supported army must have the announced attack as its current order.
- **Defensive support.** Any army that does not leave its territory receives
  it, including an army holding, supporting, pillaging or with No Orders. An
  army that attacks gives it up, even if its attack fails.
- **Join.** If one of your attacks wins on the target territory, the join
  merges with the victorious army.
- **Self-capture.** An army can attack your own empty castle to move in,
  without the castle's defense pushing it back.
- **Dispersal.** Destinations are adjacent to the starting territory or equal
  to it, and can repeat. A destination held by the enemy or contested uses up
  no troop. `*` sends all remaining nobles with the group, `*NNN` a single
  noble. Unassigned nobles stay behind as long as a troop remains there. To
  keep the chain in place, list the starting territory first:
  `BRI D BRI ATL NOR`. As a loop, the dispersal retries until every
  destination is filled.
- **Transfer.** The recipient is a castle, a village, or another player's
  army that controls its own territory. The amount is capped at
  `{{cost_base}}^(N − 1)` for an army of `N` troops. As a loop, a short stock
  sends what is left and ends the order.
- **Bad weather.** A move blocked by bad weather does not break the chain:
  the line waits for the next season.

### Combat

- An attack on a territory held by your army fails, but blocks the other
  attacks.
- In a head-to-head, the opposing player's supports do not count.
- A repelled attack still blocks its destination, unless it lost a
  head-to-head.
- A ruined castle gives no defense, even as a fief capital.

### Retreats

- On a tie within a category, the territory nearest to one of your castles or
  villages wins, then the lowest trigram.
- A host army must be yours and must not have been dislodged. It gains
  `N − 1` troops (or 1 if `N = 1`).
- The attacker's starting territory is always excluded. An anchored castle,
  of any player, blocks a retreat; an inert castle does not.
- Two armies with no other way out than the same empty territory are
  destroyed.

### Supply

- Your sources' reserve serves the nearest armies first, then the smallest.
  An army the reserve cannot cover takes just enough to feed its value `s`,
  and leaves the rest to the next ones.
- Without a capital, each territory pays its income to the nearest castle,
  otherwise to the nearest village.
- A mill with no controller produces on its own territory. The army that
  takes it finds that stock.
- Upgrading a mill is paid first from its own stock, then from that of the
  infrastructure it feeds.
- At the end of winter, a mill's stock joins the infrastructure it feeds.
  Stocks outside castles, villages, mills and depots are lost.
- **Prosperity**: each foundation renovates a ruined village, otherwise
  founds a village on the nearest free territory, otherwise turns a depot
  into a village, otherwise raises a village one level.

### Ruins and capitals

- `C C` on a ruined castle rebuilds it. On any other ruin, any building
  replaces it.
- The ruined castle of a fief capital remains the capital: the fief holds,
  without defense, and only `C C` can be built there.
- A famished army that pillages its territory can bring a fief capital to
  ruin.

### Nobles, titles and dignities

- **Bastard**: they only receive a new fief if they are the last of your
  line. A bastard card played on an heir cancels their Claim.
- **Dignities**: their bonuses are suspended in the dungeon. A hostage lady
  keeps them, and her captor also benefits from the passive bonuses.
- **Chevalier d'Éon**: once unmasked (by capture, inquiry, or a Claim
  targeting her marriage), she is excommunicated ex officio.
- **Witch**: `S R NNN CAL` calls a calamity (`PE`, `MT` or `FA`) onto her
  region if it is drawn; `S R NNN N` sets the season of the first calamity
  drawn (1, 2 or 3). Once unmasked, she is excommunicated.
- **Queen and regent**: the queen is the king's wife. She keeps her title as
  a widow, and loses it if she remarries. As regent, she issues the royal
  orders and the royal tax.
- **Crown fief**: a fief whose player has no noble left passes to the crown,
  if there is a king. Only `W G` grants it.
- **Marriage and elections**: a married noble cannot become bishop or pope.
  He can become king.
- **Inquiry on a dead noble**: it costs the price of the title they held at
  death.

### Cards

- Two identical cards on the same region: the first cancels the calamity,
  the second applies the bonus. Any further ones are lost.
- Two taxes on the same fief do not stack; the lord's tax comes before the
  royal tax. If the taxed capital falls during the season, the tax is
  cancelled.
- Competing tithes: the bishop comes before the cardinals, who come before
  the pope. Several cardinals share the output.
- Cancelled revolt: if an Abundant harvest cancels the bad harvest, pending
  revolts are cancelled and the card comes back.
- Neutral armies lose one troop each season their territory does not feed
  them.
- Tunnel (escape): the noble reappears free on your capital, otherwise on
  your castle with the lowest code.
- Ambush: the captured noble is the first in line of succession among those
  of the beaten army.
- Card resolution order: Tunnel, then Assassination, then Justice.

---

## Appendix D — A turn, start to finish

**The situation.** Hugues holds ROS, his capital, with 2 troops and his noble
HUG, and FOU, with 1 troop and his noble ODA. Brune holds ATL with 2 troops
and her noble MIA. NOR is empty.

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

**What Hugues writes.** His two nobles each issue a chain:

```text
HUG
ROS A ATL        # attack ATL from ROS
```

```text
ODA
FOU S ROS - ATL  # support the attack ROS -> ATL
```

**What Brune writes.** Nothing: her army has No Orders, but it defends
normally.

**The resolution.** The attack weighs 2 + 1 of support = 3, against 2 in
defense. The noble bonuses apply on both sides and change nothing. Hugues
enters ATL. Brune's army falls back to NOR, an empty territory that was not
attacked, and MIA follows it.

**The report.** Brune, who fought, sees the details: 3 against 2, with FOU's
support. An outside player only sees that ATL changed hands.

---

*Crown & Borough draws on [Fief](https://boardgamegeek.com/boardgame/107704/fief)
for its feudal setting and on [Diplomacy](https://boardgamegeek.com/boardgame/483/diplomacy)
for its simultaneous orders and supports.*
