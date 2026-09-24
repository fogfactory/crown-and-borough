# Règles du jeu — Crown & Borough

**Version sociable (joueurs humains).** Ce document décrit les règles
actuellement actives sur le serveur de jeu. Les valeurs chiffrées proviennent
d'`assets/balance.yaml`, qui reste la source des nombres à jouer ; en cas de
divergence, le moteur fait foi.

---

## 1. Le pitch

Crown & Borough est un jeu de stratégie médiévale par tours, sur une carte de
territoires reliés par un graphe. Les joueurs soumettent leurs ordres en
secret ; le moteur résout tout le monde **en même temps**, saison après
saison.

Une seule contrainte structure tout le reste des règles : **les ordres ne
sortent pas d'une armée, ils sortent d'un noble**, et chaque joueur n'en
possède que quelques-uns. Un noble libre ou otage n'émet qu'**une chaîne par
tour** — une chaîne étant une suite d'ordres écrite à l'avance pour une armée
entière. Comme tu ne peux pas reprogrammer chaque armée chaque tour, tu dois
anticiper : décider aujourd'hui ce qu'une armée fera dans deux ou trois tours,
et composer avec l'incertitude de ce que font les autres joueurs pendant ce
temps. C'est de là que vient tout le vocabulaire de « chaîne », de « liaison »
(single/loop) et de « réception » détaillé section 4 : ce ne sont pas des
artifices techniques, c'est la traduction directe de la rareté des nobles.

Les deux piliers de tension du jeu :

- la résolution simultanée des intentions, des soutiens et des combats —
  personne ne voit ce que les autres ont écrit avant la résolution ;
- la logistique exponentielle — les grandes concentrations de troupes coûtent
  cher à nourrir et deviennent vulnérables dès que le ravitaillement est coupé
  (section 7).

Une partie en ligne accepte de **2 à 8 joueurs** (jusqu'à 16 en local). La
carte et les données chiffrées (propriétaires, tailles d'armées, stocks,
infrastructures, position des nobles) sont visibles par tous en permanence.
Ce qui reste privé, ce sont les **intentions** : en ligne, un joueur ne voit
le détail exact d'une chaîne ou d'un combat que s'il y participe — la
section 3 le montre sur un exemple.

Chaque joueur démarre sur un territoire distinct, où un **château** est
construit gratuitement (sa **capitale**), avec {{starting_resources}} R de
stock, une armée de {{starting_troops}} troupes et {{starting_nobles}}
noble(s) libre(s).

La partie dure un nombre d'années choisi à sa création (10 par défaut) ; la
section 10 détaille la fin de partie et le score.

### Inspirations

Le projet s'inspire notamment de [Fief](https://boardgamegeek.com/boardgame/107704/fief)
pour son cadre féodal et ses enjeux territoriaux, ainsi que de
[Diplomacy](https://boardgamegeek.com/boardgame/483/diplomacy) pour la
programmation simultanée des ordres, les soutiens et la résolution des
affrontements. Ces jeux sont des inspirations de conception, pas des sources
de règles applicables à Crown & Borough.

---

## 2. Le tour en un coup d'œil

Une année comprend **quatre tours** : printemps, été, automne et **hiver**.
Le compteur de tour avance d'une unité à chaque saison, hiver compris.

Les trois saisons d'action (printemps, été, automne) fonctionnent toutes de
la même façon :

1. Chaque joueur prépare et soumet ses chaînes d'ordres pour ses nobles libres
   et ses armées.
2. Le moteur vérifie chaque soumission : une erreur de syntaxe ou de
   réception rejette la soumission fautive, sans toucher au reste de la
   partie (section 4).
3. Le moteur résout **tout le monde ensemble** : ravitaillement, intentions,
   soutiens, combats, déplacements, retraites, jonctions, dispersions et
   progression des chaînes.
4. Le contrôle territorial, la position des nobles et les événements sont mis
   à jour, puis un **rapport de tour** est produit.

Une armée n'exécute **qu'une seule ligne de sa chaîne par saison d'action** :
un ordre `A` ou `J` franchit donc au plus une case adjacente pendant cette
résolution. Par exemple, `ROS A BOI` puis `BOI A ATL` fait avancer une armée
de ROS à BOI ce tour-ci, puis de BOI à ATL au tour d'action suivant. Une
chaîne de plusieurs lignes s'étale donc sur plusieurs tours, et reste
attachée à l'armée entre deux résolutions tant qu'elle n'est ni terminée ni
cassée — la section 3 déroule un exemple complet.

L'hiver est différent : c'est une **trêve de gestion**, sans chaîne, sans
mouvement, sans combat et sans ravitaillement. Le joueur soumet une liste
d'investissements directs, traités dans l'ordre où il les a écrits
(section 8).

| Saison | Ce qui s'y passe |
|---|---|
| Printemps, été, automne | Ravitaillement calculé en premier, puis intentions, soutiens, combats, déplacements, jonctions, dispersions et progressions de chaînes résolus ensemble. Une seule ligne courante par armée. |
| Hiver | Pas de ravitaillement ni d'ordre de chaîne : les investissements sont appliqués un par un, dans la liste saisie, puis les stocks sont conservés et rapatriés. |

Les ordres de printemps, été et automne ne forment donc pas une file d'attente
entre joueurs : chacun est évalué avec les intentions du tour entier. L'hiver,
à l'inverse, est une phase strictement séquentielle.

---

## 3. Un tour joué, de bout en bout

Voici un tour complet, pour donner un visage concret au vocabulaire des
sections suivantes.

**La situation.** Hugues possède ROS (sa capitale, un château) avec une armée
de 2 troupes et son noble HUG, ainsi que FOU, une petite garnison d'1 troupe
où se trouve son second noble, ODA. ROS et FOU sont tous deux adjacents à
ATL, tenu par Brune : une armée de 2 troupes et son noble MIA. ATL est
adjacent à NOR, un territoire vide que Brune contrôle.

Hugues veut prendre ATL. Comme HUG et ODA sont deux nobles distincts, il peut
leur faire émettre chacun une chaîne ce tour-ci — c'est précisément parce
qu'il a deux nobles qu'il peut combiner une attaque et un soutien dans la
même résolution.

**Ce que Hugues écrit.** Une chaîne par noble, une ligne par ordre (l'en-tête
de noble est ajouté automatiquement par l'interface web) :

```text
HUG
ROS A ATL        # attaque ATL depuis ROS
```

```text
ODA
FOU S ROS - ATL  # soutien offensif de l'attaque ROS -> ATL
```

**Ce que Brune écrit**, sans connaître les intentions de Hugues (résolution
simultanée oblige) :

```text
MIA
H ATL            # tient sa position
```

**La résolution.** Le moteur additionne les forces engagées sur ATL :
l'attaque de Hugues pèse 2 (l'armée de ROS) + 1 (le soutien de FOU) = 3 ; la
défense de Brune pèse 2. Pour ne pas alourdir cet exemple, on ignore ici le
bonus de noble détaillé section 6 — il s'appliquerait exactement pareil des
deux côtés de ce calcul. 3 contre 2 : Hugues l'emporte, son armée occupe ATL.
L'armée de Brune est délogée et doit battre en retraite ; NOR est vide et
contrôlé par elle, c'est donc sa destination (section 6 détaille l'ordre de
priorité complet). Le noble MIA suit son armée jusqu'à NOR.

**Ce que chacun voit ensuite.** Les deux chaînes engagées n'avaient qu'une
ligne : elles sont terminées, et les deux armées de Hugues sont désormais
Sans Ordre pour la prochaine saison, à moins qu'il n'émette de nouvelles
chaînes. Côté rapport, Brune a participé au combat comme défenseure : elle
voit le détail exact des forces (3 contre 2) et le soutien engagé. Un
troisième joueur qui n'aurait été impliqué ni comme attaquant, ni comme
défenseur, ni comme soutien, verrait seulement qu'un combat a eu lieu sur ATL
et son résultat général, sans le détail des forces (section 1).

---

## 4. Écrire une chaîne

Une chaîne, c'est le **trigramme du noble émetteur** en en-tête, suivi
d'**une ligne par ordre**. Chaque ligne d'ordre a la forme
`POSITION SYMBOLE [cibles...]`. Les commentaires commencent par `#`, les
lignes vides et la casse sont normalisées par le parser.

> Dans l'interface web, l'en-tête du noble est **ajouté automatiquement avant
> l'envoi** : tu écris uniquement les lignes d'ordres.

### Liaison des ordres : single ou loop

- **single** — une ligne sans parenthèses. La chaîne s'arrête au premier
  échec, et tout ce qui suit dans la chaîne est abandonné.
- **loop** — la ligne entière est entre parenthèses, `(…)`. L'ordre est
  retenté à chaque résolution jusqu'à sa réussite ; un maintien en loop met
  l'armée en veille. Une erreur mécaniquement impossible casse toujours la
  chaîne, même en loop.

Une chaîne n'est pas limitée à une seule saison : une ligne réussie fait
progresser l'index de la chaîne, et la ligne suivante attend la résolution
suivante — c'est ce qui se passerait si la chaîne de Hugues en section 3
avait une deuxième ligne `ATL A NOR` : elle attendrait le tour d'après pour
s'exécuter. Une ligne loop conserve volontairement le même ordre lorsqu'elle
doit attendre une ouverture, et un mouvement invalidé par le mauvais temps
met la chaîne en pause de la même façon : l'ordre reste en place et retente
la saison suivante.

Si une chaîne contient une erreur, **la soumission entière est refusée avec
la ligne à corriger**, et rien n'est reçu tant qu'elle n'est pas corrigée :
syntaxe, code inconnu, cases non adjacentes, jonction qui n'est pas le
dernier ordre, soutien de sa propre case, transfert vers sa propre case ou
affectation de nobles invalide dans une dispersion. L'interface signale ces
erreurs pendant la saisie. Le transfert `T` fait exception à l'adjacence : il
utilise le réseau de ravitaillement (section 5).

### Réception : à qui appartient l'ordre

- La chaîne est attachée **immédiatement et atomiquement** à l'armée présente
  sur la position de son premier ordre ; elle remplace la chaîne précédente
  de cette armée.
- Un noble libre ou otage n'émet qu'**une seule chaîne par tour** — c'est la
  contrainte centrale décrite section 1. Il peut commander n'importe quelle
  armée de son joueur : il n'a pas besoin d'être présent sur la position du
  premier ordre (c'est ainsi qu'ODA, resté à FOU, peut tout de même faire
  agir l'armée de FOU en section 3). Une chaîne ciblant une armée qui ne lui
  appartient pas, un noble au cachot, ou un noble ayant déjà émis, est
  rejetée.
- Si **plusieurs chaînes ciblent la même armée au même tour**, leur réception
  concurrente est invalidée : aucune n'est reçue, et l'armée ne reçoit pas de
  nouvelle chaîne pour ce tour. Une chaîne déjà portée reste inchangée.
- Une armée sans chaîne est **Sans Ordre** : elle ne reçoit aucune action
  automatique, mais reste défendable (section 6).
- Un joueur sans noble libre ou otage apte à émettre n'a pas à soumettre de
  chaînes pendant une saison d'action.

---

## 5. Aide-mémoire des ordres

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
| `D` | `XXX D DEST1 DEST2 ...` | Dispersion pacifique à force 0 : les destinations sont traitées dans leur ordre d'apparition, peuvent se répéter, et les troupes arrivant sur une même case sont empilées. |
| `T` | `XXX T YYY N` | Transfert de `N` ressources vers un château, un village ou une armée adverse via le réseau de ravitaillement. |

### Attaque (`A`) et jonction (`J`)

**L'essentiel** : `YYY` doit être adjacent à `XXX` par une frontière
franchissable ; l'armée entière s'y déplace. Une attaque peut y combattre une
armée ennemie (section 6 détaille le calcul de force). Une jonction ne
combat jamais : elle a une force de déplacement pacifique de 0 et est
repoussée si la destination est contestée.

**Cas particuliers** :

- une destination est contestée dès qu'au moins une attaque adverse y
  participe et qu'aucune armée attaquante ne remporte le combat ;
- si une attaque alliée remporte le combat sur `YYY`, une jonction visant la
  même case peut fusionner avec le vainqueur ; les attaques adverses qui
  perdent ce combat ne l'empêchent pas d'arriver ;
- une armée peut attaquer son propre château vide pour s'y installer sans que
  la défense du château ne la repousse (auto-capture) ;
- la jonction doit être le **dernier ordre** de la chaîne.

### Soutien (`S`)

**L'essentiel** : un soutien renforce une armée de n'importe quelle
nationalité, à condition que l'armée soutenue accomplisse effectivement
l'action annoncée.

- **défensif** (`XXX S YYY`) : renforce l'armée qui tient `YYY`, si `YYY` est
  adjacent à `XXX` (on ne se soutient pas soi-même) ;
- **offensif** (`XXX S YYY - ZZZ`) : renforce l'attaque de `YYY` vers `ZZZ` ;
  `XXX` et `YYY` doivent chacun être adjacents à `ZZZ`, et `YYY` doit être
  l'armée qui attaque effectivement `ZZZ` (c'est le cas de FOU en section 3,
  adjacent à la fois à ATL et à ROS).

**Cas particuliers** : une attaque ratée ne crée pas de malus supplémentaire
— l'armée soutenue reste soumise au résultat normal du combat, et sa propre
chaîne continue ou casse selon sa liaison. Une attaque venue d'une case
différente de la cible soutenue peut **couper** un soutien.

### Maintien (`H`) et pillage (`P`)

`H XXX` : l'armée reste sur place et peut recevoir un soutien défensif —
c'est ce que Brune écrit en section 3.

`P XXX` : détruit l'infrastructure de la case occupée ; un bonus de pillage
({{pillage_bonus}} R) est crédité à la source alliée la plus proche, et peut
réduire une famine (section 7).

### Dispersion (`D`)

**L'essentiel** : `XXX D DEST1 DEST2 ...` traite les destinations dans leur
ordre d'apparition, avec au plus une troupe par destination. C'est un
partage pacifique à force 0 : il ne combat jamais une armée ennemie. Une
destination libre et non contestée est prise ; une destination alliée
fusionne avec l'armée présente ; une destination contestée repousse
l'affectation, sans consommer de troupe.

**Cas particuliers** :

- une destination est adjacente à `XXX` ou égale à `XXX` ; les destinations
  peuvent se répéter ;
- une destination occupée par une armée ennemie, contestée, ou sans troupe
  disponible ne consomme pas de troupe ; une destination suivante peut
  néanmoins en recevoir une ;
- plusieurs dispersions alliées qui arrivent sur la même case sont empilées
  dans une seule armée ;
- les troupes qui ne peuvent pas être envoyées restent sur la case d'origine ;
  une liste plus courte que l'armée laisse donc un résidu sur place ;
- les nobles explicitement affectés suivent le groupe produit : `*` affecte
  tous les nobles restants, `*NNN` affecte le noble `NNN` ; les nobles non
  mentionnés restent à l'origine tant qu'une troupe y demeure. Si toutes les
  troupes quittent l'origine et qu'un noble présent n'a pas de groupe
  produit, l'ordre est invalide à l'exécution ;
- la chaîne portée par l'armée suit le **premier groupe listé** :
  `BRI D ATL NOR` fait suivre la chaîne au groupe d'ATL dès qu'ATL reçoit sa
  première troupe. Pour garder la chaîne sur place tout en envoyant des
  troupes ailleurs, il faut écrire `BRI D BRI ATL NOR` — on ne saute pas à
  NOR après l'échec d'ATL lorsque le résidu reste à BRI, cela invaliderait la
  suite de la chaîne ;
- en `single`, les destinations non traitées produisent une dispersion
  partielle et la chaîne progresse quand même ; en `loop`, le résidu retente
  jusqu'à l'arrivée d'une armée sur chaque destination — si l'armée s'épuise
  avant d'avoir traité toutes les destinations, l'ordre est invalide.

```text
BRI D ATL ATL              # deux troupes empilées dans l'armée arrivée à ATL
BRI D ATL                  # une troupe vers ATL, le résidu reste sur BRI
BRI D ATL*HUG NOR          # HUG vers ATL, l'autre unité vers NOR
BRI D BRI ATL NOR          # BRI garde la chaîne, les autres groupes se séparent
(BRI D ATL NOR)            # dispersion en boucle
```

### Transfert (`T`)

**L'essentiel** : `XXX T YYY N` est exécuté après le ravitaillement, par
l'armée en `XXX`. `YYY` doit être un château, un village, ou la case d'une
armée contrôlée par un autre joueur vivant ; un dépôt sans armée ne peut pas
recevoir. Le stock source peut exister sans infrastructure.

**Cas particuliers** :

- la route suit la portée de ravitaillement du donneur (`{{supply_range}}`
  cases, plus les dépôts contrôlés) ; toute armée adverse sur une case
  intermédiaire la bloque, mais une armée adverse en destination est
  autorisée ;
- une armée affamée ne peut pas transférer ; le montant est plafonné à
  `{{cost_base}}^(N - 1)` pour une armée de `N` troupes, sans déduire les
  rations locales ; l'armée ne fait aucun autre ordre ce tour ;
- un manque de stock n'a aucun effet et ne casse pas une chaîne `single` ; en
  `loop`, le transfert retente, et si le stock restant est inférieur au
  montant demandé, le reliquat est envoyé par une livraison partielle et
  l'ordre se termine.

---

## 6. Combats, forces et retraites

### Qui gagne un combat

Une armée est l'unique entité de force d'un territoire : elle porte un
propriétaire et une taille en troupes, et toutes ses troupes partagent la
même chaîne — il n'existe pas d'ordres mixtes au sein d'une armée.

- la force d'une attaque est la **taille** de l'armée attaquante, avec
  **+{{noble_command_bonus}}** si un noble libre allié est présent sur sa
  case ;
- la force d'un soutien est la taille de l'armée soutenante, avec le même
  bonus ;
- la défense d'une armée reçoit ce bonus dans les mêmes conditions ;
- un château apporte un bonus défensif fixe de **+{{castle_defense_bonus}}**,
  même sans armée — sauf si tous les attaquants appartiennent à son
  propriétaire (voir l'auto-capture, section 5) ;
- la plus haute force **strictement unique** l'emporte ; une égalité au
  sommet produit un **statu quo**, y compris sur une case vide.

C'est exactement le calcul déroulé en section 3 : 3 contre 2, sans égalité,
Hugues l'emporte.

### Battre en retraite

Une armée délogée perd son déplacement et doit battre en retraite en bloc,
vers une destination adjacente choisie par ordre de priorité décroissant :

1. case vide contrôlée par son propriétaire (avec ou sans château), même si
   elle a été combattue ce tour — c'est le cas de NOR pour Brune en
   section 3 ;
2. case vide non contrôlée par le retraité (neutre ou ennemie), sans château
   et non combattue ce tour ;
3. armée amie adjacente non délogée (priorité à la plus petite en troupes),
   avec fusion : l'hôte gagne `N − 1` troupes si la retraitante a `N ≥ 2`
   troupes, ou `1` troupe si `N = 1` (aucune perte). Plusieurs armées
   retraitantes peuvent fusionner séquentiellement sur un même hôte ami sans
   collision destructive.

À égalité dans une catégorie, la destination la plus proche d'un château ou
village contrôlé par le propriétaire du retraité l'emporte, puis l'ordre
lexicographique (trigramme croissant). Pour les armées amies, le tri se fait
par taille croissante, puis distance à la source contrôlée la plus proche,
puis trigramme croissant. La case d'origine de l'attaquant est toujours
exclue, et les châteaux neutres ou ennemis vides défendent contre une
retraite : ils ne sont jamais une destination valide. Deux armées qui doivent
reculer sur la même case vide sans alternative sont détruites. L'ordre de
traitement des armées en retraite suit le trigramme croissant de leur case
d'origine.

Le contrôle d'un territoire suit l'armée qui s'y arrête ; un contrôle acquis
reste acquis après le départ de l'armée, jusqu'à l'arrêt d'une armée ennemie.

### Les nobles pendant un combat

Les nobles chevauchent les armées : ils suivent les déplacements, les
attaques, les jonctions, les dispersions et les retraites — c'est ainsi que
MIA finit à NOR avec l'armée de Brune en section 3. Un noble ne compte ni
dans le ravitaillement ni dans les pertes d'un combat ; il peut rester seul
sur une case après la perte de son armée. Il n'existe pas, dans cette
version, de limite de nobles portés par une armée : une armée transporte
tous les nobles présents sur sa case.

Le bonus de +{{noble_command_bonus}} ne vient que d'un noble physiquement
présent sur la case de l'armée au moment du calcul : émettre une chaîne à
distance (section 4) ne téléporte pas le noble et ne donne aucun bonus à
l'armée distante. Un noble transféré volontairement suit le groupe qui
l'accueille lors d'une dispersion (`*` ou `*NNN`, section 5) ; il n'apporte
son bonus de commandement que si ce groupe le porte effectivement au moment
du combat ou de la défense.

Lorsqu'une armée portant des nobles est **détruite** sur une case occupée par
une armée ennemie, ces nobles sont capturés et deviennent `hostage` par
défaut. Un noble otage peut continuer à émettre une chaîne (section 4) ; seul
son passage au cachot, décrit section 8, lui retire cette capacité.

---

## 7. Logistique et ravitaillement

### Le coût exponentiel d'une armée

Le ravitaillement est résolu **au début de chaque saison d'action**, avant
les ordres, les combats et les déplacements ; il n'existe pas de phase de
ravitaillement en hiver. Une armée de `N` troupes demande :

```text
coût = {{cost_base}}^(N - 1)  rations
```

| Taille | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Coût en rations | {{army_cost.1}} | {{army_cost.2}} | {{army_cost.3}} | {{army_cost.4}} | {{army_cost.5}} |

Une armée d'une seule troupe demande déjà `{{army_cost.1}}` ration : elle
n'est pas automatiquement gratuite. C'est cette progression qui rend une
grosse armée fragile dès que sa route de ravitaillement est coupée.

### D'où vient la nourriture

La production de la case occupée par une armée lui est attribuée à elle
seule, jusqu'à hauteur de sa demande ; le surplus est perdu, et le reste
constitue sa demande à ravitailler. Il n'y a jamais qu'une armée par case,
donc aucune distribution entre armées : une armée ennemie sur une case
voisine ne prend jamais la ration de ta case.

**Production vivrière d'une case (en rations)** : plaine
{{ration_terrain.plain}} ; forêt {{ration_terrain.forest}} ; colline
{{ration_terrain.hill}} ; montagne {{ration_terrain.mountain}} ; marécage
{{ration_terrain.swamp}} ; **+{{infra_rations_bonus}}** si la case porte un
château ou un village.

Exemple : une armée de 2 troupes sur une colline avec château (production
locale {{ration_terrain.hill}}, bonus château {{infra_rations_bonus}}) reçoit
2 rations, soit toute sa demande. La même armée sur un marécage (production
{{ration_terrain.swamp}}) ne reçoit qu'1 ration et doit couvrir le reste
ailleurs.

**Sources de ravitaillement** : les châteaux, villages et caches contrôlés.
Un château ou un village produit {{base_production}} R stockable par tour ;
une case ordinaire n'a pas de production propre, mais son stock (s'il y en a)
sert de cache. Le flux traverse les cases alliées, neutres ou contrôlées par
un autre joueur, et ne s'arrête que devant une case occupée par une armée
adverse. La portée de base est de {{supply_range}} cases ; chaque dépôt de
vivres contrôlé rencontré sur le trajet ajoute {{depot_range_bonus}} cases.
Un village neutre conserve son stock, inaccessible avant capture.

Chaque source calcule sa propre production en ajoutant le niveau de
**chaque moulin adjacent** : un même moulin peut alimenter toutes les sources
voisines, sans filtre de propriétaire, et un moulin orphelin (sans château ni
village adjacent) produit `0 R`. Par exemple, un village entouré de deux
moulins de niveau 1 produit `{{base_production}} + 1 + 1 R`. La présence ou
la position d'un noble ne conditionne jamais cette production.

### Stocks et famine

En cas de déficit :

1. les stocks des châteaux, villages et caches contrôlés sont épuisés en
   premier (du plus petit au plus grand, trigramme territorial en
   départage) ;
2. les armées restantes passent en **famine**, en commençant par les plus
   éloignées de leur source, puis les plus grosses, puis le trigramme
   décroissant.

Une armée en famine **combat et se défend à force 0** pour le tour, même si
elle porte un noble libre. Si elle occupe une infrastructure, elle la
**pille automatiquement** ; le bonus de pillage, diminué de sa demande
résiduelle, peut la sortir de famine. Si le pillage est insuffisant ou
impossible, elle perd **1 troupe**, sans jamais descendre sous 1 — mais elle
reste affamée et à force 0 pour toute la saison en cours, même si cette
perte rendait sa demande future soutenable ; la perte se répète à chaque
saison où l'armée reste affamée.

Exemple : une armée de 2 troupes en déficit demande 2 rations. Si ses stocks
et son pillage ne couvrent pas ce déficit, elle perd une troupe et passe à
1 troupe ; elle reste néanmoins à force 0 ce tour, même si une armée d'1
troupe ne demanderait ensuite qu'1 ration.

Dans l'interface, sélectionner une armée ou une source contrôlée affiche son
ravitaillement ou la zone qu'elle atteint (hors hiver). Un transfert en cours
de rédaction affiche aussi sa route et ses blocages.

### Ce que rapportent les infrastructures

Une case ne porte jamais qu'**une seule infrastructure** ; leur coût et leurs
conditions de construction sont détaillés section 8.

| Infrastructure | Effet v1 |
|---|---|
| Moulin | +1 R stockable par niveau à chaque source adjacente |
| Dépôt de vivres | +{{depot_range_bonus}} cases de portée de ravitaillement lorsqu'il est contrôlé |
| Château | +{{castle_defense_bonus}} défense, +{{infra_rations_bonus}} rations, produit {{base_production}} R stockable par tour, ancre de ravitaillement |
| Village | +{{infra_rations_bonus}} rations, produit {{base_production}} R stockable par tour, ancre après capture |

---

## 8. Ordres d'hiver

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

C'est ici, en hiver, que se règle le sort des nobles ennemis capturés en
combat (section 6) : `O`/`P` fait basculer un prisonnier entre `hostage` et
`dungeon` ; un otage peut encore émettre une chaîne pour son propriétaire
d'origine tant qu'il reste otage, mais plus une fois au donjon — le détenteur
peut d'ailleurs lire ces chaînes dans les parties en ligne, même lorsqu'elles
commandent une armée restée chez le propriétaire du noble. La capture
produit par défaut le statut `hostage`. `L N NNN` est émis par le
**détenteur**, pas par le propriétaire : si la capitale du propriétaire
existe et contient une armée de celui-ci, le noble y réapparaît libre ; sinon
l'ordre est rejeté.

Un transfert d'hiver ne se limite donc pas aux villages et châteaux du
donneur : `G` peut alimenter directement une structure contrôlée par le
joueur destinataire. Le débit, lui, suit les règles habituelles et ne peut
utiliser que les réserves de paiement du donneur.

Un moulin commence au niveau 1 et peut atteindre le niveau 3 inclus. La
construction coûte {{costs.mill_levels.0}} R ; les améliorations vers les
niveaux 2 et 3 coûtent respectivement {{costs.mill_levels.1}} R et
{{costs.mill_levels.2}} R. `C M` sur un moulin déjà au niveau 3 est rejeté
avec le motif `mill_max_level_reached`, sans prélèvement. Les moulins
hérités de niveau supérieur à 3 restent productifs ; seules leurs nouvelles
améliorations sont bloquées.

Les investissements qui ciblent un territoire exigent le **contrôle de ce
territoire**. Une construction remplace la structure existante uniquement
quand la règle le prévoit : un **château construit sur un village remplace
le village** et conserve le stock de la case. Un moulin orphelin ne produit
rien.

### Vocabulaire des ressources

- `R` désigne une unité de **ressource stockable** : elle se trouve dans le
  stock d'une case, est produite par une source, et sert à payer les
  investissements lorsqu'elle se trouve dans un château ou un village
  contrôlé ;
- une **ration** est une unité de nourriture consommée pendant le
  ravitaillement d'une saison d'action (section 7) ; les rations locales ne
  deviennent pas automatiquement du stock `R` ;
- le **stock** est donc la quantité de `R` conservée sur une case.

Une source est chaque château ou village contrôlé, ainsi que toute case
contrôlée qui contient un stock positif pendant une saison d'action. Chaque
château ou village produit {{base_production}} R par tour, indépendamment
des autres sources : un deuxième château est donc une deuxième source, même
si un seul reste désigné capitale. Un moulin ajoute son niveau à chaque
source adjacente, y compris à travers les frontières de propriétaire — voir
section 7 pour le détail de cette production.

**Paiement** : le coût est prélevé d'abord sur le stock de la case ciblée,
puis sur la source contrôlée la plus proche ; si la réserve totale est
insuffisante, **aucun paiement partiel** n'est effectué et l'investissement
est rejeté (signalé dans le rapport, coût non perdu). Exemple : un `C M ATL`
coûtant {{costs.mill_levels.0}} R consomme d'abord le stock d'ATL, puis le
complément depuis la source contrôlée la plus proche ; si ces stocks ne
totalisent pas le coût requis, la construction est rejetée sans prélèvement
partiel.

**Fin de l'hiver** :

- chaque stock restant d'un château ou village est conservé à hauteur de
  `ceil(stock / {{winter_stock_divisor}})` — un stock de 5 R devient donc
  3 R ;
- un dépôt de vivres conserve intégralement son stock ; les stocks hors
  château, village et dépôt sont perdus ;
- les stocks des châteaux et villages hors capitale sont rapatriés vers la
  capitale, en laissant au maximum {{village_stock_cap}} R par village et
  {{castle_stock_cap}} R par château ;
- sans capitale, ces stocks restent sur place ; les stocks de dépôt restent
  sur leur case.

Il n'est pas nécessaire de tout dépenser avant la fin de l'hiver : le stock
non dépensé est d'abord conservé, puis le surplus est rapatrié selon ces
plafonds. La conservation et le rapatriement sont effectués après les
investissements.

---

## 9. Cartes spéciales et calamités

Les ordres jouables de cartes sont soumis dans un champ `special`, séparé des
chaînes de nobles et sans besoin de noble. Les défausses d'hiver sont écrites
dans la feuille `winter`.

- `P BT ROS` : jouer Beau temps sur la région dont ROS est le seed ;
- `P RA ROS` : jouer Bonne récolte sur cette région ;
- `P RE BRU` : jouer Révolte sur le territoire BRU, uniquement si une
  mauvaise récolte active affecte sa région ;
- `D C BT` ou `D C RA` : défausser une carte, en hiver uniquement.

La main est reconstituée automatiquement en hiver après les défausses ;
aucun ordre de pioche n'est nécessaire.

Beau temps, Bonne récolte et Révolte sont jouables au printemps, en été et
en automne, mais pas en hiver. Les cartes jouées sont consommées avant la
résolution des ordres d'armée. Beau temps annule uniquement le mauvais
temps, Bonne récolte annule uniquement la mauvaise récolte ; une carte qui
annule une calamité ne produit pas son bonus régional. Deux cartes du même
kind sont consommées, mais une seule est effective : avec une calamité
active, la première annule et une seconde applique le bonus régional ; sans
calamité, la première l'applique directement. Le bonus reste plafonné à une
unité par catégorie et par région ; les cartes au-delà sont consommées sans
effet.

Le deck contient **{{special_orders.deck_size}} cartes** :
**{{special_orders.card.plague}}** peste, **{{special_orders.card.bad_weather}}**
mauvais temps, **{{special_orders.card.famine}}** mauvaise récolte,
**{{special_orders.card.fair_weather}}** beau temps,
**{{special_orders.card.abundant_harvest}}** bonne récolte et
**{{special_orders.card.revolt}}** révolte. La main est limitée à
**{{special_orders.hand_limit}} cartes**, et chaque joueur reçoit
automatiquement jusqu'à **{{special_orders.draw_orders_limit}} cartes bonus
par hiver**, après ses défausses.

Une calamité tirée est programmée dans le premier slot disponible de l'année
suivante : printemps (**{{special_orders.calamity_slots.spring}}**), été
(**{{special_orders.calamity_slots.summer}}**) ou automne
(**{{special_orders.calamity_slots.autumn}}**). Sa région est tirée de
manière déterministe lors de la programmation. L'augure du printemps révèle
le kind, la saison et la région de toutes les calamités de l'année ; les
augures futures restent cachées. Dès son tirage, la calamité programmée est
annoncée dans l'encart des cartes spéciales de l'interface, et l'annonce
reste visible jusqu'à son application ou sa contre-mesure. Aucune calamité
ne se résout en hiver.

- la peste réduit les armées par division de
  **{{special_orders.effects.plague_army_divisor}}** et peut supprimer un
  noble ;
- le mauvais temps bloque les mouvements provenant ou visant sa région, sauf
  le maintien et le soutien défensif ;
- la mauvaise récolte désactive les moulins et les bonus de rations des
  infrastructures de sa région ;
- la Révolte se joue sur un territoire (`P RE TER`) pendant les saisons
  d'action, à condition que sa région subisse une mauvaise récolte. Chaque
  carte ajoute un jet entre **{{special_orders.effects.revolt_army_min_size}}**
  et **{{special_orders.effects.revolt_army_max_size}}** troupes à l'armée
  neutre commune du territoire ; le territoire peut être neutre (simple
  brigandage), et une armée y est créée si la case est vide. Si le
  territoire est occupé, la révolte se résout comme un combat entre l'armée
  révoltée et l'occupant : le perdant se retire ou est détruit. Si la
  mauvaise récolte de la région est annulée par une Bonne récolte, les
  révoltes en attente sur la région sont annulées et le joueur récupère sa
  carte. Une rébellion vaincue se retire comme toute armée défaite, au lieu
  de disparaître. Les armées neutres ne perdent jamais leur force à cause
  d'une famine, mais perdent une troupe en fin de tour si la production
  locale de leur territoire ne suffit pas à les nourrir.

Les rumeurs publiques sont recalculées dans chaque rapport à partir des
mains bonus actuelles de tous les joueurs. Elles apparaissent lorsqu'au
moins deux joueurs ont une carte en main, sans révéler le joueur ni
l'identifiant interne de la carte. Plusieurs cartes du même kind sont
regroupées en une seule phrase graduée : niveau 1 pour quelques cartes,
niveau 2 pour une présence plus marquée, niveau 3 pour une abondance
exceptionnelle. L'échelle est recalée sur la capacité de main de la partie
(joueurs × limite de main), afin qu'un même nombre de cartes ne produise pas
le même niveau dans une petite et une grande partie.

---

## 10. Fin de partie, score et victoire

La durée d'une partie est choisie à sa création, entre 1 et 50 années (10
par défaut). Une année compte quatre tours ; l'interface affiche l'année
historique `1000 + année`, soit « Année 1001 » au premier tour.

**Élimination** : un joueur est éliminé lorsqu'il ne contrôle plus aucun
territoire et ne possède plus aucune armée. Les nobles seuls ne maintiennent
pas un joueur en lice. Un joueur éliminé ne soumet plus d'ordres.

**Fin de partie** : la partie se termine immédiatement lorsqu'un seul joueur
reste en lice (il gagne), ou sinon après la résolution du dernier hiver de
la durée choisie — le joueur au score le plus élevé gagne. Une égalité
parfaite au sommet ne désigne aucun gagnant.

**Score**, recalculé après chaque tour et visible par tous :

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
territoire contrôlé. Un noble libre compte pour son propriétaire ; un noble
capturé, otage ou au donjon, compte pour le joueur qui contrôle le
territoire où il se trouve, et non pour son propriétaire d'origine.
