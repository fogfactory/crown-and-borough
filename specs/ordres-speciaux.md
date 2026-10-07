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

## Syntaxe des ordres

Les ordres jouables du deck sont soumis dans un champ `special`, distinct des
chaînes de nobles. Les défausses d'hiver font partie de la feuille `winter`.
Aucun noble n'est requis :

- `D C BT` ou `D C RA` abandonne une carte bonus, en hiver uniquement ;
- `P BT TER` joue Beau temps au printemps, en été ou en automne ;
- `P RA TER` joue Récolte abondante au printemps, en été ou en automne ;
- `P RE TER` joue Révolte sur le territoire pendant ces saisons, si une mauvaise récolte affecte la région du territoire, si une taxe seigneuriale a été jouée sur la capitale du fief auquel appartient le territoire la saison courante ou la saison précédente, ou si un procès a exécuté une dame dans la région du territoire à la saison d'action précédente ; chaque carte ajoute un jet borné à l'armée neutre commune du territoire, qui se bat contre l'occupant le cas échéant ;
- `P PR HHH` joue un Procès sur le noble HHH, par exception à la règle « `TER` est
  le village seed d'une région » ci-dessous ; la carte est consommée à la pose et
  le procès est jugé en fin de tour (dames.md § Carte de procès) ;
- `P TX XXX` joue la Taxe seigneuriale (titres.md) au printemps, en été ou en
  automne ; XXX est, par exception à la règle « `TER` est le village seed
  d'une région » ci-dessous, la **capitale d'un fief** que le joueur détient
  (vacant compris). Elle double le revenu territorial du fief pour le tour,
  village inclus, sans jamais toucher la production des moulins ; deux cartes
  jouées sur le même fief le même tour ne se cumulent pas, la seconde est
  consommée sans effet.

La main est reconstituée automatiquement en hiver, après les ordres d'hiver
(dont `T N` et les cartes jouées, qui libèrent leur place) et après les
défausses. Le joueur reçoit `draw_orders_limit` cartes, moins une s'il a pioché
une carte de personnage cet hiver, et sans dépasser les places libres de la
main partagée (`hand_limit` moins les cartes d'ordres spéciaux, de noble et de
dignité détenues). Les joueurs sont traités par identifiant croissant. Aucun
ordre de pioche n'est nécessaire.

Les cartes jouées sont consommées puis leurs effets sont appliqués avant le
ravitaillement et la résolution simultanée des ordres d'armée.

Les aliases français et anglais sont acceptés quelle que soit la langue de
l’interface. `TER` est obligatoirement le village seed d’une région. Les kinds
de calamité ne peuvent pas être joués comme ordres de joueur. Lorsqu’un ordre
`P` est appliqué, il consomme la première carte du kind demandé dans la
main du joueur et la place dans la défausse ; l’effet est ensuite enregistré pour
une résolution simultanée par région.

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

- La peste réduit chaque armée de la région à `ceil(taille / divisor)`, avec au
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
  active affecte sa région, ou si une taxe seigneuriale a été jouée sur la
  capitale du fief auquel appartient ce territoire, la saison courante ou la
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

## Cartes prévues

Le deck pourra accueillir notamment des impôts, des mariages, des assassinats,
des nominations de cardinaux et des Claims. Les règles propres aux cartes de
succession, politique et religion restent dans leurs spécifications thématiques.
