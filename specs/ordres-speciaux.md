# Ordres spéciaux et calamités

**Milestone lié :** [Ordres spéciaux & Calamités](https://github.com/fogfactory/crown-and-borough/milestone/4)

**Dépend de :** la boucle hiver, la résolution simultanée et les rapports du socle actuel.

## Pioche et main

- chaque hiver, un joueur pioche au plus `special_orders.draw_orders_limit`
  cartes (2), tous decks confondus : le remplissage automatique des ordres
  spéciaux et la pioche `T N` du deck de personnages (au plus une carte, voir
  `succession.md`) partagent ce plafond ;
- les cartes peuvent être conservées pour une résolution ultérieure ;
- le joueur peut abandonner des cartes existantes (`D C KIND` pour une carte
  d'ordre spécial ; `D C CCC` pour une carte de la main de nobles, voir
  `succession.md`) ;
- la main est limitée à `special_orders.hand_limit` cartes (6), quel que soit
  le deck d'origine : ordres spéciaux, cartes de noble et cartes de dignité
  comptent ensemble. `T N` est rejeté (`hand_limit_reached`) quand la main est
  pleine.

Chaque carte devra déclarer son coût, ses conditions, son moment d'utilisation,
son effet et les informations visibles par les autres joueurs.

## Calamités

Un tirage peut déclencher une calamité. Le nombre maximal de calamités est de
trois par année, même si plusieurs cartes sont tirées. La règle devra définir la
pioche, le renouvellement, les défausses et la résolution déterministe des
calamités. Les calamités se résolvent au printemps, en été et en automne ;
aucune calamité ne se résout en hiver. L’augure du printemps révèle pour chaque
calamité son kind, sa saison et sa région ; les augures futures restent cachées.
La saison d'une calamité est tirée au hasard parmi les saisons de l'année
suivante qui ont encore un slot libre (un par saison) ; sa région est tirée au
hasard, sauf pour une calamité déviée par le rituel de la Sorcière
([dames.md](dames.md#sorcière-cachée)).

## Syntaxe des ordres

Les ordres jouables du deck sont soumis dans un champ `special`, distinct des
chaînes de nobles. Les défausses d'hiver font partie de la feuille `winter`.
Aucun noble n'est requis :

- `D C BT` ou `D C RA` abandonne une carte bonus, en hiver uniquement ;
- `P BT TER` joue Beau temps au printemps, en été ou en automne ;
- `P RA TER` joue Récolte abondante au printemps, en été ou en automne ;
- `P RE TER` joue Révolte sur le territoire pendant ces saisons, si une mauvaise récolte affecte la région du territoire, si un impôt (taxe ou dîme) a été joué sur la capitale du fief ou l'évêché auquel appartient le territoire la saison courante ou la saison précédente, ou si un procès a exécuté une dame dans la région du territoire à la saison d'action précédente ; chaque carte ajoute un jet borné à l'armée neutre commune du territoire, qui se bat contre l'occupant le cas échéant ;
- `P AG HHH TER` (rite gratuit, risqué) et `P AP HHH TER` (payant, sûr) font jouer
  à un ecclésiastique l'apaisement de révolte, sans carte : voir
  [religieux.md § Apaisement de révolte](religieux.md#apaisement-de-révolte) ;
- `P TX HHH XXX` joue la carte Impôts (titres.md, religieux.md) au printemps, en
  été ou en automne ; HHH est le **noble émetteur** et XXX la cible, par
  exception à la règle « `TER` est le village seed d'une région » ci-dessous :
  la **capitale d'un fief** (taxe seigneuriale ou royale, vacant compris) ou le
  **village seed d'un évêché** (dîme). L'émetteur détermine ce qui est permis :
  seigneur titré sur ses fiefs, roi (ou régente) sur tout fief constitué, évêque sur son
  évêché, cardinal sur tout évêché, pape sur tous les évêchés (priorité évêque, puis cardinaux, puis
  pape). Pour la taxe seigneuriale, `HHH` est le titulaire du fief (il peut être omis pour un fief vacant : `P TX XXX`). `P DI HHH XXX` joue explicitement une dîme ; la dîme du pape s'écrit `P DI HHH`, sans cible, et vise tous les évêchés. La taxe double le revenu
  territorial du fief pour le tour, village inclus, sans jamais toucher la
  production des moulins ; la dîme détourne la production des moulins de
  l'évêché. Deux cartes jouées sur le même fief le même tour ne se cumulent pas,
  la seconde est consommée sans effet.

- `P SO NNN`, `P SO XXX`, `P AS HHH NNN`, `P JU HHH` et `P EM AAA BBB` jouent
  les cartes d'événement Souterrain, Assassinat, Justice et Embuscade, par
  exception à la règle « `TER` est le village seed d'une région » : voir
  [Cartes d'événement](#cartes-dévénement).

La main est reconstituée automatiquement en hiver, après les ordres d'hiver
(dont `T N` et les cartes jouées, qui libèrent leur place) et après les
défausses. Le joueur reçoit `draw_orders_limit` cartes, moins une s'il a pioché
une carte de personnage cet hiver, et sans dépasser les places libres de la
main partagée (`hand_limit` moins les cartes d'ordres spéciaux, de noble et de
dignité détenues). Les joueurs sont traités par identifiant croissant. Aucun
ordre de pioche n'est nécessaire.

L'ordre des ordres d'hiver entre eux (excommunication, gestion, enquête, mariage,
élection, procès) est fixé par [hiver.md](hiver.md).

Les cartes jouées sont consommées puis leurs effets sont appliqués avant le
ravitaillement et la résolution simultanée des ordres d'armée.

Les aliases français et anglais sont acceptés quelle que soit la langue de
l’interface. `TER` est obligatoirement le village seed d’une région. Les kinds
de calamité ne peuvent pas être joués comme ordres de joueur. Lorsqu’un ordre
`P` est appliqué, il consomme la première carte du kind demandé dans la
main du joueur et la place dans la défausse ; l’effet est ensuite enregistré pour
une résolution simultanée par région.

## Visibilité des ordres dans le rapport

Un ordre refusé d'un joueur apparaît aux autres joueurs comme un simple refus
motivé : l'ordre, le noble, la dignité et la cible d'une prétention n'y figurent
pas, car ils trahiraient une carte de sa main privée. Le joueur et le
spectateur voient le refus complet. La section des cartes ne montre aux autres
joueurs ni la pose d'un Assassinat ni celle d'un Souterrain ; l'auteur lit, pour
l'Assassinat, la victime `NNN` et le commanditaire `HHH`.

## Rumeurs publiques

Les rumeurs publiques sont recalculées dans chaque rapport à partir des mains
bonus actuelles de tous les joueurs. Elles apparaissent lorsqu'au moins deux
joueurs ont une carte en main. Les rumeurs sont regroupées par kind : plusieurs
cartes du même kind ne sont jamais répétées dans le rapport, mais augmentent le
niveau sémantique de la phrase (niveau 1, 2 ou 3). Les niveaux sont recalés sur
la capacité de main de la partie, soit le nombre de joueurs multiplié par la
limite de main. Une rumeur ne révèle ni le joueur concerné ni l'identifiant
interne de la carte.

## Effets des calamités et de la révolte

- La peste frappe toute armée et tout noble qui commence ou termine le tour
  dans la région, qu'il y reste, en sorte ou y entre (retraite comprise) : les
  occupants de départ sont frappés avant la résolution des ordres, les arrivées
  une fois mouvements et retraites réglés. Une armée ou un noble n'est frappé
  qu'une fois par tour. La peste réduit l'armée à `ceil(taille / divisor)`, avec au
  moins une troupe, et peut tuer un noble selon la balance ; un noble tué reçoit
  supprimé de l'état et ne rapporte plus de points. Une chaîne émise pendant le
  tour est supprimée ; une chaîne historique déjà en cours continue.
- Le mauvais temps bloque les attaques, jonctions, dispersions, pillages et
  soutiens offensifs provenant de sa région ; le maintien et le soutien défensif
  restent possibles. Les moulins de la région ne produisent rien.
- La famine supprime les rations de terrain de chaque territoire de sa région
  et la production de base des châteaux et villages de la région ; les moulins
  ne sont pas touchés.
- La révolte est une carte bonus, jouable sur un territoire si une famine
  active affecte sa région, ou si un impôt (taxe ou dîme) a été joué sur la
  capitale du fief ou l'évêché auquel appartient ce territoire, la saison courante ou la
  saison précédente — tout territoire du fief est alors éligible, pas
  seulement sa capitale taxée —, ou si un procès a exécuté une dame dans la
  région à la saison d'action précédente. Elle crée des armées `NEUTRAL` sur les cases
  vides, selon la balance.

## Bonus régionaux

La météo agit sur les moulins, la récolte sur la terre. Une carte bonus annule
d'abord la calamité de même famille ; sinon, ou pour une seconde carte, elle
applique son bonus, au plus une fois par kind et par région :

- Beau temps **double** la production des moulins de la région ;
- Récolte abondante **double** les rations de terrain de chaque territoire de
  la région et la production de base de ses châteaux et villages.

Ces effets, tout comme ceux des calamités, sont pris en compte par la
projection de ravitaillement (`/supply`) avec les cartes du brouillon du
joueur.

## Cartes d'événement

**Milestone lié :** [Royauté](https://github.com/fogfactory/crown-and-borough/milestone/25).

Quatre cartes tirées du jeu de plateau *Fief* rejoignent le deck d'ordres
spéciaux, au même titre que Beau temps ou Impôts. Chacune se joue au
printemps, en été ou en automne par un ordre `P` (l'Assassinat et la Justice se
jouent en toute saison, hiver compris, voir ci-dessous), est gratuite, est consommée à
la pose (donc défaussée même si l'effet s'avère nul) et a son effet appliqué
avant le ravitaillement et la résolution des ordres d'armée, sauf l'Embuscade
dont l'effet est appliqué à la résolution du combat visé. Leurs poids de
tirage sont des entrées de `special_orders.bonus_weights` (`tunnel`,
`assassination`, `justice`, `ambush`), à calibrer. Un ordre dont une condition
n'est pas remplie est rejeté avec le motif indiqué et ne consomme pas la
carte. Les cartes jouées le même tour se résolvent dans l'ordre : Souterrain,
Assassinat, Justice, puis (pendant la résolution des mouvements) Embuscade.

### Cartes jouées sur une attaque

Souterrain (`P SO AAA - BBB`) et Embuscade (`P EM AAA BBB`) désignent une
attaque de `AAA` vers `BBB`, contrôlée à la soumission :

- si l'armée sur `AAA` appartient au joueur, son ordre en cours doit être une
  attaque vers `BBB` ;
- si elle appartient à un autre joueur, aucun ordre n'est exigé (celui de
  l'autre joueur est inconnu) mais l'attaque doit être possible : `AAA` et
  `BBB` sont adjacents et l'armée n'est pas neutre. La carte est un conseil
  tactique : l'effet profite à l'attaquant s'il attaque effectivement, sinon la
  carte est consommée sans effet.

La même règle s'applique aux soutiens offensifs `XXX S YYY - ZZZ` qui ouvrent
une chaîne reçue ce tour : si l'armée sur `YYY` appartient au joueur, elle doit
avoir pour ordre en cours une attaque vers `ZZZ` (`support_no_attack`) ; le
soutien d'une armée d'un autre joueur reste libre.

### Souterrain

Carte `tunnel`, code `SO`. Elle a deux usages, distingués par la forme de
l'ordre.

- **Évasion** `P SO NNN` : `NNN` est un noble du joueur détenu (`hostage` ou
  `dungeon`) par l'armée d'un autre joueur (`noble_not_held` sinon). Il
  s'échappe et réapparaît libre sur la capitale du joueur ; si elle n'est plus
  contrôlée, sur son château contrôlé de plus petit code (`tunnel_no_refuge`
  s'il n'en a aucun). Un refuge tenu par l'armée d'un autre joueur est écarté,
  pour que le noble ne se retrouve jamais avec une armée étrangère. Le détenteur
  voit l'évasion dans son rapport ; les autres joueurs ne voient aucune rumeur
  nominative.
- **Attaque par souterrain** `P SO AAA - BBB` : désigne l'attaque depuis `AAA`
  vers `BBB`, vérifiée à la soumission (`tunnel_no_attack`) selon la règle des
  [cartes jouées sur une attaque](#cartes-jouées-sur-une-attaque). Cette attaque, et elle seule, ignore les
  fortifications de `BBB` (bonus du château, de la cité ou du village fortifié) :
  les autres attaquants sur `BBB` les affrontent normalement. Si l'attaque n'a
  finalement pas lieu (ordre annulé ou invalide à la résolution), la carte est
  consommée sans effet. Le rapport de combat signale l'attaque menée par
  souterrain et indique, pour le défenseur, la force affrontée sans le bonus de
  fortification en plus de sa force totale.

### Assassinat

Carte `assassination`, code `AS`. `P AS HHH NNN` : `HHH` est le **commanditaire**,
un noble libre du joueur (ni `hostage` ni `dungeon`), et `NNN` un noble d'un
autre joueur, où qu'il soit et quel que soit son statut. `NNN` meurt sans jet
de dé : mort normale ([succession.md](succession.md)) avec fiefs, prétentions,
chaîne émise ce tour supprimée, et couronne si c'est le roi. La mort est
enregistrée avec la cause `assassination`. Les motifs de rejet sont
`assassination_commissioner_invalid` (`HHH` inconnu, d'un autre joueur ou non
libre), `assassination_unknown_noble` (`NNN` inconnu ou déjà mort) et
`assassination_own_noble` (`NNN` appartient au joueur) ; la carte n'est alors
pas consommée. La carte se joue en **toute saison**. Au printemps, en été et en
automne, l'ordre `P AS HHH NNN` est écrit dans les ordres spéciaux et la mort
s'applique à la pose, avant le ravitaillement et les ordres d'armée. En hiver
il est écrit sur la feuille d'hiver et s'applique à l'étape 9 de
[hiver.md](hiver.md#ordre-de-résolution), après les verdicts des procès : la
carte est consommée au début de l'étape 8, avant le remplissage des mains, et
la mort est appliquée à l'étape 9 (joueurs par identifiant, puis ordre de la
feuille). Un ordre dont la victime ou le commanditaire a quitté le jeu entre
temps (exécuté par un procès du même hiver, par exemple) est rejeté à l'étape 9
et la carte, déjà consommée, est perdue sans effet. Une carte rejetée à l'étape 8 (motifs ci-dessus ou `no_card_for_kind`)
n'est pas consommée et ne fait pas échouer l'hiver. Les ordres sont vérifiés à la soumission ; une seconde carte
visant un noble déjà tué ce tour est rejetée à la résolution.

Le commanditaire et le joueur qui a posé la carte sont enregistrés avec
l'assassinat, dans l'entrée de lignée du noble tué (`RemovedNoble.Commissioner`,
l'identifiant de `HHH`, et `RemovedNoble.Author`) ; la carte Justice lit le
commanditaire. Le rapport public annonce la mort de `NNN` sans nommer de
coupable (ni le joueur qui a joué la carte), y compris au propriétaire de `NNN`,
qui apprend la mort mais pas le commanditaire ; seul le joueur qui a posé la
carte lit le nom de `HHH` dans le rapport. Une
rumeur publique signale qu'un assassinat a eu lieu.

**Rapport de l'enquêteur.** Lorsqu'une enquête (`Q`) apprend à un joueur le lien
commanditaire ↔ victime (enquête sur la victime ou le commanditaire, vivant ou
mort), son rapport — et lui seul, avec la vue spectateur — porte un événement
`assassination_revealed` : « votre enquête révèle que HHH a commandité
l'assassinat de NNN ». Il est émis une seule fois par lien et par joueur, y
compris quand le lien est appris à la pose d'un assassinat par une enquête
antérieure sur la future victime ou son commanditaire , y compris pour le
propriétaire de la victime ; le joueur qui a posé la carte, déjà informé par
l'événement d'assassinat, n'en reçoit pas. Aucune rumeur pour les autres joueurs.

**Qui connaît le lien commanditaire ↔ victime.** Le lien est connu de :
1. le joueur qui a posé la carte ;
2. tout joueur ayant mené une [enquête](religieux.md#enquête) (`Q HHH NNN`) sur
   la victime **ou** sur le commanditaire, que l'enquête ait eu lieu avant ou
   après l'assassinat.

Modèle : chaque enquête réussie est consignée côté serveur dans un registre
(`GameState.Inquiries`, un couple joueur / noble par cible, sans doublon) ; la
connaissance n'est pas stockée par assassinat mais déduite à la lecture
(`GameState.KnowsAssassination`) de l'auteur de l'entrée de
lignée et de ce registre. Le propriétaire de la victime n'est pas informé
d'office : il doit enquêter comme tout autre joueur. Une enquête antérieure sur `HHH` ou `NNN` donne donc
le lien dès que l'assassinat a lieu, et une enquête postérieure le révèle.

**Dans l'état (`state.json`).** Pour un joueur qui connaît le lien, la
projection (`internal/api`) renseigne `deceased[].commissioner` (code de `HHH`
sur la victime) et `commissioned` (codes des victimes) sur le commanditaire,
vivant (`nobles[]`) ou mort (`deceased[]`). Aucun autre joueur ne voit ces
champs ; la projection publique (sans joueur) non plus ; un spectateur voit
tous les liens. Le web affiche le lien sur l'entrée de la victime et sur celle
du commanditaire dans la vue Lignée, et sur le commanditaire présent dans le
poste de commandement du territoire.

### Justice

Carte `justice`, code `JU`. `P JU HHH` cible n'importe quel noble `HHH`, de
n'importe quel joueur (le sien compris), où qu'il soit et quel que soit son
statut (libre, otage, cachot), à l'exception du Roi, toujours protégé. Si `HHH`
a été le commanditaire d'au moins un assassinat, quelle qu'en soit la victime,
il est exécuté : mort normale (fiefs, prétentions, cartes), enregistrée avec la
cause `execution`. Sinon la carte est consommée sans effet. Les motifs de rejet
sont `justice_unknown_noble` (`HHH` déjà mort) et `justice_king` (`HHH` est le
Roi) ; la carte n'est alors pas consommée.

La carte se joue en **toute saison**. Au printemps, en été et en automne, l'ordre
`P JU HHH` est écrit dans les ordres spéciaux et l'exécution s'applique à la
pose, après les assassinats du tour. En hiver il est écrit sur la feuille
d'hiver et s'applique à l'étape 9 de [hiver.md](hiver.md#ordre-de-résolution),
après les verdicts des procès puis les assassinats : la carte est consommée au
début de l'étape 8, et un ordre dont la cible est morte entre temps est rejeté,
carte perdue.

Seuls les joueurs qui connaissent le lien (voir « Assassinat ») savent qui est
un commanditaire : les autres jouent Justice à l'aveugle, sur un soupçon. Ni
l'aperçu de la feuille d'hiver ni la projection du ravitaillement ne révèlent
l'issue d'une Justice avant sa résolution. Le rapport public nomme le joueur qui
a joué la carte et le noble exécuté (« justice rendue pour un assassinat ») ; un
échec n'est visible que du joueur qui a joué la carte (« aucun effet »), sans
rumeur publique.

Le commanditaire d'un assassinat peut aussi être exécuté par un procès à deux
cardinaux ([dames.md § Procès](dames.md#procès)), sans excommunication
préalable et sans fenêtre de révolte. Le Roi échappe toujours au procès.

### Embuscade

Carte `ambush`, code `EM`. `P EM AAA BBB` désigne l'attaque de l'origine `AAA`
vers la cible `BBB`, vérifiée à la soumission (`ambush_no_attack`) selon la règle
des [cartes jouées sur une attaque](#cartes-jouées-sur-une-attaque) et relue à
la résolution. Si cette attaque est **victorieuse** et que
l'armée défenseuse bat en retraite avec au moins un noble libre, un noble libre
adverse de cette armée est capturé : celui qui arrive en tête de la ligne de
succession de son propriétaire. Il est capturé comme à la destruction d'une
armée ([gdd.md](gdd.md), nobles `hostage` par défaut, bâtard au `dungeon`) et
détenu par l'armée attaquante, donc par son propriétaire (le joueur de la carte
lorsque l'attaque est la sienne). Sans victoire, sans retraite ou sans noble
dans l'armée en retraite, la carte est consommée sans effet.

## Cartes prévues

Le deck pourra encore accueillir des mariages. Les cartes de Claim et de
cardinal appartiennent au deck de nobles. Les règles propres aux cartes de
succession, politique et religion restent dans leurs spécifications thématiques.
