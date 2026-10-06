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
   avant l'énumération des intentions d'armée.
4. Le moteur résout simultanément les intentions, les soutiens, les combats,
   les déplacements, les retraites, les jonctions, les dispersions, les
   transferts de ressources et la progression des chaînes.
5. Le contrôle territorial et les déplacements de nobles sont mis à jour à
   partir des positions et des résultats de cette résolution.
6. Le moteur résout le ravitaillement de fin de tour sur ces positions et ce
   contrôle définitifs, territoires tout juste capturés compris : revenu
   territorial, production des moulins et rations, puis famine (section 5).
7. Le serveur construit un rapport de tour typé à partir des événements de la
   résolution.

La chaîne soumise est attachée immédiatement à l'armée présente sur la
position de son premier ordre. Elle remplace la chaîne précédente de cette
armée. Il n'y a pas de délai entre la soumission et la réception.

### Phase d'hiver

L'hiver est une trêve de gestion : aucune chaîne d'action, aucun mouvement,
aucun combat et aucun ravitaillement ne sont résolus. Le joueur soumet une
liste d'investissements directs, traités dans l'ordre saisi :

- `T N` — piocher une carte dans le deck de nobles (une fois par hiver) ;
- `R N CCC XXX` — jouer la carte de noble `CCC` de la main pour recruter ce noble sur `XXX` ;
- `D N NNN CCC` — jouer la carte de dignité `CCC` (`BAS` : bâtard) de la main sur le noble `NNN` ;
- `D C CCC` — défausser sans la jouer la carte `CCC` (trigramme de noble ou code de dignité) de la main de nobles, gratuit et sans limite par hiver ;
- `A N XXX` — annoblir gratuitement une armée sur `XXX` lorsque le joueur n'a plus aucun noble ;
- `R T XXX` — recruter une troupe sur `XXX` ;
- `C M XXX` — construire ou améliorer un moulin sur `XXX` ;
- `C C XXX` — construire un château sur `XXX` vide, ou fortifier le village
  de `XXX` ([economie.md](economie.md#village-fortifié)) ;
- `C D XXX` — construire un dépôt de vivres sur `XXX` ;
- `E C XXX` — désigner le château de `XXX` comme capitale ;
- `O N NNN` — placer le noble prisonnier `NNN` en statut `hostage` ;
- `P N NNN` — placer le noble prisonnier `NNN` en statut `dungeon` ;
- `L N NNN` — libérer le noble de code `NNN` ;
- `G XXX YYY N` — transférer `N` ressources du château ou village `XXX` vers
  le château ou village `YYY` d'un autre joueur ;
- `T F NNN XXX YYY ZZZ …` — constituer un fief : `NNN` est le noble titulaire,
  `XXX` la capitale (premier territoire, qui doit porter un château), suivi
  du reste du groupe (voir [titres.md](titres.md#constitution-dun-fief)) ;
- `T A NNN XXX` — attribuer le fief vacant de capitale `XXX` au noble
  `NNN` du joueur qui le détient.
- `M N XXX YYY` — marier le noble `XXX` du joueur au noble `YYY` d'un autre
  joueur ; l'ordre n'est conclu que si l'autre joueur soumet `M N YYY XXX`
  le même hiver (voir [succession.md](succession.md#conclusion-dun-mariage)).

`XXX` est le trigramme du territoire ciblé, sauf pour `O N`, `P N`, `L N` et
`M N`, qui ciblent des nobles. La feuille d'hiver peut aussi contenir `D C KIND` pour
défausser une carte bonus (voir « Cartes bonus et calamités » ci-dessous).

Les investissements territoriaux exigent le contrôle du territoire ciblé et
qu'il ne soit pas **occupé contre son contrôleur** (titres.md) : une armée
adverse ou une révolte y stationnant rejette l'ordre sans prélèvement
(`territory_occupied_by_other_player`). Le recrutement d'une troupe exige en
outre un noble libre du joueur, situé sur la cible ou sur un territoire
adjacent à celle-ci par une frontière franchissable.
Les nobles se recrutent depuis un deck de nobles partagé par tous les
joueurs et généré de façon déterministe à partir de la seed de partie : des
cartes de noble (nom, trigramme et sexe, à parts égales entre hommes et
femmes) et des cartes de dignité (le bâtard). `T N` ajoute la carte du dessus
à la main du joueur, une fois par joueur et par hiver (`noble_draw_already_used`
au-delà ; `hand_limit_reached` quand la main partagée — ordres spéciaux, cartes
de noble et de dignité — atteint `special_orders.hand_limit` ; `noble_deck_empty` quand la pioche et la défausse sont vides ; une
pioche vide est reconstituée en mélangeant la défausse). `D C CCC` défausse une carte
de la main de nobles sans la jouer : elle rejoint la défausse du deck telle quelle,
libère une place de main et n'est pas nommée dans le rapport public
(`card_not_in_hand` si elle n'est pas en main). `R N CCC XXX` joue une
carte de noble de la main : le noble apparaît sur `XXX`, qui exige une
infrastructure de peuplement (château ou village) et une armée du joueur, et
le joueur ne doit pas déjà posséder 4 nobles vivants (libres, otages ou au
cachot). Le recrutement ne coûte aucune ressource. Une carte absente de la
main est rejetée (`card_not_in_hand`). Chaque noble porteur de la dignité de
**bâtard** relève ce plafond de 1 (6 au plus), quel que soit son statut ;
ce noble est toujours le dernier de la ligne de succession, ne reçoit un
nouveau titre que s'il est le dernier de sa lignée, garde celui qu'il détient, ne peut pas être roi, est placé
directement au cachot quand il est capturé en combat, et son mariage n'est
pas une alliance. `D N NNN CCC` joue une carte de dignité sur un noble du
joueur (`noble_not_owned`, `noble_already_bastard`). Un ordre rejeté est
signalé dans le rapport avec son motif ; toutes les conditions sont
vérifiées avant de consommer la carte, donc un ordre rejeté ne la consomme
pas. La dignité de bâtard compte comme un titre dans le score. Une carte
jouée reste suivie tant que son noble vit ; à la mort de ce noble, sa carte de
noble sort du jeu et une nouvelle carte de noble du même sexe (prénom encore
libre, aucune s'il n'en reste pas) rejoint la défausse, tandis qu'une carte de
dignité y retourne, de même que lorsque la dignité est retirée par un effet.

| Investissement | Coût en R |
|---|---:|
| Château | 10 |
| Moulin | 3 (construction niveau 1), 5 (niveau 2), 7 (niveau 3) |
| Troupe | 1 |
| Noble | 2 |
| Dépôt de vivres | 3 |
| Changement de statut d'un noble | 0 |
| Libération d'un noble | 0 |
| Fief (par territoire du groupe) | 2 |
| Attribution d'un fief vacant | 0 |

Un moulin peut atteindre le niveau 3 inclus. Une construction coûte 3 R et les
passages aux niveaux 2 et 3 coûtent respectivement 5 R et 7 R. Un ordre `C M`
sur un moulin déjà au niveau 3 est rejeté avec le motif
`mill_max_level_reached`, sans prélèvement. Les moulins de niveau supérieur à 3
déjà présents dans une partie restent valides et productifs ; cette limite ne
bloque que les nouvelles améliorations.

Les coûts sont prélevés d'abord sur le stock de la case ciblée, puis sur la
source contrôlée la plus proche. Si la réserve totale est insuffisante, aucun
prélèvement partiel n'est effectué.

Seuls les stocks de châteaux et villages contrôlés, **non occupés contre leur
contrôleur**, sont des réserves de paiement en hiver ; les caches ordinaires
et les dépôts ne paient pas les investissements. Un moulin fait exception à
cette règle pour sa propre amélioration (`C M`) : il puise d'abord sur son
propre stock, puis sur celui de l'infrastructure qui recevrait sa production
(château en priorité, sinon village), avant de recourir au paiement d'hiver
habituel (voir [economie.md](economie.md#amélioration-dun-moulin)) — sous la
même réserve : une infrastructure occupée contre son contrôleur ne fait
jamais partie de ces sources.

À la fin de l'hiver :

- chaque stock restant d'un château, d'un village ou d'un moulin est conservé à
  hauteur de `ceil(stock / 2)` ;
- un stock situé dans un dépôt de vivres est conservé intégralement ;
- tout stock situé hors château, village, moulin ou dépôt est perdu ;
- les stocks des châteaux et villages sont rapatriés vers la capitale, en
  laissant au maximum 1 R par village et 2 R par château hors capitale ; le
  stock d'un moulin n'est jamais rapatrié ; une colonie occupée contre son
  contrôleur ne rapatrie pas non plus son stock, qui y reste et suit la
  conservation normale ; une capitale remplacée par `E C` ce même hiver
  rapatrie encore son surplus en tant qu'ancienne capitale avant de perdre son
  ancrage (voir [titres.md](titres.md#contrôle-et-occupation)) ;
- sans capitale, les stocks restent sur place ;
- un territoire contrôlé hors fief qui ne porte plus d'ancre (ni la capitale
  du joueur, ni une armée du joueur) redevient neutre à la fin de l'hiver,
  après ce rapatriement (voir [titres.md](titres.md#contrôle-et-occupation)) ;
- tout fief encore vacant (sans titulaire) est **attribué par défaut** au
  premier noble de la ligne de succession du joueur qui le détient, quel que
  soit son statut, avec un avertissement dans le rapport ; sans aucun noble
  vivant à ce moment, le fief reste simplement vacant
  (il n'est jamais dissous faute d'attribution, voir
  [titres.md](titres.md#perte-et-vacance-dun-fief)) ;
- la saison suivante est le printemps.

Les stocks hors château et village ne peuvent pas payer les investissements
hivernaux, à l'exception du stock d'un moulin pour sa propre amélioration (voir
ci-dessus). Un transfert d'hiver débite un château ou village contrôlé par le
donneur, mais peut viser directement le château ou village contrôlé par un autre
joueur ; la destination n'a pas besoin d'appartenir au donneur.

### Cartes bonus et calamités

Chaque joueur détient une main de cartes bonus. Pendant l'hiver, la feuille
d'hiver peut contenir `D C KIND` pour défausser une carte, sans noble requis ;
la main est ensuite reconstituée automatiquement, après les ordres d'hiver et
les défausses. La limite de main (`special_orders.hand_limit`) est partagée
avec les cartes du deck de nobles (nobles et dignités), et un joueur pioche au
plus `special_orders.draw_orders_limit` cartes par hiver tous decks confondus,
dont au plus une du deck de nobles : le remplissage n'excède ni ce plafond,
diminué de la pioche de noble éventuelle, ni les places libres de la main
partagée. Il n'existe pas d'ordre de
pioche (`T C`). Les cartes jouables passent par une soumission `special`
distincte des chaînes : `P KIND TER` est autorisé au printemps, en été et en
automne.

La limite de main, le remplissage automatique, la taille et la composition du
deck, ainsi que les capacités des slots de calamité, sont chargées depuis
`assets/balance.yaml`. La génération initiale du deck est déterministe à partir
de la seed de partie. Au printemps, l'augure révèle le kind, la saison et la
région de toutes les calamités de l'année ; les augures futures restent cachées.

Les calamités et les cartes bonus sont appliquées avant la résolution
simultanée des ordres d'armée :

- la peste réduit les armées et peut affecter les nobles ;
- le mauvais temps bloque les déplacements provenant de sa région et arrête
  ses moulins ; le Beau temps double leur production ;
- la famine supprime les rations de terrain de sa région et son revenu
  territorial ; la Récolte abondante les double ;
- la révolte est une carte bonus conditionnelle qui crée des armées `NEUTRAL`.

Le détail des cartes est suivi dans [`ordres-speciaux.md`](ordres-speciaux.md).

### Joueurs, départ et élimination

Une partie accepte de 2 à 16 joueurs dans le moteur ; une partie en ligne est
limitée à 2 à 8 joueurs. Chaque joueur commence sur un territoire distinct qui
n'est ni un village ni une montagne, et qui compte au moins deux voisins
franchissables eux-mêmes non montagneux : cette réserve garantit à la fois que
la garnison de départ ne meurt jamais de faim et qu'il existe assez de
territoires voisins viables pour y placer les avant-postes de départ. Les
territoires de départ sont séparés d'au moins quatre étapes dans le graphe des
frontières franchissables. Un château y est construit gratuitement et devient
la capitale par défaut. Le joueur y reçoit sa garnison, ses nobles et ses
ressources de départ selon `assets/balance.yaml` ; il reçoit en outre
`starting_outposts` armées d'une troupe chacune, placées dès le début de la
partie sur autant de territoires voisins non montagneux distincts de sa
capitale, retenus en priorité pour leur ration de terrain la plus élevée.

Chaque territoire de départ possède un village dédié, situé à exactement deux
étapes de lui dans le graphe franchissable, à au moins trois étapes de tout
autre territoire de départ, et à au moins deux étapes de tout autre village :
aucun village n'est adjacent à un territoire de départ. La carte porte en
outre `N + 1` chefs-lieux neutres, séparés d'au moins trois étapes de tout
territoire de départ et d'au moins deux étapes de tout village dédié. Un
chef-lieu et un village dédié sont des infrastructures identiques en jeu (même
revenu, même possibilité de fortification) ; seul le chef-lieu sert de seed à
la partition régionale statique, calculée par BFS multi-source sur les
frontières franchissables et publiée avec la carte via `regions[].seed`.

Un joueur est éliminé lorsqu'il ne contrôle plus aucun territoire et ne possède
plus aucune armée. Les nobles seuls ne maintiennent pas un joueur en lice. Le
dernier joueur vivant gagne la partie.

## 3. Carte, terrains et villages

### Génération et graphe

La carte est générée de manière déterministe à partir d'une seed. Elle contient
`8 x joueurs` territoires de jeu et `(joueurs + 1) x 4` territoires
supplémentaires, et porte `2 x joueurs + 1` villages neutres au total :
`joueurs + 1` chefs-lieux et un village dédié par territoire de départ (voir
« Joueurs, départ et élimination » ci-dessus). Les territoires sont des
polygones nommés par une commune de `communes.csv` ; le trigramme de la
commune est l'identifiant unique du territoire, exactement trois lettres
majuscules, unique et stable pour une seed donnée.

Chaque frontière géométrique commune à deux territoires est conservée et
qualifiée :

- une frontière franchissable appartient au graphe des déplacements ;
- une frontière infranchissable reste visible mais ne permet pas le passage ;
- il n'existe pas de liaison artificielle sans frontière commune ;
- le graphe franchissable est connexe ;
- le graphe franchissable ne contient aucun point d'articulation : toute paire
  de territoires, et donc toute paire de lieux-dits (châteaux de départ ou
  villages neutres), est reliée par au moins deux chemins sans territoire
  intermédiaire commun ;
- le degré franchissable de chaque territoire est compris entre 2 et le maximum
  du terrain : 3 en montagne, marécage ou colline, 5 en plaine ou forêt.

Lors de l'élagage des frontières, une frontière montagne/montagne ou
montagne/marécage est supprimée avec une probabilité de 50 %. Les autres
frontières non plain/plain ont une probabilité de suppression de 15 % et les
frontières plain/plain restent franchissables ; aucune suppression n'est
acceptée si elle crée un point d'articulation ou rompt la connexité.

Les armées se déplacent d'une case adjacente au plus par résolution, quelle que
soit la nature du terrain. Le terrain influence la production de rations et les
contraintes de génération, pas une vitesse de déplacement cachée.

### Production

Les territoires sauvages ne produisent pas de ressource `R` stockable. La
production vivrière instantanée, consommée sur place et perdue si elle n'est
pas utilisée, vaut :

- 2 rations en plaine ;
- 1 ration en forêt ;
- 1 ration en colline ;
- 0 ration en montagne ;
- 1 ration en marécage.

Un château ou un village n'ajoute aucune ration : seul le terrain nourrit une
armée sur place. La famine supprime les rations de terrain de sa région ; la
Récolte abondante les double. La table est dans `assets/balance.yaml`
(`ration_terrain`).

Une case ne porte qu'une seule infrastructure.

C'est le territoire contrôlé, et non l'infrastructure bâtie, qui produit la
ressource `R` stockable : chaque saison d'action (jamais en hiver), chaque
territoire contrôlé rapporte `territory_income` R, plus `village_income` R
s'il porte un village (voir `assets/balance.yaml`). Ce revenu est crédité en
fin de tour, avec le reste du ravitaillement (section 5), sur le contrôle
final du tour : un territoire capturé pendant le tour verse son revenu à son
nouveau contrôleur, pas à celui du début de tour. Un territoire membre d'un
fief le verse à la **capitale du fief** plutôt qu'à la capitale du joueur, y
compris lorsque le territoire producteur ou la capitale du fief elle-même est
occupée par une armée adverse : ce revenu n'est jamais intercepté par
l'occupant (voir [titres.md](titres.md#contrôle-et-occupation)). Hors fief, il
est versé directement au stock de la capitale du joueur ; sans capitale, le
revenu de chaque territoire va au château contrôlé le plus proche, sinon au
village contrôlé le plus proche, sinon il est perdu. La famine supprime ce
revenu dans la région du territoire qui le produit ; la Récolte abondante le
double, comme pour les rations.

Un village est une infrastructure rare et neutre à la génération ; il ne
peut pas être construit. Neutre, il produit `village_income` R par tour dans
son propre stock, qu'un joueur ne peut utiliser qu'après sa capture. Contrôlé,
il ne produit plus rien localement : son revenu suit la règle ci-dessus. Un
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
En ligne, le serveur applique cette règle et ne renvoie à chaque joueur que
sa projection filtrée ; un hôte observateur reçoit la projection complète. La
session hotseat locale renvoie la projection globale, sauf lorsqu'un joueur
est demandé explicitement.

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
- une attaque coupe un soutien si elle vise l'armée soutenante, vient d'un
  autre joueur et d'une case différente de la cible soutenue (la destination
  d'un soutien offensif, la case tenue d'un soutien défensif), même si
  l'attaque échoue ; un soutien est aussi coupé lorsque l'armée soutenante est
  délogée ;
- toutes les intentions sont calculées ensemble avant les déplacements ;
- la plus haute force strictement unique gagne ;
- une égalité au sommet produit un statu quo, y compris sur une case vide ;
- une attaque ne déloge jamais une armée de son propre joueur qui reste sur
  place : elle échoue, mais sa force continue d'empêcher les autres attaques
  d'entrer sur cette case ;
- les soutiens apportés par le joueur d'une armée ne comptent pas pour la
  déloger : pour gagner, l'attaque doit dépasser la défense et chaque autre
  attaque sans ces soutiens ; ils comptent en revanche pour empêcher les autres
  attaques d'entrer ;
- lorsque deux armées s'attaquent mutuellement (face-à-face), chacune oppose sa
  force sans les soutiens du joueur adverse : la plus forte l'emporte et déloge
  l'autre si elle bat aussi les autres attaques sur sa destination ; à égalité,
  ou entre armées du même joueur, les deux échouent ;
- une attaque qui échoue continue d'empêcher les autres attaques d'entrer sur
  sa destination, même si son armée est délogée, sauf si elle a perdu un
  face-à-face ;
- des attaques en cercle (A vers B, B vers C, C vers A) réussissent toutes
  lorsque rien d'autre ne s'y oppose ;
- une armée délogée perd son déplacement et doit battre en retraite ;
- une jonction et une dispersion ont une puissance de déplacement pacifique,
  n'attaquent pas et sont repoussées par une destination contestée. Une jonction
  ou une troupe dispersée peut toutefois fusionner avec l'attaquant allié qui
  remporte le combat sur sa destination ; les attaques adverses qui perdent ce
  combat ne l'empêchent pas d'arriver. Si aucun attaquant n'atteint la case, la
  destination reste contestée et le mouvement pacifique est repoussé. Une destination est contestée
  lorsqu'au moins une attaque adverse y participe et qu'aucune armée
  attaquante ne remporte le combat (statu quo ou défense conservée) ;
- un château, ou un village fortifié ([economie.md](economie.md#village-fortifié)),
  apporte son bonus défensif fixe, même sans armée, tant qu'il reste
  **ancré** — membre d'un fief ou capitale d'un joueur (voir
  [titres.md](titres.md#contrôle-et-occupation)) — sauf si tous les attaquants
  appartiennent à son propriétaire (auto-capture d'une place amie vide) ; un
  château ou un village fortifié vide qui n'est ni membre d'un fief ni la
  capitale d'un joueur est **inerte** et n'apporte aucun bonus ; le château de
  la capitale d'un fief est une cité et apporte à la place un bonus fixe
  supérieur, sans cumul avec le bonus de château (voir
  [titres.md](titres.md#constitution-dun-fief)) — un village fortifié ne peut
  jamais être la capitale d'un fief, qui exige un château ;
- une jonction ou une dispersion dont l'origine est visée par une attaque,
  quel qu'en soit l'auteur (allié, ennemi, ou une attaque à force nulle faute
  de vivres), est annulée d'emblée : aucune de ses troupes ne part, qu'elle
  gagne ou perde ce combat ;
- lorsque des ordres dépendent les uns des autres en cycle, les mouvements
  réussissent dès qu'une résolution cohérente avec ces règles le permet
  (mouvement circulaire).

### Ravitaillement et famine

Le ravitaillement est résolu en fin de tour d'action (voir section 2), sur les
positions et le contrôle territorial définitifs du tour, territoires tout
juste capturés compris : revenu territorial, production des moulins, rations
de terrain, puis famine. Une armée de `N` troupes sur une case demande :

`coût = 2^(N - 1)`

La production vivrière de la case est consommée par l'armée qui l'occupe,
jusqu'à hauteur de sa demande. Le surplus est perdu ; le reste constitue la
demande à ravitailler.

Les châteaux, villages et caches contrôlés contenant un stock positif sont les
sources de ravitaillement. Une armée consomme en priorité le stock de sa case.
Le flux traverse les cases alliées, neutres ou contrôlées par un autre joueur et
ne s'arrête que devant une case occupée par une armée adverse. Un château, un
village ou un dépôt adverse sans armée ne bloque donc pas le flux. Une case
**occupée contre son contrôleur** (titres.md) n'est en revanche plus
elle-même une source ni un dépôt utilisable, ni pour le contrôleur ni pour
l'occupant. La portée de base est de 3 cases ; chaque dépôt de vivres
contrôlé, non occupé, rencontré sur le trajet ajoute 2 cases.

En cas de déficit :

1. les stocks des châteaux, villages et caches contrôlés sont épuisés du plus
   petit au plus grand, avec le trigramme territorial comme départage ;
2. les armées restantes passent en famine, en commençant par les plus éloignées
   de leur source, puis les plus grosses, puis le trigramme décroissant.

Une armée qui termine le tour en famine est marquée **affamée**, un statut qui
persiste jusqu'à la résolution de ravitaillement suivante. Pendant tout le
tour suivant, une armée affamée combat et se défend à force 0, même si elle
est commandée par un noble : le bonus de noble ne s'applique alors pas ; elle
ne peut pas émettre de transfert de ressources (`T`, section 6). Ce premier
tour de famine ne lui inflige aucune autre conséquence : ni pillage de
l'infrastructure de sa case, ni perte de troupe, ce qui laisse au joueur tout
le tour suivant pour réagir en déplaçant l'armée ou en lui envoyant des
ressources par transfert.

À la résolution de ravitaillement du tour suivant, son statut est recalculé
exactement comme celui de n'importe quelle armée : si elle a atteint une
source suffisante entre-temps, elle redevient valide dès ce tour. Si elle est
encore en déficit alors qu'elle est déjà affamée, elle pille automatiquement
l'infrastructure de sa case, si elle en occupe une : le bonus de pillage,
diminué de sa demande résiduelle, peut la sortir de famine. Si le pillage est
insuffisant ou impossible, elle perd une troupe, sans jamais descendre sous 1
troupe, et reste affamée pour le tour d'après.

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
| `J` | `XXX J YYY` | Déplacement pacifique et jonction ; doit être le dernier ordre de la chaîne. Si une attaque alliée remporte le combat sur `YYY`, la jonction fusionne avec son vainqueur. |
| `P` | `P XXX` | Détruit l'infrastructure de la case occupée et crédite le bonus de pillage à la source alliée la plus proche. |
| `D` | `XXX D XXX YYY ...` | Dispersion pacifique : les destinations sont traitées dans leur ordre d'apparition, peuvent se répéter et reçoivent au plus une unité chacune ; les unités arrivées sur une même case sont empilées dans une seule armée, y compris lorsqu'elles viennent de plusieurs armées alliées. |
| `T` | `XXX T YYY N` | Transfert d'action vers un château, un village ou une armée adverse via le réseau de ravitaillement. `N` est plafonné à `2^(taille - 1)`. |

Un soutien défensif renforce une armée qui tient sa case. Un soutien offensif
renforce une attaque précise. Un soutien peut viser toute nationalité et ne
produit aucun effet si l'armée soutenue n'accomplit pas l'action annoncée.

Une chaîne `single` se casse au premier échec. Une chaîne `loop` retente l'ordre
jusqu'à sa réussite. Un maintien en boucle met l'armée en veille jusqu'à la
réception d'une nouvelle chaîne. Une dispersion traite chaque destination dans
son ordre d'apparition, sans introduire d'attaque : une destination occupée par
une armée ennemie, contestée ou sans unité disponible ne consomme pas d'unité et
les unités restantes demeurent à l'origine. Une destination occupée par une
armée alliée, ou prise par l'attaquant allié qui y remporte le combat, reçoit
la troupe et la fusionne avec cette armée. En mode
`single`, les destinations non traitées font progresser la chaîne avec une
dispersion partielle. En mode `loop`, le résidu retente jusqu'à l'arrivée d'une
armée sur toutes les destinations ; une liste qui épuise l'armée avant d'avoir
traité toutes ses destinations est invalide à l'exécution.

Une destination occupée par une armée alliée peut recevoir une dispersion : les
unités s'y empilent avec l'armée présente. Plusieurs dispersions d'un même joueur
peuvent donc converger vers une destination vide ou alliée et produisent une
seule armée. Une dispersion adverse ne peut ni entrer dans cette pile ni partager
la destination ; si des propriétaires différents la contestent, aucune de ces
arrivées pacifiques ne s'effectue.

Un transfert qui manque de ressources n'a aucun effet et n'interrompt pas la
chaîne. En boucle, un transfert vide le reliquat du stock par une livraison
partielle avant de progresser. Une armée affamée (section 5) ne peut pas
émettre de transfert. Les stocks présents sur les cases
ordinaires sont des sources de ravitaillement pendant les tours d'action ;
l'armée locale les consomme en priorité.

Une armée sans chaîne est Sans Ordre et ne reçoit aucun soutien automatique.
Toute erreur statique d'une chaîne est contrôlée à la soumission : syntaxe,
forme d'un ordre, codes inconnus, non-adjacence, jonction qui n'est pas le
dernier ordre, soutien défensif de sa propre case, transfert vers sa propre
position ou affectation de nobles invalide dans une dispersion. La soumission
est alors refusée avec la ligne fautive à corriger, et aucune partie de la
chaîne n'est reçue. À l'exécution, seules les conditions du monde (position de
l'armée, infrastructure, contrôle, route, nobles présents) peuvent casser une
chaîne, quel que soit son mode de liaison.

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
inversement. `hostage` est l'état par défaut après capture, sauf pour un noble
porteur de la dignité de bâtard, qui est placé directement au `dungeon`.

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

Une armée défaite bat en retraite en bloc vers une destination adjacente
déterminée par ordre de priorité décroissant :

1. Case vide **ancrée** au retraité : membre d'un fief du retraité, ou sa
   propre capitale (voir [titres.md](titres.md#contrôle-et-occupation)), même
   si elle a été combattue ce tour.
2. Toute autre case vide non ancrée au retraité, sans château **ancré** (à
   quiconque) et non combattue ce tour ; une case que le retraité contrôlait
   simplement de façon positionnelle, y compris celle qu'il vient de quitter
   ce même tour, relève de cette deuxième priorité comme n'importe quelle
   autre case vide — seul l'ancrage donne la première priorité.
3. Armée amie adjacente non délogée (priorité à la plus petite en troupes), avec
   fusion : la taille de l'hôte augmente de `N − 1` si la retraitante a `N ≥ 2`
   troupes, sinon de `1` (`N = 1` sans perte). Plusieurs armées retraitantes
   fusionnent séquentiellement sur un même hôte ami sans collision destructive.

À égalité dans un bucket, la destination la plus proche d'un château ou village
contrôlé par le propriétaire du retraité l'emporte, puis l'ordre lexicographique
(trigramme croissant). Pour les armées amies, le tri s'effectue par taille
croissante, puis distance à la source contrôlée la plus proche, puis trigramme
croissant.

La case d'origine de l'attaquant est toujours exclue. Un château **ancré** — 
membre d'un fief ou capitale d'un joueur, y compris neutre ou ennemi pour le
retraité — défend contre une retraite et n'est jamais une destination valide ;
un château vide **inerte** (ni fief, ni capitale, ni armée) redevient en
revanche une destination valide de deuxième priorité, comme l'absence de
château. Deux armées qui doivent reculer sur la même case vide sont détruites si
aucune alternative ne subsiste. L'ordre de traitement des armées en retraite suit
le trigramme croissant de leur case d'origine.

## 7. Infrastructures et contrôle

Les infrastructures appartiennent à leur case. Le joueur qui contrôle la case
en bénéficie ; il n'y a pas de propriétaire stocké sur l'infrastructure.

La prise de contrôle reste positionnelle hors fief : une armée qui s'arrête sur
une case en prend le contrôle. Ce contrôle est cependant **éphémère** hors
fief : un territoire ne reste « à quelqu'un » que tant qu'il est
**ancré** — membre d'un fief, capitale du joueur (une exception permanente,
même sans armée), ou actuellement occupé par une armée de ce joueur. Dès
qu'aucune de ces trois conditions n'est plus vraie, le territoire redevient
neutre (sans propriétaire), jusqu'à ce qu'une armée, quelle qu'elle soit,
s'y arrête à nouveau et le reprenne positionnellement. Une révolte `NEUTRAL`
qui s'arrête sur une case ne prend jamais le contrôle : elle l'occupe, ce qui
libère la case si son ancien contrôleur n'y a plus d'armée et ne l'ancre
autrement (voir [titres.md](titres.md#contrôle-et-occupation)).

Dans un fief, le contrôle est **transitif** (titres.md) : un membre
non-capitale reste contrôlé par le propriétaire du fief même lorsqu'une armée
adverse, ou une révolte `NEUTRAL`, s'y arrête (elle l'**occupe** sans le
contrôler) ; seule la prise de la **capitale** du fief transfère le contrôle
de tous ses membres au conquérant en une seule fois.

| Infrastructure | Condition | Effet v1 | Coût |
|---|---|---|---:|
| Moulin | Construction sur case vide contrôlée, adjacente à un château ou village (voisinage requis pour la construction seulement, pas pour l'amélioration) | Inerte tant qu'il n'est pas ancré ou occupé (voir ci-dessus) ; sinon `N` R par niveau, versés à une seule infrastructure : le château adjacent du même contrôleur, sinon le village adjacent du même contrôleur, sinon la case du moulin elle-même (voir [economie.md](economie.md#moulins)) | 3 / 5 / 7 |
| Dépôt de vivres | Aucune condition structurelle | +2 cases de portée de ravitaillement, seulement lorsqu'il est ancré ou occupé (voir ci-dessus) ; inerte sinon | 3 |
| Château | Aucune | +1 défense tant qu'il reste ancré ou occupé (voir ci-dessus), sinon inerte ; verse le revenu territorial (§3) lorsqu'il est contrôlé ; devient une cité (+2 défense au lieu de +1) lorsqu'il est la capitale d'un fief ([titres.md](titres.md#constitution-dun-fief)) | 10 |
| Village | Généré neutre, non constructible | Verse le revenu territorial une fois contrôlé ; produit `village_income` R par tour dans son propre stock tant qu'il reste neutre, qu'il n'ait jamais été tenu ou qu'il vienne d'être abandonné (seule infrastructure qui ne devient jamais inerte, voir ci-dessus) | — |

Un moulin isolé (sans château ni village adjacent du même contrôleur) produit
sur sa propre case, tant qu'il reste ancré ou occupé ; cette case devient
alors elle-même une source de ravitaillement pour son contrôleur. La
production n'y est pas rapatriée automatiquement : un ordre de transfert (`T`)
reste nécessaire pour l'acheminer. La contrainte de voisinage productif ne
s'applique qu'à la construction d'un nouveau moulin : un moulin déjà bâti peut
toujours être amélioré, même isolé, en payant sur son propre stock (voir
[economie.md](economie.md#moulins)). Une construction remplace la structure
existante uniquement lorsque la règle de l'ordre le prévoit : un château
construit sur un village remplace le village et conserve le stock de la case.

## 8. Évolution du document

Le cœur décrit dans les sections 1 à 7 constitue la base v1. Les ajouts futurs
doivent être introduits sous forme de règles complémentaires : politiques,
ordres spéciaux, diplomatie enrichie ou brouillard de guerre. Ils seront
proposés et suivis dans GitHub avant d'être intégrés au document.

## 9. Durée de partie et score final

Une partie est créée avec une durée comprise entre 1 et 50 années, avec une
valeur par défaut de 10 années. Une année conserve exactement quatre tours :
printemps, été, automne et hiver. Le compteur interne `year` commence à 1 ;
l'interface affiche l'année historique `1000 + year`, soit « Année 1001 » au premier
tour joué.

La partie se termine après la résolution du dernier tour de la durée choisie,
ou immédiatement lorsqu'un seul joueur reste en lice. Les scores sont recalculés
après chaque tour et sont visibles par tous les joueurs.

Le score d'un joueur est son score de titres : chaque titre détenu rapporte
1 point, quel que soit son rang (baronnie, comté, duché pour l'instant ;
cardinal, pape, roi et dignité suivront). Territoire, infrastructure, armée
et noble détenus ne rapportent plus rien par eux-mêmes. Voir
[titres.md § Score de titres](titres.md#score-de-titres) pour le détail et
l'état de livraison.

À la fin d'une partie, un unique survivant gagne toujours, même si la durée
vient d'être atteinte. Sinon, le joueur qui possède le score le plus élevé gagne.
Une égalité parfaite de score ne désigne aucun gagnant officiel — y compris
l'égalité 0-0 fréquente avant que les autres sources de titres et la
pondération par mariage ne soient livrées.
