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

Une partie accepte de **2 à 16 joueurs**. Le serveur v1 est une session
« hotseat » : la carte et les données chiffrées (propriétaires, tailles
d'armées, stocks, infrastructures, nobles) sont visibles par tous.

Chaque joueur commence sur un territoire distinct : un **château** y est
construit gratuitement (il devient la **capitale**), avec 10 R de stock, une
armée d'une troupe et un noble libre.

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

La non-adjacence d'un ordre est contrôlée lors de son exécution : les ordres
antérieurs restent valides et le suffixe est abandonné.

Une chaîne n'est pas limitée à une seule saison : une ligne réussie fait
progresser l'index de la chaîne et la ligne suivante attend la résolution
suivante. Une ligne `loop` conserve volontairement le même ordre lorsqu'elle
doit attendre une ouverture.

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

### Attaque (`A`) et jonction (`J`)

`YYY` doit être **adjacent** à `XXX` par une frontière franchissable.
L'armée entière se déplace vers `YYY`. Une attaque peut y combattre une armée
ennemie ; la jonction ne combat pas et est repoussée si la destination est
contestée. La jonction doit être le dernier ordre de la chaîne. Une jonction et
une dispersion ne sont jamais des attaques : elles ont une force de déplacement
pacifique de 0 et ne délogent personne.

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
(2 R) est crédité à la source alliée la plus proche et peut réduire une famine.

### Dispersion (`D`)

`XXX D DEST1 DEST2 ...` traite les destinations dans leur ordre d'apparition,
avec au plus une troupe par destination. C'est un partage pacifique à force 0 :
il ne combat pas une armée présente ; une destination libre et non contestée est
prise, tandis qu'une destination contestée repousse cette affectation et ne
reçoit pas de troupe.

- une destination est adjacente à `XXX` ou égale à `XXX` ; les destinations
  peuvent se répéter ;
- une destination occupée, combattue ou sans troupe disponible ne consomme pas
  de troupe ; une destination suivante peut néanmoins recevoir une troupe ;
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

---

## 5. Ordres d'hiver

L'hiver n'accepte **aucune chaîne ni mouvement** : uniquement des
investissements directs, une ligne par ordre, appliqués dans l'ordre saisi.

| Investissement | Syntaxe | Condition | Coût (R) |
|---|---|---|---|
| Recruter un noble | `R N XXX` | `XXX` contrôlé, avec un château ou un village et une armée du joueur | 2 |
| Recruter une troupe | `R T XXX` | `XXX` contrôlé, et un noble libre du joueur sur `XXX` ou adjacent | 1 |
| Construire ou améliorer un moulin | `C M XXX` | `XXX` contrôlé ; nouveau moulin sur case **vide** adjacente à un château ou village productif, ou moulin existant adjacent à cette source | 3 |
| Construire un château | `C C XXX` | `XXX` contrôlé | 10 |
| Construire un dépôt de vivres | `C D XXX` | `XXX` contrôlé | 3 |
| Désigner une capitale | `E C XXX` | un château contrôlé sur `XXX` | 0 |
| Placer un noble en otage | `O N NNN` | `NNN` est un prisonnier adverse détenu par le joueur | 0 |
| Placer un noble au donjon | `P N NNN` | `NNN` est un prisonnier adverse détenu par le joueur | 0 |
| Libérer un noble | `L N NNN` | `NNN` est détenu par le joueur ; la capitale de son propriétaire contient une armée de celui-ci | 0 |

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
  investissements ;
- une **ration** est une unité de nourriture consommée pendant le
  ravitaillement d'une saison d'action. Les rations locales sont produites et
  distribuées sur place ; elles ne deviennent pas automatiquement du stock `R` ;
- le **stock** est donc la quantité de `R` conservée sur une case.

Une source est chaque château ou village contrôlé. Chaque source produit `1 R`
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
`structure_present` : une case ne porte jamais deux infrastructures.

**Paiement** : le coût est prélevé d'abord sur le stock de la case ciblée, puis
sur la source contrôlée la plus proche ; si la réserve totale est insuffisante,
**aucun paiement partiel** n'est effectué et l'investissement est rejeté
(signalé dans le rapport, coût non perdu).

Exemple : un `C M ATL` coûtant 3 R consomme 1 R du stock d'ATL puis 2 R de la
source contrôlée la plus proche. Si ces deux stocks ne totalisent que 2 R, la
construction est rejetée et aucun des 2 R n'est retiré.

**Fin de l'hiver** :

- chaque stock restant est conservé à hauteur de `ceil(stock / 2)` ;
- les stocks hors capitale sont rapatriés vers la capitale, en laissant au
  maximum **1 R par village** et **2 R par château** ;
- sans capitale, les stocks restent sur place.

Il n'est pas nécessaire de tout dépenser avant la fin de l'hiver : le stock non
dépensé est d'abord conservé, puis le surplus est rapatrié selon ces plafonds.
Un stock de 5 R devient donc 3 R avec `ceil(5 / 2)`. La conservation et le
rapatriement sont effectués après les investissements, et une case sans château
ni village ne conserve pas de stock. Exemple : un village hors capitale garde
au plus 1 R après conservation ; le surplus rejoint la capitale, tandis qu'un
château hors capitale peut garder 2 R.

---

## 6. Armées, combats et logistique

### Armées et force

Une armée est l'unique entité de force d'un territoire : elle porte un
propriétaire et une taille en troupes. Toutes ses troupes partagent la même
chaîne ; il n'existe pas d'ordres mixtes au sein d'une armée.

- la force d'une attaque est la **taille** de l'armée attaquante, avec **+1** si
  un noble libre allié est présent sur sa case ;
- la force d'un soutien est la taille de l'armée soutenante, avec **+1** si un
  noble libre allié est présent sur sa case ;
- la défense d'une armée reçoit le même bonus de **+1** lorsqu'elle est
  commandée par un noble libre allié ;
- un château apporte un bonus défensif fixe de **+1**, même sans armée ;
- la plus haute force **strictement unique** gagne ; une égalité au sommet
  produit un **statu quo**, y compris sur une case vide ;
- une armée délogée perd son déplacement et doit **battre en retraite** ;
- une retraite part en bloc vers une case adjacente libre, non combattue
  pendant le tour et différente de l'origine de l'attaquant (départage par
  trigramme croissant ; deux armées sans alternative sur la même case sont
  détruites).

Le contrôle d'un territoire suit l'armée qui s'y arrête ; un contrôle acquis
reste acquis après le départ de l'armée, jusqu'à l'arrêt d'une armée ennemie.

### Ravitaillement exponentiel

Le ravitaillement est résolu **au début de chaque saison d'action**, avant les
ordres, les combats et les déplacements. Il n'existe pas de phase de
ravitaillement en hiver. Une armée d'une seule troupe demande `1` ration :
elle n'est pas automatiquement gratuite.

Une armée de `N` troupes demande :

```text
coût = 2^(N - 1)  rations
```

| Taille | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Coût en rations | 1 | 2 | 4 | 8 | 16 |

La production vivrière de la case de l'armée lui est attribuée à elle seule :
une armée ne consomme que la production de la case qu'elle occupe, au plus
**une ration**, et le reste constitue sa demande à ravitailler. Il n'y a
jamais qu'une armée par case, donc aucune distribution entre armées : une
armée ennemie sur une case voisine ne prend jamais la ration de ta case.

Les brigands et autres armées neutres prennent eux aussi la ration de la case
qu'ils occupent, mais ne reçoivent aucun complément depuis les stocks des
sources contrôlées par un joueur.

Exemple : une armée de 2 troupes sur une colline portant un château
(production 1 + 2 = 3 rations) reçoit 1 ration ; sa demande restante est
2 − 1 = 1 ration à couvrir par ses sources. Une armée sur un marécage
(production 0) ne reçoit rien et doit couvrir toute sa demande.

**Production vivrière de la case** : 1 ration en plaine, forêt ou colline ;
0 ration en montagne ou marécage ; **+2 rations** supplémentaires si la case
porte un château ou un village.

**Sources de ravitaillement** : les **châteaux et villages contrôlés**. Un
château ou un village produit **1 R stockable par tour**. Le flux traverse les
cases alliées ou neutres et s'arrête devant une case ennemie. La portée de base
est de **3 cases** ; chaque dépôt de vivres contrôlé rencontré sur le trajet
ajoute **2 cases**. Un village neutre conserve son stock, inaccessible au joueur
avant capture.

Chaque source calcule sa propre production `R` : sa production de base, plus le
niveau de **chaque moulin adjacent**. Un même moulin peut donc alimenter toutes
les sources voisines ; il n'est pas réservé au propriétaire de sa case. Un
moulin orphelin, sans château ni village adjacent, produit `0 R`. Par exemple,
un village entouré de deux moulins de niveau 1 produit `1 + 1 + 1 = 3 R` ; les
mêmes moulins ajoutent aussi leur niveau à tout château voisin. La présence ou
la position d'un noble ne conditionne jamais `C M XXX` ni cette production : un
noble situé en NOR n'empêche pas le joueur de construire `C M ATL` si ATL est
vide, contrôlé et adjacent à la source requise.

### Stocks et famine

En cas de déficit :

1. les stocks des châteaux et villages contrôlés sont épuisés (du plus petit au
   plus grand, trigramme territorial en départage) ;
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
(uniquement hors hiver).

### Infrastructures

Une case ne porte qu'**une seule infrastructure**.

| Infrastructure | Condition | Effet v1 | Coût |
|---|---|---|---|
| Moulin | Construction sur case vide contrôlée, adjacente à un château ou village ; amélioration d'un moulin existant adjacent à cette source | +1 R stockable par niveau à **chaque** source adjacente | 3 |
| Dépôt de vivres | Aucune | +2 cases de portée de ravitaillement lorsqu'il est contrôlé | 3 |
| Château | Aucune | +1 défense, +2 rations, produit 1 R stockable par tour, ancre de ravitaillement | 10 |
| Village | Généré neutre, **non constructible** | +2 rations, produit 1 R stockable par tour, ancre après capture | — |

---

## 7. Nobles : capture, déplacement et capacité

Les nobles **chevauchent les armées** : ils suivent les déplacements, les
attaques, les jonctions, les dispersions et les retraites. Un noble ne compte ni
dans le ravitaillement ni dans les pertes d'un combat. Un noble libre du joueur,
présent sur la case de son armée, lui donne une seule fois **+1 de puissance** ;
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
le bonus de `+1` que si ce groupe le porte effectivement au moment du combat ou
de la défense.

Un joueur qui ne possède aucun noble libre ou otage apte à émettre n'a pas à
soumettre de chaînes pendant une saison d'action.

Les nobles affectés lors d'une **dispersion** doivent tous être répartis entre
les destinations (`*` ou `*NNN`), voir section 4.

---

## 8. Victoire

La règle de victoire prévue (documentée dans les spécifications) :

> Un joueur est **éliminé** lorsqu'il ne contrôle plus aucun territoire et ne
> possède plus aucune armée. Les nobles seuls ne maintiennent pas un joueur en
> lice. Le **dernier joueur vivant gagne** la partie.

**État actuel du serveur** : cette règle de fin de partie **n'est pas encore
appliquée par le moteur**. Les parties sont donc ouvertes : le cycle des saisons
et la résolution continuent tant que les joueurs soumettent des ordres. La
détection de l'élimination et la condition de victoire seront activées dans une
version ultérieure.
