# Économie et prospérité

**Milestone lié :** [Économie & Fiefs](https://github.com/fogfactory/crown-and-borough/milestone/19)
(le transfert de ressources a été livré dans
[Économie & Prospérité](https://github.com/fogfactory/crown-and-borough/milestone/8))

**Dépend de :** [Ravitaillement](ravitaillement.md) et
[Titres](titres.md) pour le modèle de contrôle et les fiefs ; la dîme
religieuse dépend de [Religieux](religieux.md).

Les règles ci-dessous sont des décisions de conception actées pour la refonte,
pas encore implémentées sauf mention contraire. Chaque section renvoie à
l'issue qui la livre ; les points « à trancher » sont tranchés au début de
cette issue. Aucune compatibilité avec les parties existantes n'est requise
(version majeure).

## Vocabulaire

- **Contrôlé** : le statut porté par la case (`OwnerID`). Hors fief, la prise
  est positionnelle (dernier joueur dont une armée s'est arrêtée sur la case),
  mais son maintien est **éphémère** : la case ne reste au joueur que tant
  qu'elle est ancrée (fief, capitale du joueur) ou qu'une de ses armées y
  stationne encore ([#215](https://github.com/fogfactory/crown-and-borough/issues/215)).
  Dans un fief, il est transitif (joueur qui détient le fief). Voir
  [titres.md](titres.md#contrôle-et-occupation).
- **Occupé** : une armée est présente sur la case. Information dérivée, jamais
  stockée.

## Objectif de la refonte

L'économie v1 incite à entourer un unique château de moulins au niveau
maximal : la production dépend uniquement de l'infrastructure bâtie, jamais du
territoire lui-même. La refonte inverse cette incitation en faisant du
territoire contrôlé la source de revenu principale, et en réduisant la
capacité d'une case isolée à nourrir une grosse armée sur place.

## Rations de terrain

Issue : [#191](https://github.com/fogfactory/crown-and-borough/issues/191).

**Appliqué.** Cette table remplace celle de `gdd.md` §3 et §5. Objectif : réduire
le plafond d'armée soutenable sans logistique, et faire du siège un vrai
levier d'affamement plutôt qu'un détail négligeable.

| Terrain | Rations v1 | Rations retenues |
|---|---:|---:|
| Plaine | 3 | 2 |
| Forêt | 2 | 1 |
| Colline | 2 | 1 |
| Montagne | 1 | 0 |
| Marécage | 1 | 1 |
| Bonus château/village | +2 | supprimé |

Un château ou un village ne relève plus la production locale de sa case : un
siège en montagne (0 ration locale, aucun bonus) affame une armée qui ne
dispose d'aucune autre source de ravitaillement.

### Cartes météo et récolte

Les cartes se répartissent désormais en deux familles, sans effet croisé :

| Carte | Effet économique dans sa région |
|---|---|
| Mauvais temps | Moulins à l'arrêt (0 R), en plus du blocage des mouvements. |
| Beau temps | Annule le mauvais temps ; sinon, production des moulins doublée. |
| Mauvaise récolte | Rations de terrain et revenu territorial supprimés. |
| Récolte abondante | Annule la mauvaise récolte ; sinon, rations de terrain et revenu territorial doublés. |

Les bonus additifs de la v1 (`bonus_mill_production`, `bonus_army_ration`)
sont retirés de la balance. La récolte s'applique au revenu territorial
(territoires et villages, voir ci-dessous) exactement de la même façon,
dans la région du territoire qui produit ce revenu.

## Position de départ

Issue : [#203](https://github.com/fogfactory/crown-and-borough/issues/203).

**Appliqué.** Cette section détaille et complète `gdd.md` §2. Objectif :
garantir que chaque position de départ reste viable sous la table de rations
ci-dessus, et donner à chaque joueur de quoi étendre son territoire dès le
premier tour.

Le territoire de départ n'est jamais une montagne et compte toujours au moins
deux voisins franchissables eux-mêmes non montagneux ; son terrain naturel
(plaine, forêt, colline ou marécage) n'est en revanche jamais réécrit. Cette
seule contrainte de filtrage suffit à garantir la viabilité économique du
premier tour : une garnison de deux troupes (`starting_troops`) coûte 2 R par
tour, couverts par la ration locale (au moins 1 R hors plaine) complétée par
le revenu territorial de la capitale et son stock de départ
(`starting_resources`), quel que soit le terrain retenu.

En plus de sa garnison, chaque joueur reçoit `starting_outposts` armées d'une
troupe chacune, placées dès la création de la partie sur autant de
territoires voisins non montagneux distincts de sa capitale — retenus par
ration de terrain décroissante, puis par trigramme croissant en cas d'égalité.
Une armée d'une troupe coûte 1 R par tour, toujours couvert par la seule
ration locale d'un terrain non montagneux : un avant-poste ne dépend donc
d'aucun stock ni d'aucun ravitaillement pour se nourrir. Un avant-poste ne
porte aucune infrastructure ni aucune ressource propre à la création, mais
étant adjacent à la capitale (un château), il satisfait la condition de
voisinage productif requise pour y construire un moulin dès le premier hiver.

## Revenu territorial

Issue : [#192](https://github.com/fogfactory/crown-and-borough/issues/192)
(hors fief), revenu dirigé vers la capitale d'un fief par
[#196](https://github.com/fogfactory/crown-and-borough/issues/196).
**Appliqué.**

- À chaque tour d'action (printemps, été, automne ; jamais en hiver), chaque
  territoire contrôlé rapporte `territory_income` R (1), plus
  `village_income` R (1) s'il porte un village.
- Le revenu est crédité en **fin de tour**, avec le reste du ravitaillement
  ([#208](https://github.com/fogfactory/crown-and-borough/issues/208)), sur le
  contrôle territorial définitif du tour : un territoire capturé pendant le
  tour verse son revenu à son nouveau contrôleur, pas à celui du début de
  tour. Il n'est pas acheminé par le réseau et ne peut pas être intercepté.
- Hors fief, il est versé au stock de la **capitale du joueur**. Sans
  capitale (ou si elle vient de tomber), le revenu de **chaque territoire**
  est calculé indépendamment : il va au château contrôlé le plus proche
  (distance en frontières franchissables, départage par trigramme), sinon au
  village contrôlé le plus proche, sinon il est perdu. Deux territoires du
  même joueur peuvent donc alimenter des destinations différentes le même
  tour tant qu'aucune capitale n'existe.
- Dans un fief, il est versé au stock de la **capitale du fief** — y compris
  lorsque le territoire producteur ou la capitale du fief elle-même est
  occupée par une armée adverse, le revenu n'étant jamais intercepté par
  l'occupant. Le regroupement du rapport de revenu se fait par (destination,
  fief) : un fief dont la capitale coïncide avec la capitale du joueur
  produit donc une ligne distincte de celle du reste du domaine, même s'ils
  partagent la même destination. Lorsque la capitale d'un fief taxé
  ([#189](https://github.com/fogfactory/crown-and-borough/issues/189)) est
  capturée pendant le tour même où la taxe seigneuriale s'applique, la taxe
  est annulée pour ce tour de transition : personne ne touche le doublement,
  ni l'ancien ni le nouveau détenteur du fief.
- La production de base des châteaux et villages contrôlés
  (`base_production`) est supprimée : le revenu territorial la remplace. Le
  revenu territorial et la production des moulins restent deux flux
  distincts ; la règle de destination unique d'un moulin est précisée par
  [#195](https://github.com/fogfactory/crown-and-borough/issues/195)
  ci-dessous.
- Un village **neutre** continue de produire `village_income` R par tour dans
  son propre stock, récupéré à sa capture, qu'il n'ait jamais été tenu ou qu'il
  vienne d'être abandonné faute d'ancrage
  ([#215](https://github.com/fogfactory/crown-and-borough/issues/215)) : c'est
  la seule infrastructure qui garde une production neutre après abandon,
  contrairement à un moulin (voir « Moulins » ci-dessous).

## Flux de la ressource R

| Source | Bénéficiaire par défaut | Avec taxe |
|---|---|---|
| Territoire d'un fief, sans village | 1 R → capitale du fief | Seigneur : 2 R → capitale du fief. Roi (si le seigneur ne taxe pas ce tour) : +1 R → capitale du roi, 1 R de base reste au fief |
| Territoire d'un fief, avec village | 2 R → capitale du fief (1 territoire + 1 village) | Seigneur : 4 R → capitale du fief. Roi (si non taxé localement) : +2 R → capitale du roi, 2 R de base restent au fief |
| Territoire contrôlé hors fief, sans village | 1 R → capitale du joueur | — (pas de taxe hors fief) |
| Territoire contrôlé hors fief, avec village | 2 R → capitale du joueur | — |
| Village neutre | 1 R → son propre stock | — |
| Moulin (niveau `N`) | `N` R → château adjacent du même contrôleur, sinon village adjacent du même contrôleur, sinon reste sur le moulin | Dîme religieuse jouée sur l'évêché : `N` R → capitale du joueur qui a joué la dîme, au lieu du village/château adjacent |

La taxe du seigneur ([#189](https://github.com/fogfactory/crown-and-borough/issues/189))
est appliquée par le moteur ; la taxe royale et la dîme restent suivies dans
les milestones Politique royale et Religieux.

La taxe seigneuriale double toujours exactement le revenu de territoire,
village inclus ; elle ne touche jamais la production des moulins. La dîme
(voir [religieux.md](religieux.md)) fait l'inverse : elle ne touche que les
moulins d'un évêché, jamais le revenu de territoire. Les deux mécaniques
portent donc sur des flux disjoints et ne peuvent pas entrer en conflit sur le
même territoire le même tour.

## Moulins

Issue : [#195](https://github.com/fogfactory/crown-and-borough/issues/195).
**Appliqué.**

La production des moulins suit le même timing de fin de tour que le revenu
territorial et la famine ([#208](https://github.com/fogfactory/crown-and-borough/issues/208)) :
elle se calcule sur le contrôle et l'occupation définitifs du tour, avec les
mêmes calamités météo (mauvais temps, Beau temps). L'algorithme de production
et de destination d'un moulin lui-même n'en est pas modifié. Un moulin détruit
par un ordre de pillage (`T P`) pendant la résolution des mouvements ne
produit rien ce tour-là : la destruction précède le calcul de fin de tour.

Un moulin outre-fief et hors capitale, sans armée dessus, est **inerte**
([#215](https://github.com/fogfactory/crown-and-borough/issues/215)) : il ne
produit rien du tout, ni pour lui-même ni pour un voisin, tant qu'il reste
dans cet état — qu'il n'ait jamais été tenu ou qu'il vienne d'être abandonné.
Un moulin membre d'un fief, sur la capitale d'un joueur, ou actuellement tenu
par une armée, reste actif et produit normalement, décrit ci-dessous.

Un moulin actif de niveau `N` produit `N` R (`2N` sous le Beau temps, 0 sous le
mauvais temps) et verse sa production à **une seule** infrastructure, dans cet
ordre de préférence :

1. le château adjacent **contrôlé par le même joueur que la case du moulin** ;
2. sinon le village adjacent, même exigence de contrôle ;
3. sinon la case du moulin elle-même.

Un château ou un village adjacent contrôlé par un autre joueur est ignoré : on
passe au candidat suivant plutôt que de lui verser la production. Le
contrôleur « neutre » (case sans propriétaire) est un contrôleur comme un
autre : un moulin neutre ne verse donc jamais à un château ou un village d'un
joueur, seulement à un village neutre adjacent (il n'existe pas de « château
neutre »), sinon sur sa propre case. Entre plusieurs candidats du même type et
du même contrôleur, le départage se fait par trigramme, comme pour le revenu
territorial (#192). Un moulin ne compte donc plus pour chaque château ou
village adjacent : c'est la correction du double comptage qui existait avant
#195.

Une production restée sur un moulin isolé (sur sa propre case) n'est pas
automatiquement acheminée : elle nécessite un ordre de transfert (`T`, voir
ci-dessous) porté par une armée. Elle rend cependant la case du moulin
elle-même éligible comme source de ravitaillement pour son contrôleur (au même
titre qu'un château ou un village). En hiver, le stock d'un moulin est
conservé à `ceil(stock / 2)`, comme celui d'un château ou d'un village, et
n'est **pas** rapatrié vers la capitale.

### Construction contre amélioration

La contrainte de voisinage productif (`mill_requires_productive_neighbor` :
un moulin ne peut être bâti que sur une case contrôlée elle-même porteuse d'un
château ou d'un village, ou adjacente à une telle case) ne s'applique qu'à la
**construction** initiale d'un moulin. Elle ne s'applique pas à son
**amélioration** : un moulin isolé (sans château ni village adjacent du même
contrôleur) peut toujours être amélioré, en payant sur son propre stock.

### Amélioration d'un moulin

Le paiement d'une amélioration de moulin (`C M` sur un moulin existant) puise
dans cet ordre :

1. le stock présent sur le moulin lui-même ;
2. le stock de l'infrastructure qui recevrait sa production (château en
   priorité, sinon village, selon les mêmes règles de contrôle et de
   départage que ci-dessus) ;
3. le paiement d'hiver habituel (réseau des châteaux et villages contrôlés).

Le stock d'un moulin fait ainsi exception à la règle selon laquelle seuls les
châteaux et villages paient les investissements d'hiver.

## Village fortifié

Issue : [#193](https://github.com/fogfactory/crown-and-borough/issues/193).

`C C XXX` sur un village contrôlé le **fortifie** pour le coût d'un château
(10 R). Le village fortifié est un village porteur d'un simple indicateur, pas
un nouveau type d'infrastructure : il conserve son stock, sa production et son
bonus de revenu comme n'importe quel village, et gagne en plus le bonus
défensif d'un château (`castle_defense_bonus`), avec la même exception
d'auto-capture. Il compte pour 2 points de score comme tout village (GDD §9,
pas les 5 points d'un château), ne peut pas être désigné capitale par `E C`
(qui exige un château) et ne peut pas être la capitale d'un fief (`T F`, qui
exige également un château). Un `C C` sur un village déjà fortifié est rejeté
sans prélèvement.

## Densité des villages

Issue : [#202](https://github.com/fogfactory/crown-and-borough/issues/202).

La carte porte `2 x N + 1` villages plutôt que `N + 1` : chaque territoire de
départ reçoit son propre village dédié, en plus des `N + 1` chefs-lieux qui
seedent le découpage régional (voir [`gdd.md`](gdd.md#3-carte-terrains-et-villages)
et [`cartographie.md`](cartographie.md#villages-neutres-du-socle) pour les
contraintes de placement). L'objectif est de rapprocher une source de revenu
territorial et de ravitaillement de chaque joueur dès le début de partie, sans
changer la taille de la carte : le nombre de territoires par territoire de
départ (`TerritoriesPerPlayer = 8`) et par chef-lieu (`TerritoriesPerSeat =
4`) reste inchangé, seule la densité de villages dans l'enveloppe déjà réservée
augmente.

Un village dédié et un chef-lieu sont des infrastructures identiques en jeu :
même revenu territorial, même production neutre tant qu'ils ne sont pas tenus,
même possibilité de fortification (`C C`, voir ci-dessus). Aucun des deux ne
peut être désigné capitale ni devenir la capitale d'un fief. La seule
différence est fonctionnelle et hors économie : le chef-lieu identifie la seed
d'une région dans `regions[].seed` ; le village dédié n'en identifie aucune.

Les constantes de placement (distance exacte de deux étapes entre un
territoire de départ et son village dédié, séparation minimale de trois étapes
avec tout autre territoire de départ, séparation minimale de deux étapes entre
villages) sont des constantes de génération de carte
(`internal/engine/mapgen`), pas des valeurs de `assets/balance.yaml` : elles
façonnent la topologie de la carte, pas l'équilibrage économique.

## Portée de ravitaillement

Un château, un village ou un dépôt de vivres contribue au ravitaillement
(ancre, portée de base, bonus de portée) du joueur qui **contrôle** sa case.
Une conquête récente hors fief est contrôlée dès qu'une armée s'y arrête et
garde donc immédiatement sa valeur logistique. Un château ou un dépôt hors
fief et hors capitale perd cette valeur dès que sa case n'est plus contrôlée
par personne, ce qui arrive dès que la dernière armée qui l'ancrait
positionnellement en repart
([#215](https://github.com/fogfactory/crown-and-borough/issues/215)) ; un
dépôt inerte de la sorte n'étend alors la portée de personne, exactement
comme un dépôt occupé contre son contrôleur ci-dessous.

Une case contrôlée mais **occupée contre son contrôleur** (titres.md,
[#196](https://github.com/fogfactory/crown-and-borough/issues/196)) n'est
plus une source de ravitaillement, ni pour le contrôleur ni pour l'occupant ;
un dépôt qui s'y trouve n'étend la portée de personne. L'occupant peut
toujours piller son infrastructure ; le contrôleur, lui, ne peut plus
consommer le stock local tant que la case reste occupée. Les investissements
d'hiver ciblés sur une case occupée sont rejetés sans prélèvement
(`territory_occupied_by_other_player`), et une telle case ne peut ni payer un
investissement ni recevoir le rapatriement de stock de fin d'hiver (voir
« Hiver » ci-dessous).

## Transfert de ressources

Deux ordres dédiés permettent de transférer des ressources entre joueurs.

### Tours d'action

`XXX T YYY N` transfère `N` ressources depuis le stock de la case occupée par
l'armée émettrice vers `YYY`.

- `XXX` est la position de l'armée au tour d'exécution ; la chaîne peut contenir
  plusieurs transferts comme n'importe quels autres ordres.
- `YYY` doit être un château, un village ou la case d'une armée contrôlée par un
  autre joueur vivant. Un dépôt sans armée n'est pas une destination valide.
  Lorsque `YYY` porte une armée, celle-ci doit **contrôler** sa propre case :
  une armée qui ne fait que l'occuper (par exemple un membre de fief non
  contrôlé par son propriétaire) ne peut pas recevoir le transfert
  (`transfer_target_occupied`).
- Le stock source ne nécessite pas de château ni de village. Toute case
  contrôlée peut conserver un cache pendant un tour d'action, sauf si elle
  est occupée contre son contrôleur (`transfer_source_not_controlled`) : ni
  l'émetteur ni le réseau ne peuvent alors s'appuyer dessus.
- Le transfert suit le réseau de ravitaillement du donneur : portée de base de
  trois cases, bonus des dépôts contrôlés et blocage par les armées adverses.
  L'armée adverse du destinataire bloque également lorsqu'elle se trouve sur
  une case intermédiaire ; elle est autorisée sur la case destination.
- Le réseau peut être estimé à la soumission et dans l'overlay, puis est
  recalculé sur la position de début du tour lorsque l'ordre est exécuté. Une
  route bloquée rend l'ordre invalide.
- Une armée affamée ([#208](https://github.com/fogfactory/crown-and-borough/issues/208))
  ne peut pas émettre de transfert. Elle ne transporte pas plus que sa
  consommation brute : `2^(N - 1)` ressources pour une armée
  de `N` troupes, sans déduire les rations locales.
- L'ordre est traité pendant la résolution des ordres d'armée, avant le
  ravitaillement de fin de tour. L'armée qui le porte ne réalise aucun autre
  ordre pendant ce tour.
- Un manque de ressources n'a aucun effet et ne casse pas une chaîne `single` ;
  elle avance alors vers l'ordre suivant. En mode `loop`, elle retente le
  transfert.
- En mode `loop`, si le stock restant est inférieur au montant demandé, tout le
  stock restant est transféré et l'ordre est terminé. Cette dernière livraison
  peut donc être partielle.

Exemple :

```text
XXX A YYY
YYY T AAA 1
YYY T BBB 1
(YYY T CCC 2)
YYY A ZZZ
```

Les stocks sont consommés comme sources de ravitaillement. Une armée mange en
priorité le stock de sa case, puis le réseau de sources contrôlées.

### Hiver

`G XXX YYY N` est un ordre de gestion hivernal. Il ne dépend d'aucune armée,
n'est pas interceptable et n'utilise pas le plafond de transport.

- `XXX` doit être un château ou un village contrôlé par le donneur, non
  occupé contre son contrôleur ; le débit suit les règles habituelles des
  paiements d'hiver, qui excluent elles aussi toute colonie ou tout moulin
  occupé (`territory_occupied_by_other_player`).
- `YYY` doit être un château ou un village contrôlé par un autre joueur vivant.
  Il n'est donc pas nécessaire que la destination appartienne au donneur : le
  transfert peut alimenter directement le château ou le village du joueur
  destinataire.
- Les armées, dépôts et caches nus ne peuvent pas être la destination d'un
  transfert hivernal.
- Le transfert est appliqué avant la conservation et le rapatriement des stocks.
- Les stocks d'un château ou d'un village sont conservés à `ceil(stock / 2)`.
- Les stocks d'un dépôt de ravitaillement sont conservés intégralement.
- Tout stock placé sur une autre case est perdu à l'hiver. Les stocks hors
  château et village ne peuvent pas payer les investissements hivernaux.
- Le rapatriement de fin d'hiver ne concerne pas une colonie occupée contre
  son contrôleur : son stock y reste et suit la conservation normale
  (`ceil(stock / 2)`) plutôt que d'être remonté vers la capitale.

## Prospérité

Issue : [#197](https://github.com/fogfactory/crown-and-borough/issues/197).

Objectif : faire apparaître de nouveaux villages sans action directe d'un
joueur, tout en laissant sa gestion (contrôle, sièges, fiefs) influencer où
et quand ça arrive. Le déclencheur reste un exode plutôt qu'un surplus : un
lieu-dit éprouvé voit sa population fuir et fonder un village ailleurs. La
règle est évaluée en hiver, après la conservation des stocks.

Règle retenue pour une première version — **exode élargi** :

1. le lieu-dit fuit vers la case libre (sans infrastructure) la plus proche
   **non adjacente à un village ou un château existant**, en respectant
   l'ordre de priorité suivant : dans un fief du joueur qui contrôle le
   lieu-dit d'origine, sinon contrôlée par ce joueur, sinon n'importe quelle
   case libre restante, y compris neutre ou contrôlée par un autre joueur ;
2. si aucune case ne satisfait la contrainte d'adjacence à aucun niveau de
   priorité, une dégradation en cascade s'applique : un dépôt de vivres est
   amélioré en village, à défaut un moulin est amélioré en village, sinon rien
   ne se passe.

La contrainte de non-adjacence évite qu'un nouveau village apparaisse collé à
une infrastructure existante ; elle s'applique à tous les niveaux de priorité
de l'étape 1, pas seulement au dernier.

Le village fondé appartient au **contrôleur de la case** d'arrivée : le
détenteur du fief, le contrôleur positionnel, ou personne si la case est
neutre.

> À trancher dans #197, par des parties de test :
>
> - le déclencheur : pertes cumulées de l'année par la guerre (pillage, stock
>   consommé par la famine ou le siège, calamité) au-delà de `N`
>   (recommandé), ou perte de stock à la conservation d'hiver au-delà de `N` ;
> - la valeur de `N`, dans la balance, calibrée avec la nouvelle table de
>   rations (l'ancien seuil `N = 5` supposait une plaine à 3 et le bonus
>   château/village) ;
> - le sort du lieu-dit d'origine (conservé ou dégradé) ;
> - le départage entre cases à égalité de distance (recommandé : trigramme).

Piste complémentaire, non retenue pour une première version mais à garder en
réserve si la carte reste trop statique en pratique : une croissance passive
et indépendante des joueurs, où un territoire neutre inoccupé et sans conflit
à proximité depuis plusieurs tours a une chance déterministe (seedée comme le
deck) de fonder un village chaque année. Contrairement à l'exode, cette piste
ne dépend d'aucune décision de joueur.
