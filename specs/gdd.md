# Game Design Document : Crown & Borough v1

Ce document fixe le cœur de règles de la v1. Les futures politiques, les
ordres spéciaux et les autres compléments seront ajoutés séparément. Ils
devront préserver les invariants décrits ici plutôt que redéfinir les
mécaniques de base.

## 1. Vision et principes

`Crown & Borough` est un jeu de stratégie médiévale par tours, sur une carte
de territoires reliés par un graphe. Les joueurs programment des chaînes
d'ordres pour leurs armées ; toutes les chaînes sont résolues simultanément.

La tension de la v1 repose sur deux piliers :

- la résolution simultanée des intentions, des soutiens et des combats ;
- la logistique exponentielle, qui rend les grandes concentrations de troupes
  coûteuses et vulnérables.

La géographie est une connaissance commune. Les données chiffrées de la carte
(propriétaires, tailles d'armées, stocks, infrastructures et nobles) sont
visibles par tous dans la v1 actuelle. Une politique de divulgation plus
restrictive pourra être ajoutée plus tard sans modifier les combats ni le
ravitaillement.

## 2. Cycle du jeu et saisons

Une année comprend quatre tours : printemps, été, automne et hiver. Le compteur
`turn` progresse d'une unité à chaque saison, hiver compris.

### Tours d'action

Au printemps, en été et en automne :

1. Chaque joueur prépare et soumet ses chaînes d'ordres pour ses nobles libres
   et ses armées, ainsi que sa soumission `special` indépendante.
2. Le moteur vérifie les soumissions. Une erreur de syntaxe ou de réception
   empêche la résolution de la soumission concernée sans modifier l'état.
3. Le moteur consomme et agrège les cartes valides, puis applique leurs effets
   avant le ravitaillement et l'énumération des intentions d'armée.
4. Le moteur résout simultanément le ravitaillement, les intentions, les
   soutiens, les combats, les déplacements, les retraites, les jonctions, les
   dispersions et la progression des chaînes.
5. Le contrôle territorial, les déplacements de nobles et les événements sont
   mis à jour.
6. Le serveur construit un rapport de tour typé à partir des événements de la
   résolution.

La chaîne soumise est attachée immédiatement à l'armée présente sur la
position de son premier ordre. Elle remplace la chaîne précédente de cette
armée. Il n'y a pas de délai entre la soumission et la réception.

### Phase d'hiver

L'hiver est une trêve de gestion : aucune chaîne d'action, aucun mouvement,
aucun combat et aucun ravitaillement ne sont résolus. Le joueur soumet une
liste d'investissements directs, traités dans l'ordre saisi :

- `R N XXX` — recruter un noble sur `XXX` ;
- `R T XXX` — recruter une troupe sur `XXX` ;
- `C M XXX` — construire ou améliorer un moulin sur `XXX` ;
- `C C XXX` — construire un château sur `XXX` ;
- `C D XXX` — construire un dépôt de vivres sur `XXX` ;
- `E C XXX` — désigner le château de `XXX` comme capitale ;
- `O N NNN` — placer le noble prisonnier `NNN` en statut `hostage` ;
- `P N NNN` — placer le noble prisonnier `NNN` en statut `dungeon` ;
- `L N NNN` — libérer le noble de code `NNN`.

`XXX` est le trigramme du territoire ciblé, sauf pour `O N`, `P N` et `L N`,
qui ciblent un noble.

La soumission `special` est distincte de la feuille `winter` : elle contient
les ordres du deck, sans noble requis. `P KIND TER` est autorisé au printemps,
en été et en automne ;
`D C KIND` et `T C` sont réservés à l’hiver. La limite de main, la limite de
tirages, la taille et la composition du deck, ainsi que les capacités des slots
de calamité, sont chargées depuis `assets/balance.yaml`. La génération initiale du deck est déterministe à partir
de la seed de partie. Les investissements territoriaux exigent le contrôle du
territoire ciblé. Le
recrutement d'une troupe exige en outre un noble libre du joueur, situé sur la
cible ou sur un territoire adjacent à celle-ci par une frontière franchissable.
Le recrutement d'un noble exige une
infrastructure de peuplement (château ou village) et une armée du joueur sur la
case. Un ordre rejeté est signalé dans le rapport et son investissement est
perdu.

| Investissement | Coût en R |
|---|---:|
| Château | 10 |
| Moulin | 3 |
| Troupe | 1 |
| Noble | 2 |
| Dépôt de vivres | 3 |
| Changement de statut d'un noble | 0 |
| Libération d'un noble | 0 |

Les coûts sont prélevés d'abord sur le stock de la case ciblée, puis sur la
source contrôlée la plus proche. Si la réserve totale est insuffisante, aucun
prélèvement partiel n'est effectué.

À la fin de l'hiver :

- chaque stock restant est conservé à hauteur de `ceil(stock / 2)` ;
- les stocks sont rapatriés vers la capitale, en laissant au maximum 1 R par
  village et 2 R par château hors capitale ;
- sans capitale, les stocks restent sur place ;
- la saison suivante est le printemps.

Une partie accepte de 2 à 16 joueurs. Chaque joueur commence sur un territoire
distinct qui n'est pas un village neutre. Les territoires de départ sont séparés
d'au moins quatre étapes dans le graphe des frontières franchissables. Un
château y est construit gratuitement, devient la capitale par défaut, et le
joueur reçoit ses nobles, ses armées et ses ressources de départ selon
`assets/balance.yaml`. Les `N + 1` villages neutres générés sur la carte restent
distincts des `N` châteaux de départ. Ils servent de seeds à une partition
régionale statique, calculée par BFS multi-source sur les frontières franchissables
et publiée avec la carte.

Un joueur est éliminé lorsqu'il ne contrôle plus aucun territoire et ne possède
plus aucune armée. Les nobles seuls ne maintiennent pas un joueur en lice. Le
dernier joueur vivant gagne la partie.

## 3. Carte, terrains et villages

### Génération et graphe

La carte est générée de manière déterministe à partir d'une seed. Elle contient
`8 x joueurs` territoires de jeu et `(joueurs + 1) x 4` territoires
supplémentaires dédiés aux `joueurs + 1` villages neutres. Les territoires sont
des polygones nommés par une commune de `communes.csv` ; le trigramme de la
commune est l'identifiant unique du territoire, exactement trois lettres
majuscules, unique et stable pour une seed donnée.

Chaque frontière géométrique commune à deux territoires est conservée et
qualifiée :

- une frontière franchissable appartient au graphe des déplacements ;
- une frontière infranchissable reste visible mais ne permet pas le passage ;
- il n'existe pas de liaison artificielle sans frontière commune ;
- le graphe franchissable est connexe ;
- le degré franchissable de chaque territoire est compris entre 2 et le maximum
  du terrain : 3 en montagne, marécage ou colline, 5 en plaine ou forêt.

Les armées se déplacent d'une case adjacente au plus par résolution, quelle que
soit la nature du terrain. Le terrain influence la production de rations et les
contraintes de génération, pas une vitesse de déplacement cachée.

### Production

Les territoires sauvages ne produisent pas de ressource `R` stockable. La
production vivrière instantanée, consommée sur place et perdue si elle n'est
pas utilisée, vaut :

- 1 ration en plaine, forêt ou colline ;
- 0 ration en montagne ou marécage ;
- 2 rations supplémentaires si la case porte un château ou un village.

Une case ne porte qu'une seule infrastructure.

Un village est une infrastructure rare et neutre à la génération. Il produit 1
R stockable par tour et conserve son stock, qu'il soit neutre ou contrôlé. Un
joueur ne peut utiliser le stock d'un village neutre qu'après sa capture. Un
château construit sur un village le remplace et conserve le stock de la case.

## 4. Information et divulgation par joueur

La géographie de `map.json` est commune. Dans la v1, les éléments de la couche
dynamique restent également visibles : contrôle, ressources, taille des
armées, infrastructures et localisation des nobles. Le brouillard de guerre
qui pourrait réduire ces informations est une extension séparée.

La divulgation des ordres et des combats suit cependant une règle distincte.
Elle limite les chiffres qui décrivent les intentions et les affrontements :

- **Chaînes connues :** un joueur connaît les chaînes qu'il a lancées. Le
  détenteur d'un noble otage connaît également les chaînes émises par ce noble,
  même lorsqu'elles sont lancées par son propriétaire. Cette connaissance reste
  valable tant que la chaîne reste compatible avec la progression de l'armée.
  Si une chaîne différente, lancée par un noble adverse, remplace celle-ci, le
  propriétaire de l'armée connaît le remplacement et la nouvelle chaîne n'est
  pas révélée aux tiers par cette seule réception. Un tiers qui connaissait la
  chaîne précédente conserve cette information tant que les actions publiques
  de l'armée restent compatibles avec sa trajectoire connue ; une action
  contradictoire invalide alors cette connaissance.
- **Combats impliqués :** un joueur reçoit le résultat exact d'une attaque dans
  laquelle il intervient comme attaquant, défenseur ou soutien. Cela comprend
  les forces pertinentes et le résultat du combat.
- **Combats non impliqués :** le joueur voit que les ordres ont été traités et
  leur résultat général, mais pas le détail des puissances engagées.

Cette règle concerne les chaînes et les combats. Elle ne constitue pas encore
un brouillard de guerre général sur les tailles d'armées ou les ressources.
La projection serveur filtrée par joueur est une fonctionnalité online à
réaliser ; le serveur v1 actuel renvoie encore la projection globale.

## 5. Armées, combats et logistique

### Armées et force

Une armée est l'unique entité de force d'un territoire. Elle porte un
propriétaire et une taille abstraite en troupes. Toutes les troupes d'une armée
partagent la même chaîne ; il n'existe pas d'ordres mixtes au sein d'une armée.

Une attaque est un déplacement vers une case adjacente. Deux attaques qui
convergent vers une même case restent des contendantes distinctes, même si
elles appartiennent au même joueur. Pour cumuler des forces, il faut un soutien
explicite.

Les règles de combat sont les suivantes :

- la force d'attaque est la taille de l'armée attaquante, augmentée de `+1` si
  elle est commandée par un noble ;
- la force d'un soutien est la taille de l'armée soutenante, augmentée de `+1`
  si elle est commandée par un noble ;
- la défense d'une armée inclut le même bonus de commandement de `+1`, en plus
  du bonus fixe d'un château le cas échéant ;
- une attaque peut couper un soutien si elle vient d'une case différente de la
  cible soutenue ;
- toutes les intentions sont calculées ensemble avant les déplacements ;
- la plus haute force strictement unique gagne ;
- une égalité au sommet produit un statu quo, y compris sur une case vide ;
- une armée délogée perd son déplacement et doit battre en retraite ;
- une jonction et une dispersion ont une puissance de déplacement pacifique,
  n'attaquent pas et sont repoussées par une destination contestée ;
- un château apporte son bonus défensif fixe, même sans armée.

### Ravitaillement exponentiel

Une armée de `N` troupes sur une case demande :

`coût = 2^(N - 1)`

La production vivrière de la case est distribuée aux armées présentes, toutes
nationalités confondues, au plus une ration par armée et en commençant par la
plus grosse. Le reste constitue la demande à ravitailler.

Les châteaux et les villages contrôlés sont les sources de ravitaillement. Le
flux traverse les cases alliées ou neutres et s'arrête devant une case ennemie.
La portée de base est de 3 cases ; chaque dépôt de vivres contrôlé rencontré
sur le trajet ajoute 2 cases.

En cas de déficit :

1. les stocks des châteaux et villages contrôlés sont épuisés du plus petit au
   plus grand, avec le trigramme territorial comme départage ;
2. les armées restantes passent en famine, en commençant par les plus éloignées
   de leur source, puis les plus grosses, puis le trigramme décroissant.

Une armée en famine combat à force 0 pour le tour et ne peut que se déplacer à
force 0, même si elle est commandée par un noble. Si elle se trouve sur une
infrastructure, elle la pille
automatiquement. Le bonus de pillage, diminué de sa demande résiduelle, peut la
sortir de famine. Si le pillage est insuffisant ou impossible, elle perd une
troupe, sans jamais descendre sous 1 troupe. Elle reste
néanmoins en famine pour le tour en cours : même si cette perte rendait sa
demande future soutenable, la désorganisation lui conserve une force de 0 pour
ce tour.

## 6. Ordres et chaînes de commandement

### Syntaxe

Une chaîne commence par le trigramme du noble émetteur, suivi d'une ligne par
ordre. Les commentaires commençant par `#`, les lignes vides et la casse sont
normalisés par le parser.

```text
JEA # noble émetteur
BRI A ATL
BRI S ATL - NOR
(BRI S ATL)
(ATL A NOR)
H BRI
BRI J ROS
P BRI
BRI D BRI ATL NOR
```

Une ligne entre parenthèses est une transition `loop`. Une ligne sans
parenthèses est `single`.

Les ordres sont :

| Symbole | Syntaxe | Effet |
|---|---|---|
| `A` | `XXX A YYY` | Déplacement ou attaque vers `YYY` adjacente. |
| `S` | `XXX S YYY` ou `XXX S YYY - ZZZ` | Soutien défensif de `YYY`, ou soutien offensif de l'attaque `YYY` vers `ZZZ`. |
| `H` | `H XXX` | Maintien sur `XXX`. |
| `J` | `XXX J YYY` | Déplacement pacifique et jonction ; doit être le dernier ordre de la chaîne. |
| `P` | `P XXX` | Détruit l'infrastructure de la case occupée et crédite le bonus de pillage à la source alliée la plus proche. |
| `D` | `XXX D XXX YYY ...` | Dispersion pacifique : les destinations sont traitées dans leur ordre d'apparition, peuvent se répéter et reçoivent au plus une unité chacune ; les unités arrivées sur une même case sont empilées dans une seule armée. |

Un soutien défensif renforce une armée qui tient sa case. Un soutien offensif
renforce une attaque précise. Un soutien peut viser toute nationalité et ne
produit aucun effet si l'armée soutenue n'accomplit pas l'action annoncée.

Une chaîne `single` se casse au premier échec. Une chaîne `loop` retente l'ordre
jusqu'à sa réussite. Un maintien en boucle met l'armée en veille jusqu'à la
réception d'une nouvelle chaîne. Une dispersion traite chaque destination dans
son ordre d'apparition, sans introduire d'attaque : une destination occupée,
combattue ou sans unité disponible ne consomme pas d'unité et les unités
restantes demeurent à l'origine. En mode `single`, les destinations non traitées
font progresser la chaîne avec une dispersion partielle. En mode `loop`, le
résidu retente jusqu'à l'arrivée d'une armée sur toutes les destinations ; une
liste qui épuise l'armée avant d'avoir traité toutes ses destinations est
invalide à l'exécution.

Une armée sans chaîne est Sans Ordre et ne reçoit aucun soutien automatique.
Une erreur mécaniquement impossible casse immédiatement la chaîne, quel que
soit son mode de liaison. La non-adjacence d'un ordre est contrôlée lors de son
exécution : les ordres antérieurs restent valides et le suffixe est abandonné.

### Réception et capacité des nobles

Une chaîne est une émission complète. Un noble libre ou otage ne peut en émettre
qu'une par tour ; une nouvelle chaîne remplace celle portée par l'armée ciblée. La
chaîne s'applique à l'armée entière et son premier ordre indique explicitement
la position de réception.

Si plusieurs chaînes ciblent la même armée au même tour, leur réception
concurrente est invalidée : aucune de ces chaînes n'est reçue et l'armée ne
reçoit aucune nouvelle chaîne pour ce tour. Une chaîne déjà portée reste
inchangée.

Un noble `hostage` est détenu mais peut encore émettre une chaîne. Un noble
`dungeon` est au cachot et ne peut plus émettre. Les ordres d'hiver `O N NNN` et
`P N NNN` ne ciblent qu'un noble adverse détenu sur la case d'une armée du
joueur ; ils peuvent faire passer le statut de `hostage` à `dungeon` et
inversement. `hostage` est l'état par défaut après capture.

Les nobles chevauchent les armées : ils suivent les déplacements et les
retraites. Une dispersion peut affecter explicitement les nobles présents, avec
`*` pour tous les nobles restants ou `*NNN` pour un noble précis. Les nobles non
mentionnés restent à l'origine tant qu'une troupe y demeure ; si toutes les
troupes quittent l'origine et qu'un noble présent n'a pas de groupe produit,
l'ordre est invalide à l'exécution. Un noble ne compte ni dans le ravitaillement
ni dans les pertes d'un combat ; un noble libre commandant fournit uniquement le
bonus fixe de `+1` décrit à la section 5.

Lorsqu'une armée est détruite sur une case occupée par une armée ennemie, les
nobles qu'elle portait sont capturés et deviennent `hostage`. Ils peuvent encore
émettre des chaînes, dont le détail sera connu du détenteur dans les parties en
ligne. Le détenteur peut les libérer en hiver avec `L N NNN` ; si la capitale du
propriétaire existe et contient une armée de celui-ci, ils réapparaissent libres
dans cette capitale. Un joueur sans noble libre ou otage apte à émettre n'a pas
à soumettre de chaînes pendant une saison d'action.

### Retraites

Une armée défaite bat en retraite en bloc vers une case adjacente libre, non
combattue pendant le tour et différente de la case d'origine de l'attaquant.
Les retraites sont départagées par trigramme croissant. Deux armées qui doivent
reculer sur la même case sont détruites si aucune alternative ne subsiste.

## 7. Infrastructures et contrôle

Les infrastructures appartiennent à leur case. Le joueur qui contrôle la case
en bénéficie ; il n'y a pas de propriétaire stocké sur l'infrastructure.

| Infrastructure | Condition | Effet v1 | Coût |
|---|---|---|---:|
| Moulin | Construction sur case vide contrôlée, adjacente à un château ou village ; amélioration d'un moulin existant adjacent à cette source | +1 R stockable par niveau à chaque source adjacente | 3 |
| Dépôt de vivres | Aucune condition structurelle | +2 cases de portée de ravitaillement lorsqu'il est contrôlé | 3 |
| Château | Aucune | +1 défense, +2 rations, production de 1 R stockable par tour, ancre de ravitaillement | 10 |
| Village | Généré neutre, non constructible | +2 rations, production de 1 R stockable par tour, ancre après capture | — |

Un moulin isolé est orphelin et ne produit rien. Une construction remplace la
structure existante uniquement lorsque la règle de l'ordre le prévoit : un
château construit sur un village remplace le village et conserve le stock de
la case. Le contrôle reste acquis après le départ d'une armée jusqu'à l'arrêt
d'une armée ennemie.

## 8. Évolution du document

Le cœur décrit dans les sections 1 à 7 constitue la base v1. Les ajouts futurs
doivent être introduits sous forme de règles complémentaires : politiques,
ordres spéciaux, diplomatie enrichie ou brouillard de guerre. Ils seront
proposés et suivis dans GitHub avant d'être intégrés au document.

## 9. Durée de partie et score final

Une partie est créée avec une durée comprise entre 1 et 50 années, avec une
valeur par défaut de 10 années. Une année conserve exactement quatre tours :
printemps, été, automne et hiver. Le compteur interne `year` commence à 1 ;
l'interface affiche l'année historique `1000 + year`, soit `AN 1001` au premier
tour joué.

La partie se termine après la résolution du dernier tour de la durée choisie,
ou immédiatement lorsqu'un seul joueur reste en lice. Les scores sont recalculés
après chaque tour et sont visibles par tous les joueurs.

Le score d'un joueur est la somme des éléments suivants :

| Élément | Points |
|---|---:|
| Territoire contrôlé | 1 |
| Village contrôlé | 2 |
| Moulin contrôlé | 1 |
| Château contrôlé | 5 |
| Noble détenu | 2 |
| Troupe | 1 par unité dans ses armées |
| Ressource `R` | 1 par unité en stock sur ses territoires contrôlés |

Les points d'infrastructure et de ressource ne sont attribués que lorsque le
territoire est contrôlé. Un noble libre est compté pour son propriétaire. Un
noble capturé, qu'il soit otage ou au donjon, est compté pour le joueur qui
contrôle le territoire où il se trouve ; il ne compte pas pour son propriétaire
initial. Un territoire neutre ne rapporte aucun élément de score.

À la fin d'une partie, un unique survivant gagne toujours, même si la durée
vient d'être atteinte. Sinon, le joueur qui possède le score le plus élevé gagne.
Une égalité parfaite de score ne désigne aucun gagnant officiel.
