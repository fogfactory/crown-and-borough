# Règles du jeu — Crown & Borough

**Version sociable (joueurs humains).** Ce document décrit les règles
actuellement actives dans le serveur de jeu. Les valeurs chiffrées proviennent
d'`assets/balance.yaml`, qui reste la source des nombres à jouer ; en cas de
divergence, le moteur fait foi.

---

## 1. Aperçu

Crown & Borough est un jeu de stratégie médiévale par tours, sur une carte de
territoires reliés par un graphe. Chaque joueur programme des **chaînes
d'ordres** pour ses armées ; toutes les chaînes sont résolues **simultanément**.

Les deux piliers de la tension :

- la résolution simultanée des intentions, des soutiens et des combats ;
- la logistique exponentielle : les grandes concentrations de troupes sont
  coûteuses et vulnérables au ravitaillement.

Une partie en ligne accepte de **2 à 8 joueurs** (jusqu'à 16 en partie locale).
La carte et les données chiffrées (propriétaires, tailles d'armées, stocks,
infrastructures, nobles) sont visibles par tous. En ligne, chaque joueur ne voit
en revanche que le détail de ses propres chaînes et des combats auxquels il
participe.

Chaque joueur commence sur un territoire distinct : un **château** y est
construit gratuitement (il devient la **capitale**), avec {{starting_resources}} R
de stock, une armée de {{starting_troops}} troupes et {{starting_nobles}} noble(s)
libre(s).

La partie dure un nombre d'années choisi à sa création (10 par défaut) ; voir
la section 8 pour la fin de partie et le score.

### Inspirations

Le projet s'inspire notamment de [Fief](https://boardgamegeek.com/boardgame/107704/fief)
pour son cadre féodal et ses enjeux territoriaux, ainsi que de
[Diplomacy](https://boardgamegeek.com/boardgame/483/diplomacy) pour la
programmation simultanée des ordres, les soutiens et la résolution des
affrontements. Ces jeux sont des inspirations de conception, pas des sources
de règles applicables à Crown & Borough.

---

## 2. Cycle du jeu et saisons

Une année comprend **quatre tours** : printemps, été, automne et **hiver**. Le
compteur de tour progresse d'une unité à chaque saison, hiver compris (un
nouveau printemps après l'hiver).

### Tours d'action (printemps, été, automne)

1. Chaque joueur prépare et soumet ses chaînes d'ordres pour ses nobles libres
   et ses armées.
2. Le moteur vérifie les soumissions : une erreur de syntaxe ou de réception
   empêche la résolution de la soumission concernée, sans modifier l'état.
3. Le moteur résout simultanément le ravitaillement, les intentions, les
   soutiens, les combats, les déplacements, les retraites, les jonctions, les
   dispersions et la progression des chaînes.
4. Le contrôle territorial, la position des nobles et les événements sont mis à
   jour, puis un **rapport de tour** est produit.

Une armée exécute au plus **une ligne de sa chaîne par saison d'action**. Un
ordre `A` ou `J` franchit donc au plus une case adjacente pendant cette
résolution. La chaîne reste attachée à l'armée entre les tours : ses lignes
suivantes sont exécutées aux saisons suivantes tant que la chaîne n'est pas
terminée ou cassée. Par exemple, `ROS A BOI` puis `BOI A ATL` fait avancer la
même armée de ROS à BOI ce tour-ci, puis de BOI à ATL au tour d'action suivant.
Une dispersion peut créer plusieurs groupes en une résolution, mais reste un
déplacement pacifique vers des cases adjacentes.

### Phase d'hiver

L'hiver est une **trêve de gestion** : aucune chaîne d'action, aucun mouvement,
aucun combat et aucun ravitaillement. Le joueur soumet une liste
d'investissements directs, traités dans l'ordre saisi (voir section 5).

| Saison | Ordres et timing |
|---|---|
| Printemps, été, automne | Le ravitaillement est calculé au **début de la résolution**, puis les intentions, soutiens, combats, déplacements, jonctions, dispersions et progressions de chaînes sont résolus ensemble. Il n'y a qu'une ligne courante par armée. |
| Hiver | Aucun ravitaillement ni ordre de chaîne : les investissements directs sont appliqués séquentiellement dans la liste saisie, puis les stocks sont conservés et rapatriés. |

Les ordres des saisons P/E/A ne forment donc pas une file d'attente entre
joueurs : chacun est évalué avec les intentions du tour. L'hiver est au
contraire une phase de gestion séquentielle.

---

## 3. Écrire une chaîne

Une chaîne est composée du **trigramme du noble émetteur** (ligne d'en-tête)
puis d'**une ligne par ordre**.

> Dans l'IHM web, l'en-tête du noble est **ajouté automatiquement avant
> l'envoi** : vous écrivez uniquement les lignes d'ordres.

Chaque ligne d'ordre est de la forme `POSITION SYMBOLE [cibles...]`. Les
commentaires commencent par `#`, les lignes vides et la casse sont normalisées
par le parser.

Exemple de chaîne complète :

```text
HUG              # en-tête : noble émetteur (ajouté par l'IHM dans le web)
ROS A BOI        # attaquer BOI depuis ROS
BOI S BRU - FOU  # soutien offensif
BOI J ROS        # jonction (doit être le dernier ordre)
```

### Liaison des ordres

- **single** : une ligne sans parenthèses. La chaîne s'arrête au premier échec,
  et le suffixe est abandonné.
- **loop** : la ligne entière est entre parenthèses, `(…)`. L'ordre est retenté
  à chaque résolution jusqu'à sa réussite ; un maintien en loop met l'armée en
  veille. Une erreur mécaniquement impossible casse toujours la chaîne.

Pour les ordres de mouvement, un ordre dont la position et la cible ne sont pas
adjacentes est rejeté lors de la soumission de la chaîne, sans réception
partielle de la chaîne. Le transfert `T` utilise à la place le réseau de
ravitaillement.

Une chaîne n'est pas limitée à une seule saison : une ligne réussie fait
progresser l'index de la chaîne et la ligne suivante attend la résolution
suivante. Une ligne `loop` conserve volontairement le même ordre lorsqu'elle
doit attendre une ouverture. Un mouvement invalidé par le mauvais temps met la
chaîne en pause : l'ordre reste en place et retente la saison suivante.

### Réception

- La chaîne est attachée **immédiatement et atomiquement** à l'armée présente
  sur la position de son premier ordre ; elle remplace la chaîne précédente de
  cette armée.
- Un noble libre ou otage n'émet qu'**une seule chaîne par tour**. Il peut
  commander n'importe quelle armée de son joueur : il n'a pas besoin d'être
  présent sur la position du premier ordre. Une chaîne ciblant une armée qui
  ne lui appartient pas, un noble au cachot ou un noble ayant déjà émis est
  rejetée.
- Si **plusieurs chaînes ciblent la même armée au même tour**, leur réception
  concurrente est invalidée : aucune n'est reçue et l'armée ne reçoit pas de
  nouvelle chaîne pour ce tour.
- Une armée sans chaîne est **Sans Ordre** : elle ne reçoit aucune action
  automatique.

---

## 4. Aide-mémoire des ordres

Les ordres ci-dessous sont disponibles au printemps, en été et en automne.
`XXX`, `YYY`, `ZZZ` sont des trigrammes de territoires ; `NNN` un trigramme de
noble. Aucun ne coûte de ressource en saison d'action.

| Symbole | Syntaxe | Effet |
|---|---|---|
| `A` | `XXX A YYY` | Attaque ou déplacement vers `YYY` adjacente. |
| `S` | `XXX S YYY` | Soutien défensif de l'armée qui tient `YYY`. |
| `S` | `XXX S YYY - ZZZ` | Soutien offensif de l'attaque de `YYY` vers `ZZZ`. |
| `H` | `H XXX` | Maintien sur `XXX`. |
| `J` | `XXX J YYY` | Jonction pacifique vers `YYY` adjacente ; **doit être le dernier ordre**. |
| `P` | `P XXX` | Pillage de l'infrastructure de la case occupée. |
| `D` | `XXX D DEST1 DEST2 ...` | Dispersion pacifique à force 0 : les destinations sont traitées dans leur ordre d'apparition, peuvent se répéter et les troupes arrivant sur une même case sont empilées. |
| `T` | `XXX T YYY N` | Transfert de `N` ressources vers un château, village ou une armée adverse via le réseau de ravitaillement. |

### Attaque (`A`) et jonction (`J`)

`YYY` doit être **adjacent** à `XXX` par une frontière franchissable.
L'armée entière se déplace vers `YYY`. Une attaque peut y combattre une armée
ennemie ; la jonction ne combat pas et est repoussée si la destination est
contestée. Si une attaque alliée remporte le combat sur `YYY`, la jonction peut
fusionner avec le vainqueur ; les attaques adverses qui perdent ce combat ne
l'empêchent pas d'arriver. Une armée peut également attaquer son propre château
vide pour s'y installer (auto-capture, voir section 6). La jonction doit être le
dernier ordre de la chaîne. Une jonction et une dispersion ne sont jamais des
attaques : elles ont une force de déplacement pacifique de 0 et ne délogent
personne. Une destination est contestée lorsqu'au moins une attaque adverse y
participe et qu'aucune armée attaquante ne remporte le combat.

### Soutien (`S`)

Un soutien renforce une armée de **n'importe quelle nationalité** :

- **défensif** (`XXX S YYY`) : renforce l'armée qui tient `YYY`, si `YYY` est
  adjacent à `XXX` (on ne se soutient pas soi-même) ;
- **offensif** (`XXX S YYY - ZZZ`) : renforce l'attaque de `YYY` vers `ZZZ`.

Pour un soutien offensif, `XXX` et `YYY` doivent chacun être adjacents à la
destination `ZZZ`, et `YYY` doit être l'armée qui attaque effectivement `ZZZ`.
Une attaque ratée ne crée pas de malus supplémentaire : l'armée reste soumise
au résultat normal du combat et sa chaîne continue ou casse selon sa liaison.

Il ne compte que si l'armée soutenue accomplit l'action annoncée. Une attaque
venue d'une case différente de la cible soutenue peut **couper** un soutien.

### Maintien (`H`) et pillage (`P`)

`H XXX` : l'armée reste sur place et peut recevoir un soutien défensif.
`P XXX` : détruit l'infrastructure de la case occupée ; un bonus de pillage
({{pillage_bonus}} R) est crédité à la source alliée la plus proche et peut réduire une famine.

### Dispersion (`D`)

`XXX D DEST1 DEST2 ...` traite les destinations dans leur ordre d'apparition,
avec au plus une troupe par destination. C'est un partage pacifique à force 0 :
il ne combat pas une armée ennemie ; une destination libre et non contestée est
prise, une destination alliée fusionne avec l'armée présente, tandis qu'une
destination contestée repousse cette affectation et ne reçoit pas de troupe.

- une destination est adjacente à `XXX` ou égale à `XXX` ; les destinations
  peuvent se répéter ;
- une destination occupée par une armée ennemie, contestée ou sans troupe
  disponible ne consomme pas de troupe ; une destination suivante peut néanmoins
  recevoir une troupe ;
- une destination occupée par une armée alliée peut recevoir la troupe et la
  fusionne avec l'armée présente ; plusieurs dispersions alliées qui arrivent
  sur la même case sont empilées dans une seule armée ;
- les troupes qui ne peuvent pas être envoyées restent sur la case d'origine ;
  une liste plus courte que l'armée laisse donc un résidu sur place ;
- les troupes arrivées sur une même destination sont empilées dans une seule
  armée ;
- les nobles explicitement affectés suivent le groupe produit : `*` affecte
  tous les nobles restants, `*NNN` affecte le noble `NNN` ; les nobles non
  mentionnés restent à l'origine tant qu'une troupe y demeure ;
- si toutes les troupes quittent l'origine et qu'un noble présent n'a pas de
  groupe produit, l'ordre est invalide à l'exécution ;
- la chaîne portée par l'armée suit le **premier groupe listé**. Ainsi,
  `BRI D ATL NOR` fait suivre la chaîne au groupe d'ATL lorsque ATL reçoit la
  première troupe ; pour garder la chaîne sur place tout en envoyant des
  troupes ailleurs, il faut écrire `BRI D BRI ATL NOR`. On ne saute pas à NOR
  après l'échec d'ATL lorsque le résidu reste à BRI : cela invaliderait la suite
  de la chaîne ;
- en `single`, les destinations non traitées produisent une dispersion
  partielle et la chaîne progresse ; en `loop`, le résidu retente jusqu'à
  l'arrivée d'une armée sur chaque destination ; si l'armée est épuisée avant
  d'avoir traité toutes les destinations, l'ordre est invalide.

Exemples :

```text
BRI D ATL ATL              # deux troupes empilées dans l'armée arrivée à ATL
BRI D ATL                  # une troupe vers ATL, le résidu reste sur BRI
BRI D ATL*HUG NOR          # HUG vers ATL, l'autre unité vers NOR
BRI D BRI ATL NOR          # BRI garde la chaîne, les autres groupes se séparent
(BRI D ATL NOR)            # dispersion en boucle
```

### Transfert (`T`)

`XXX T YYY N` est exécuté après le ravitaillement par l'armée située en `XXX`.
`YYY` doit être un château, un village ou la case d'une armée contrôlée par un
autre joueur vivant ; un dépôt sans armée ne peut pas recevoir. Le stock de la
case source peut exister sans infrastructure. La route suit la portée de
ravitaillement du donneur (`{{supply_range}}` cases, plus les dépôts contrôlés) et toute armée
adverse sur une case intermédiaire la bloque ; l'armée adverse en destination
est autorisée.

Une armée affamée ne peut pas transférer. Le montant est plafonné à `{{cost_base}}^(N - 1)`
pour une armée de `N` troupes, sans déduire les rations locales. Elle ne fait
aucun autre ordre pendant ce tour. Un manque de stock n'a aucun effet et ne
casse pas la chaîne `single`. En `loop`, le transfert retente ; si le stock
restant est inférieur au montant, le reliquat est envoyé par une livraison
partielle et l'ordre se termine.

---

## 5. Ordres d'hiver

L'hiver n'accepte **aucune chaîne ni mouvement** : uniquement des
investissements directs, une ligne par ordre, appliqués dans l'ordre saisi.

| Investissement | Syntaxe | Condition | Coût (R) |
|---|---|---|---|
| Recruter un noble | `R N XXX` | `XXX` contrôlé, avec un château ou un village et une armée du joueur | {{costs.noble}} |
| Recruter une troupe | `R T XXX` | `XXX` contrôlé, et un noble libre du joueur sur `XXX` ou adjacent | {{costs.troop}} |
| Construire ou améliorer un moulin | `C M XXX` | `XXX` contrôlé ; nouveau moulin sur case **vide** adjacente à un château ou village productif, ou moulin existant adjacent à cette source | {{costs.mill_levels.0}} (N1), {{costs.mill_levels.1}} (N2), {{costs.mill_levels.2}} (N3) |
| Construire un château | `C C XXX` | `XXX` contrôlé | {{costs.castle}} |
| Construire un dépôt de vivres | `C D XXX` | `XXX` contrôlé | {{costs.supply_depot}} |
| Désigner une capitale | `E C XXX` | un château contrôlé sur `XXX` | 0 |
| Placer un noble en otage | `O N NNN` | `NNN` est un prisonnier adverse détenu par le joueur | 0 |
| Placer un noble au donjon | `P N NNN` | `NNN` est un prisonnier adverse détenu par le joueur | 0 |
| Libérer un noble | `L N NNN` | `NNN` est détenu par le joueur ; la capitale de son propriétaire contient une armée de celui-ci | {{costs.liberation}} |
| Transférer des ressources | `G XXX YYY N` | `XXX` est un château ou village contrôlé par le donneur ; `YYY` est un château ou village contrôlé par un autre joueur | 0 |

Un transfert d'hiver ne se limite donc pas aux villages et châteaux du donneur :
il peut alimenter directement une structure contrôlée par le joueur destinataire.
Le débit, lui, suit les règles habituelles et ne peut utiliser que les réserves
de paiement du donneur.

Un moulin commence au niveau 1 et peut atteindre le niveau 3 inclus. La
construction coûte {{costs.mill_levels.0}} R ; les améliorations vers les niveaux
2 et 3 coûtent respectivement {{costs.mill_levels.1}} R et
{{costs.mill_levels.2}} R. `C M` sur un moulin déjà au niveau 3 est rejeté avec
le motif `mill_max_level_reached` et aucun stock n'est prélevé. Les moulins de
niveau supérieur à 3 déjà présents dans une partie sont conservés et restent
productifs ; seules leurs nouvelles améliorations sont bloquées.

### Otage et donjon

Les ordres `O N NNN` et `P N NNN` ciblent un noble **prisonnier adverse** détenu
sur la case d'une armée du joueur. `O` le place en statut `hostage` (otage) et
`P` en statut `dungeon` (donjon). La capture produit par défaut le statut
`hostage`. Un noble otage peut émettre une nouvelle chaîne tant qu'il est
détenu ; un noble au cachot ne le peut pas. Les ordres peuvent faire passer un
prisonnier d'un statut à l'autre.

Les investissements qui ciblent un territoire exigent le **contrôle de ce
territoire**. Une
construction remplace la structure existante uniquement quand la règle le
prévoit : un **château construit sur un village remplace le village** et
conserve le stock de la case. Un moulin seul (orphelin) ne produit rien.

### Vocabulaire des ressources

- `R` désigne une unité de **ressource stockable** : elle se trouve dans le
  stock d'une case, est produite par une source et sert à payer les
  investissements lorsqu'elle se trouve dans un château ou un village contrôlé ;
- une **ration** est une unité de nourriture consommée pendant le
  ravitaillement d'une saison d'action. Les rations locales sont produites et
  distribuées sur place ; elles ne deviennent pas automatiquement du stock `R` ;
- le **stock** est donc la quantité de `R` conservée sur une case.

Une source est chaque château ou village contrôlé, ainsi que toute case
contrôlée qui contient un stock positif pendant une saison d'action. Une case
ordinaire n'a pas de production, mais l'armée qui l'occupe consomme son stock
local avant les sources plus éloignées. Chaque château ou village produit `{{base_production}} R`
par tour, indépendamment des autres sources. Un deuxième château est donc une
deuxième source de production et de ravitaillement, même si un seul château
reste désigné comme capitale. Un moulin est construit uniquement sur une case
vide adjacente à un château ou un village productif ; il augmente la production
de **toutes** les sources voisines, sans filtre de propriétaire. Par exemple,
un moulin de niveau 1 entre un village et deux châteaux ajoute `+1 R` à chacun
de ces trois points de production. Même si le moulin se trouve sur une case
contrôlée par un autre joueur, il ajoute ce bonus à une source voisine contrôlée
par le joueur concerné. Un noble situé ailleurs sur la carte n'empêche pas
`C M ATL` et n'est pas requis pour le construire. Si la case de construction
porte déjà une autre infrastructure, l'ordre est rejeté avec
`structure_present` : une case ne porte jamais deux infrastructures. Un moulin
ajoute son niveau à chaque source adjacente, jusqu'à son niveau réel ; les
moulins hérités de niveau supérieur à 3 restent donc productifs.

**Paiement** : le coût est prélevé d'abord sur le stock de la case ciblée, puis
sur la source contrôlée la plus proche ; si la réserve totale est insuffisante,
**aucun paiement partiel** n'est effectué et l'investissement est rejeté
(signalé dans le rapport, coût non perdu).

Exemple : un `C M ATL` coûtant {{costs.mill_levels.0}} R consomme d'abord le
stock d'ATL puis le complément depuis la source contrôlée la plus proche. Si
ces stocks ne totalisent pas le coût requis, la construction est rejetée et
aucun prélèvement partiel n'est effectué.

**Fin de l'hiver** :

- chaque stock restant d'un château ou village est conservé à hauteur de
  `ceil(stock / {{winter_stock_divisor}})` ;
- un dépôt de vivres conserve intégralement son stock ;
- les stocks hors château, village et dépôt sont perdus ;
- les stocks des châteaux et villages hors capitale sont rapatriés vers la
  capitale, en laissant au maximum **{{village_stock_cap}} R par village** et **{{castle_stock_cap}} R par château** ;
- sans capitale, ces stocks restent sur place ; les stocks de dépôt restent sur
  leur case.

Les stocks hors château et village ne peuvent pas payer les investissements
d'hiver. Il n'est pas nécessaire de tout dépenser avant la fin de l'hiver : le stock non
dépensé est d'abord conservé, puis le surplus est rapatrié selon ces plafonds.
Un stock de 5 R devient donc 3 R avec `ceil(5 / 2)`. La conservation et le
rapatriement sont effectués après les investissements, et une case sans château,
village ou dépôt ne conserve pas de stock. Exemple : un village hors capitale garde
au plus 1 R après conservation ; le surplus rejoint la capitale, tandis qu'un
château hors capitale peut garder 2 R.

### Cartes spéciales

Le deck contient **{{special_orders.deck_size}} cartes**, dont **{{special_orders.card.plague}} peste**, **{{special_orders.card.bad_weather}} mauvais temps**, **{{special_orders.card.famine}} mauvaises récoltes**, **{{special_orders.card.fair_weather}} beaux temps**, **{{special_orders.card.abundant_harvest}} bonnes récoltes** et **{{special_orders.card.revolt}} révoltes**.

La main est limitée à **{{special_orders.hand_limit}} cartes**. Après les défausses
d'hiver, chaque joueur reçoit automatiquement jusqu'à **{{special_orders.draw_orders_limit}} cartes bonus**. Les calamités sont programmées dans les slots printemps (**{{special_orders.calamity_slots.spring}}**), été (**{{special_orders.calamity_slots.summer}}**) et automne (**{{special_orders.calamity_slots.autumn}}**). La peste réduit les armées par division de **{{special_orders.effects.plague_army_divisor}}**.

## 6. Cartes spéciales et calamités

Les ordres jouables de cartes sont soumis dans un champ `special`, séparé des
chaînes de nobles. Les défausses d'hiver sont écrites dans la feuille `winter`.
Aucun noble n'est nécessaire.

- `P BT ROS` : jouer Beau temps sur la région dont ROS est le seed ;
- `P RA ROS` : jouer Bonne récolte sur cette région ;
- `P RE BRU` : jouer Révolte sur le territoire BRU, uniquement si une mauvaise récolte active affecte sa région ;
- `D C BT` ou `D C RA` : défausser une carte, en hiver uniquement ;
La main est reconstituée automatiquement en hiver après les défausses. Aucun ordre
de pioche n'est nécessaire.

Les cartes Beau temps, Bonne récolte et Révolte sont jouables au printemps, en
été et en automne, mais pas en hiver. Les cartes jouées sont consommées avant la
résolution des ordres d'armée. Beau temps annule uniquement le mauvais temps et
Bonne récolte annule uniquement la mauvaise récolte. Si une carte annule une calamité,
elle ne produit pas son bonus régional. Deux cartes du même kind sont
consommées, mais une seule est effective : avec une calamité active, la
première carte annule et une seconde applique le bonus régional ; sans
calamité, la première carte l'applique. Le bonus reste plafonné à une unité
par catégorie et par région, les cartes au-delà étant consommées sans effet.

Le deck contient **{{special_orders.deck_size}} cartes** :
**{{special_orders.card.plague}}** peste, **{{special_orders.card.bad_weather}}**
mauvais temps, **{{special_orders.card.famine}}** mauvaise récolte,
**{{special_orders.card.fair_weather}}** beau temps,
**{{special_orders.card.abundant_harvest}}** bonnes récoltes et
**{{special_orders.card.revolt}}** révoltes. La main est limitée à
**{{special_orders.hand_limit}} cartes** et chaque joueur reçoit automatiquement
jusqu'à **{{special_orders.draw_orders_limit}} cartes bonus par hiver**, après ses
défausses.

Une calamité tirée est programmée dans le premier slot disponible de l'année
suivante : printemps (**{{special_orders.calamity_slots.spring}}**), été
(**{{special_orders.calamity_slots.summer}}**) ou automne
(**{{special_orders.calamity_slots.autumn}}**). Sa région est tirée de manière
déterministe lors de la programmation. L'augure du printemps révèle le kind, la
saison et la région de toutes les calamités de l'année ; les augures futures
restent cachées.

Dès son tirage, la calamité programmée est annoncée dans l'encart des cartes
spéciales de l'interface. L'annonce reste visible jusqu'à l'application de la
calamité ou sa contre-mesure. Aucune calamité ne se résout en hiver.

- la peste réduit les armées par division de **{{special_orders.effects.plague_army_divisor}}** et peut supprimer un noble ;
- le mauvais temps bloque les mouvements provenant ou visant sa région, sauf le maintien et le soutien défensif ;
- la mauvaise récolte désactive les moulins et les bonus de rations des infrastructures de sa région ;
- la Révolte se joue sur un territoire (`P RE TER`) pendant les saisons d'action, à condition que sa région subisse une mauvaise récolte. Chaque carte ajoute un jet entre **{{special_orders.effects.revolt_army_min_size}}** et **{{special_orders.effects.revolt_army_max_size}}** troupes à l'armée neutre commune du territoire ; le territoire peut être neutre (simple brigandage) et une armée y est créée si la case est vide. Si le territoire est occupé, la révolte est résolue comme un combat entre l'armée révoltée et l'occupant : le perdant se retire ou est détruit. Si la mauvaise récolte de la région est annulée par une Bonne récolte, les révoltes en attente sur la région sont annulées et le joueur récupère sa carte. Une rébellion vaincue se retire comme toute armée défaite au lieu de disparaître. Les armées neutres ne perdent jamais leur force à cause d'une famine, mais perdent une troupe en fin de tour si la production locale de leur territoire ne suffit pas à les nourrir.

Les rumeurs publiques sont recalculées dans chaque rapport à partir des mains
bonus actuelles de tous les joueurs. Elles apparaissent lorsqu'au moins deux
joueurs ont une carte en main, sans révéler le joueur ni l'identifiant interne de
la carte. Plusieurs cartes du même kind sont regroupées en une seule phrase
graduée : niveau 1 pour quelques cartes, niveau 2 pour une présence plus marquée
et niveau 3 pour une abondance exceptionnelle. L'échelle est recalée sur la
capacité de main de la partie (joueurs × limite de main), afin qu'un même nombre
de cartes ne produise pas le même niveau dans une petite et une grande partie.

---

## 7. Armées, combats et logistique

### Armées et force

Une armée est l'unique entité de force d'un territoire : elle porte un
propriétaire et une taille en troupes. Toutes ses troupes partagent la même
chaîne ; il n'existe pas d'ordres mixtes au sein d'une armée.

- la force d'une attaque est la **taille** de l'armée attaquante, avec **+{{noble_command_bonus}}** si
  un noble libre allié est présent sur sa case ;
- la force d'un soutien est la taille de l'armée soutenante, avec **+{{noble_command_bonus}}** si un
  noble libre allié est présent sur sa case ;
- la défense d'une armée reçoit le même bonus de **+{{noble_command_bonus}}** lorsqu'elle est
  commandée par un noble libre allié ;
- un château apporte un bonus défensif fixe de **+{{castle_defense_bonus}}**, même sans armée, sauf si
  tous les attaquants appartiennent au propriétaire du château : une armée peut
  ainsi attaquer son propre château vide pour s'y installer sans être repoussée
  par la défense du château (auto-capture) ;
- la plus haute force **strictement unique** gagne ; une égalité au sommet
  produit un **statu quo**, y compris sur une case vide ;
- une armée délogée perd son déplacement et doit **battre en retraite** ;
- une armée défaite bat en retraite selon l'ordre de priorité décroissant :
  1. case vide contrôlée par le propriétaire (avec ou sans château), même si elle
     a été combattue ce tour ;
  2. case vide non contrôlée (neutre ou ennemie), sans château et non combattue
     ce tour ;
  3. armée amie adjacente non délogée (priorité à la plus petite en troupes) avec
     fusion : l'hôte gagne `N − 1` troupes si la retraitante a `N ≥ 2` troupes,
     ou `1` troupe si `N = 1` (aucune perte). Plusieurs armées peuvent fusionner
     séquentiellement sur le même hôte ami sans destruction.
  À égalité dans une catégorie, la destination la plus proche d'un château ou
  village contrôlé l'emporte, puis le trigramme croissant. Pour les armées amies,
  le tri s'effectue par taille croissante, puis distance à la source contrôlée la
  plus proche, puis trigramme croissant. La case d'origine de l'attaquant est
  toujours exclue. Les châteaux neutres ou ennemis vides défendent contre une
  retraite et sont exclus. Deux armées qui doivent reculer sur la même case vide
  sans alternative sont détruites. L'ordre de résolution des retraites suit le
  trigramme croissant de leur case d'origine.

Le contrôle d'un territoire suit l'armée qui s'y arrête ; un contrôle acquis
reste acquis après le départ de l'armée, jusqu'à l'arrêt d'une armée ennemie.

### Ravitaillement exponentiel

Le ravitaillement est résolu **au début de chaque saison d'action**, avant les
ordres, les combats et les déplacements. Il n'existe pas de phase de
ravitaillement en hiver. Une armée d'une seule troupe demande `{{army_cost.1}}` ration :
elle n'est pas automatiquement gratuite.

Une armée de `N` troupes demande :

```text
coût = {{cost_base}}^(N - 1)  rations
```

| Taille | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Coût en rations | {{army_cost.1}} | {{army_cost.2}} | {{army_cost.3}} | {{army_cost.4}} | {{army_cost.5}} |

La production vivrière de la case de l'armée lui est attribuée à elle seule :
une armée consomme la production de la case qu'elle occupe jusqu'à hauteur de
sa demande, le surplus est perdu et le reste constitue sa demande à ravitailler. Il n'y a
jamais qu'une armée par case, donc aucune distribution entre armées : une
armée ennemie sur une case voisine ne prend jamais la ration de ta case.

Exemple : une armée de 2 troupes sur une colline portant un château
(production locale : {{ration_terrain.hill}} ; bonus du château : {{infra_rations_bonus}})
reçoit 2 rations, soit toute sa demande ; le surplus éventuel est perdu. Une armée
de 2 troupes sur un marécage (production {{ration_terrain.swamp}}) reçoit 1 ration et doit couvrir
sa demande restante de 1 ration.

**Production vivrière de la case (en rations)** : plaine {{ration_terrain.plain}} ;
forêt {{ration_terrain.forest}} ; colline {{ration_terrain.hill}} ; montagne
{{ration_terrain.mountain}} ; marécage {{ration_terrain.swamp}} ; **+{{infra_rations_bonus}}**
si la case porte un château ou un village.

**Sources de ravitaillement** : les **châteaux, villages et caches contrôlés**.
Un château ou un village produit **{{base_production}} R stockable par tour** ; un cache ordinaire
ne produit rien. Le flux traverse les
cases alliées, neutres ou contrôlées par un autre joueur et ne s'arrête que devant
une case occupée par une armée adverse. La portée de base
est de **{{supply_range}} cases** ; chaque dépôt de vivres contrôlé rencontré sur le trajet
ajoute **{{depot_range_bonus}} cases**. Un village neutre conserve son stock, inaccessible au joueur
avant capture.

Chaque source calcule sa propre production `R` : sa production de base, plus le
niveau de **chaque moulin adjacent**. Un même moulin peut donc alimenter toutes
les sources voisines ; il n'est pas réservé au propriétaire de sa case. Un
moulin orphelin, sans château ni village adjacent, produit `0 R`. Par exemple,
un village entouré de deux moulins de niveau 1 produit `{{base_production}} + 1 + 1 R` ; les
mêmes moulins ajoutent aussi leur niveau à tout château voisin. La présence ou
la position d'un noble ne conditionne jamais `C M XXX` ni cette production : un
noble situé en NOR n'empêche pas le joueur de construire `C M ATL` si ATL est
vide, contrôlé et adjacent à la source requise.

### Stocks et famine

En cas de déficit :

1. les stocks des châteaux, villages et caches contrôlés sont épuisés (du plus
   petit au plus grand, trigramme territorial en départage) ;
2. les armées restantes passent en **famine**, en commençant par les plus
   éloignées de leur source, puis les plus grosses, puis le trigramme
   décroissant.

Une armée en famine **combat et se défend à force 0** pour le tour, même si elle
porte un noble libre. Si elle se trouve sur une infrastructure, elle la **pille
automatiquement** ; le bonus de
pillage, diminué de sa demande résiduelle, peut la sortir de famine. Si le
pillage est insuffisant ou impossible, elle perd **1 troupe**, sans jamais
descendre sous 1. Elle reste néanmoins affamée et à force 0 pour toute la
saison en cours, même si cette perte rendait sa demande future soutenable. Cette
perte se répète à chaque saison où l'armée est encore affamée.

Exemple : une armée de 2 troupes en déficit demande 2 rations. Si ses stocks et
son pillage ne couvrent pas le déficit, elle perd une troupe et passe à 1 ; elle
reste à force 0 ce tour, même si une armée de 1 troupe ne demande ensuite qu'une
ration.

L'endpoint `GET /api/supply?territory=XXX` permet de prévisualiser le
ravitaillement d'une armée ou la zone atteinte depuis une source contrôlée
(uniquement hors hiver). Avec `&target=YYY`, il estime aussi la route d'un
transfert et ses blocages.

### Infrastructures

Une case ne porte qu'**une seule infrastructure**.

| Infrastructure | Condition | Effet v1 | Coût |
|---|---|---|---|
| Moulin | Construction sur case vide contrôlée, adjacente à un château ou village ; amélioration d'un moulin existant adjacent à cette source, jusqu'au niveau 3 | +1 R stockable par niveau à **chaque** source adjacente | {{costs.mill_levels.0}} / {{costs.mill_levels.1}} / {{costs.mill_levels.2}} |
| Dépôt de vivres | Aucune | +{{depot_range_bonus}} cases de portée de ravitaillement lorsqu'il est contrôlé | {{costs.supply_depot}} |
| Château | Aucune | +{{castle_defense_bonus}} défense, +{{infra_rations_bonus}} rations, produit {{base_production}} R stockable par tour, ancre de ravitaillement | {{costs.castle}} |
| Village | Généré neutre, **non constructible** | +{{infra_rations_bonus}} rations, produit {{base_production}} R stockable par tour, ancre après capture | — |

---

## 7. Nobles : capture, déplacement et capacité

Les nobles **chevauchent les armées** : ils suivent les déplacements, les
attaques, les jonctions, les dispersions et les retraites. Un noble ne compte ni
dans le ravitaillement ni dans les pertes d'un combat. Un noble libre du joueur,
présent sur la case de son armée, lui donne une seule fois **+{{noble_command_bonus}} de puissance** ;
les nobles adverses détenus, les nobles otages et les nobles au cachot ne donnent
pas ce bonus. Un noble peut rester seul sur une case après la perte de son armée.

**Capacité de commandement** :

- un noble **libre ou otage** émet au plus **une chaîne par tour** (nouvelle
  chaîne = nouveau tour) ;
- un noble **au cachot** (`dungeon`) ne peut pas émettre de nouvelle chaîne ;
- le noble peut donner cette chaîne à **n'importe quelle armée de son joueur** ;
  il n'est pas nécessaire qu'il soit présent sur la case de réception ;
- la chaîne s'applique à l'armée entière. Le bonus de commandement ne vient que
  d'un noble libre allié **présent sur la case de l'armée au moment du calcul** :
  émettre une chaîne à distance ne téléporte pas le noble et ne donne pas de
  bonus à l'armée distante.

> Il n'existe pas, dans cette version, de limite de **nobles portés par une
> armée** : une armée transporte tous les nobles présents sur sa case.

**Capture** : lorsqu'une armée portant des nobles est **détruite** sur une case
occupée par une armée ennemie, les nobles qu'elle portait sont capturés et
deviennent par défaut `hostage`. Un noble otage peut continuer à émettre une
chaîne ; seul son passage au cachot lui retire cette capacité. Le joueur qui le
détient peut lire les chaînes émises par cet otage dans les parties en ligne,
même si elles commandent une armée restée chez le propriétaire du noble.

**Libération** : pendant l'hiver, `L N NNN` est émis par le joueur qui détient
le prisonnier, et non par son propriétaire. Si la capitale du propriétaire
existe et contient une armée de celui-ci, le noble réapparaît **libre dans cette
capitale** ; sinon l'ordre est rejeté.

Un transfert volontaire de noble passe par une dispersion : par exemple,
`BRI D ATL*HUG NOR` envoie HUG avec le groupe d'ATL. Le noble `HUG` n'accorde
le bonus de `+{{noble_command_bonus}}` que si ce groupe le porte effectivement au moment du combat ou
de la défense.

Un joueur qui ne possède aucun noble libre ou otage apte à émettre n'a pas à
soumettre de chaînes pendant une saison d'action.

Les nobles affectés lors d'une **dispersion** doivent tous être répartis entre
les destinations (`*` ou `*NNN`), voir section 4.

---

## 8. Fin de partie, score et victoire

La durée d'une partie est choisie à sa création, entre 1 et 50 années (10 par
défaut). Une année compte quatre tours ; l'interface affiche l'année historique
`1000 + année`, soit « Année 1001 » au premier tour.

**Élimination** : un joueur est éliminé lorsqu'il ne contrôle plus aucun
territoire et ne possède plus aucune armée. Les nobles seuls ne maintiennent pas
un joueur en lice. Un joueur éliminé ne soumet plus d'ordres.

**Fin de partie** : la partie se termine :

- immédiatement lorsqu'un seul joueur reste en lice : il gagne ;
- sinon, après la résolution du dernier hiver de la durée choisie : le joueur
  qui a le score le plus élevé gagne. Une égalité parfaite au sommet ne désigne
  aucun gagnant.

**Score** : il est recalculé après chaque tour et visible par tous.

| Élément | Points |
|---|---:|
| Territoire contrôlé | 1 |
| Village contrôlé | 2 |
| Moulin contrôlé | 1 |
| Château contrôlé | 5 |
| Noble détenu | 2 |
| Troupe | 1 par unité dans ses armées |
| Ressource `R` | 1 par unité en stock sur ses territoires contrôlés |

Les infrastructures et les ressources ne rapportent des points que sur un
territoire contrôlé. Un noble libre compte pour son propriétaire. Un noble
capturé, otage ou au donjon, compte pour le joueur qui contrôle le territoire où
il se trouve, et non pour son propriétaire d'origine.
