# Titres et victoire

**Milestone lié :** [Titres & Victoire](https://github.com/fogfactory/crown-and-borough/milestone/2)

**Dépend de :** la carte, le contrôle territorial et le ravitaillement du
socle actuel ; les flux de ressources dépendent de
[Économie et prospérité](economie.md) et les titres religieux de
[Religieux](religieux.md).


Les règles de contrôle, de fief et de taxe ci-dessous sont des décisions de
conception actées pour le milestone
[Économie & Fiefs](https://github.com/fogfactory/crown-and-borough/milestone/19).
Chaque section renvoie à l'issue qui la livre. La constitution, la perte et la
vacance d'un fief, ainsi que les points qui en découlent, sont livrées par
[#194](https://github.com/fogfactory/crown-and-borough/issues/194). Aucune
compatibilité avec les parties existantes n'est requise (version majeure).

## Contrôle et occupation

Issue : [#196](https://github.com/fogfactory/crown-and-borough/issues/196)
(contrôle transitif dans un fief), rendu éphémère hors fief par
[#215](https://github.com/fogfactory/crown-and-borough/issues/215).
**Livré.**

La prise de contrôle reste positionnelle : une armée qui s'arrête sur une
case en prend le contrôle. Mais le maintien de ce contrôle diffère selon
l'**ancrage** de la case :

- **ancrée** : membre d'un fief, ou capitale d'un joueur (une exception
  permanente, au même titre qu'une capitale de fief, même sans aucune armée
  dessus). Une case ancrée reste au joueur qui la contrôle indéfiniment, sans
  qu'une armée y stationne.
- **non ancrée** : toute autre case. Son contrôle est **éphémère** : elle ne
  reste « à quelqu'un » que tant qu'une armée de ce joueur y stationne
  actuellement. Dès que ce n'est plus le cas — l'armée est partie, délogée,
  détruite — la case redevient neutre, jusqu'à ce qu'une armée, quelle qu'elle soit,
  s'y arrête à nouveau et la reprenne positionnellement. Une révolte
  `NEUTRAL` qui s'y arrête ne prend jamais le contrôle (voir « occupé »
  ci-dessous) : elle ne fait donc jamais gagner cette libération à son
  ancien contrôleur.

Le contrôle n'est jamais stocké : il est dérivé à la demande (propriétaire du
fief pour un membre de fief, sinon joueur dont la capitale s'y trouve, sinon
propriétaire de l'armée non `NEUTRAL` qui y stationne). Lorsqu'une case
portant une infrastructure perd ainsi son contrôle à la fin d'une passe qui
modifie le contrôle (mise à jour du contrôle territorial d'un tour d'action,
ou hiver après le rapatriement des stocks), un événement `control_changed`
(raison `abandoned`) est rapporté ; une case vide sans intérêt n'en produit
pas, pour ne pas noyer le rapport.

- **contrôlé** : le statut de la case, dérivé par les
  règles d'ancrage ci-dessus. Dans un fief, la case est contrôlée par le
  joueur qui détient le fief, même lorsqu'une armée adverse (ou une révolte
  `NEUTRAL`) s'y arrête. Seule la prise de la capitale du fief transfère le
  fief et donc le contrôle de tous ses territoires (voir ci-dessous).
- **occupé** : une armée est présente sur la case. C'est une information
  dérivée, jamais stockée. Une case a un **contrôleur** (case contrôlée)
  et est **occupée contre son contrôleur** lorsqu'une armée y stationne dont
  le propriétaire diffère de ce contrôleur — une révolte `NEUTRAL` y compris.
  Cette notion s'applique à toute case contrôlée, en fief ou non (une
  capitale de joueur occupée par une révolte en relève tout autant), mais ne
  change jamais son contrôleur : seule la prise de la capitale d'un fief (ou
  la prise positionnelle hors fief) transfère le contrôle.

Une case d'un fief occupée contre son contrôleur :

- ne rapporte pas son revenu à l'occupant : le revenu continue vers la
  capitale du fief, jamais intercepté (voir
  [economie.md](economie.md#revenu-territorial)) ;
- n'est plus une source de ravitaillement ni un dépôt utilisable, ni pour le
  contrôleur ni pour l'occupant (voir
  [economie.md](economie.md#portée-de-ravitaillement)) ;
- rejette tout investissement d'hiver ciblé sur elle, sans prélèvement
  (motif `territory_occupied_by_other_player`) ;
- ne peut ni payer un investissement d'hiver ni recevoir de rapatriement de
  stock de fin d'hiver ;
- garde son bonus défensif (château ou cité) pour l'occupant : aucun
  changement sur ce point, précédent déjà établi pour une révolte sur une
  cité ;
- reste pillable par l'occupant, sans effet sur le fief lorsque ce n'est pas
  la capitale.

La prise de la capitale d'un fief transfère le contrôle de **tous** ses
membres au conquérant en une seule passe, même ceux occupés par une tierce
armée (l'occupation continue, seul le contrôleur change). Hors fief, la prise
de contrôle reste positionnelle, mais son maintien est désormais éphémère
(voir ci-dessus, [#215](https://github.com/fogfactory/crown-and-borough/issues/215)) :
sans ancrage, il ne survit pas au départ de la dernière armée du contrôleur.

La capitale d'un joueur n'est pas un fief implicite, mais elle est un ancrage
permanent au même titre qu'une capitale de fief : hors fief, une capitale
reste contrôlée indéfiniment, avec ou sans armée dessus, tant qu'elle n'est
pas prise par une autre armée qui s'y arrête positionnellement (une révolte
`NEUTRAL` ne la prend jamais).

## Constitution d'un fief

Issue : [#194](https://github.com/fogfactory/crown-and-borough/issues/194).

Un fief se constitue par l'**ordre d'hiver** `T F NNN XXX YYY ZZZ …` : `NNN`
est le noble titulaire, `XXX` la **capitale du fief** (premier territoire du
groupe), et `YYY ZZZ …` le reste du groupe. Le groupe doit :

- compter au moins 3 territoires, contigus par des frontières franchissables
  (BFS restreint au groupe, frontières géométriques non franchissables
  exclues) ;
- être entièrement contrôlé par le joueur ;
- avoir un château sur sa capitale (les autres territoires du groupe peuvent
  porter d'autres châteaux, sans effet particulier) ;
- ne contenir aucun territoire appartenant déjà à un fief ;
- ne compter aucune armée adverse **ni NEUTRAL** (révolte) sur l'une de ses
  cases : l'une ou l'autre bloque la constitution.

Tout rejet est explicite et ne prélève rien : toutes les conditions
ci-dessus, dans cet ordre logique, sont vérifiées avant tout paiement. Le
coût vaut `fief_per_territory` (2 R) par territoire et se paie comme les
autres investissements d'hiver, débité comme un `C C`/`C M` classique (case
ciblée puis réseau de paiement d'hiver habituel). Le titre dépend de la
taille :

| Titre | Territoires | Coût par défaut |
|---|---:|---:|
| Baronnie | 3 | 6 R |
| Comté | 4 | 8 R |
| Marquisat | 5 | 10 R |
| Duché | 6 et plus | 2 R par territoire |

Le titre appartient au **noble** titulaire, qui doit être un noble du
joueur, quel que soit son statut (un otage ou un prisonnier peut être
titulaire ; il ne vote pas tant qu'il est prisonnier, un otage vote). La
constitution ne modifie pas son statut. Il doit aussi respecter
la **ligne de succession** : tous les nobles placés au-dessus de lui (ordre
d'achat, voir [succession.md](succession.md#lignée)) détiennent déjà un fief de
titre équivalent ou supérieur à celui qui est constitué, faute de quoi l'ordre
est rejeté (`succession_rank_blocked`). Un même noble peut
porter **plusieurs titres** simultanément, et un joueur peut détenir plusieurs
fiefs. Lorsqu'un fief est créé, son château capitale devient une **cité** et
apporte `+2` en défense **au total** : ce bonus remplace celui du château
(`castle_defense_bonus`, aujourd'hui 1) plutôt que de s'y ajouter, avec la même
exception d'auto-capture (aucun bonus si tous les attaquants appartiennent au
propriétaire d'une cité vide).

L'attribution d'un fief vacant se fait par l'ordre d'hiver `T A NNN XXX` :
`NNN` est un noble du joueur qui détient le fief, `XXX` sa capitale.
Cet ordre est gratuit (0 R) et soumis à la même règle de ligne de succession
que la constitution (`succession_rank_blocked`).

L'agrandissement d'un fief existant (annexion et changement de rang) est
spécifié dans [politique.md](politique.md#seigneurie-et-annexion).

## Perte et vacance d'un fief

Issue : [#194](https://github.com/fogfactory/crown-and-borough/issues/194),
attribution par défaut livrée par
[#196](https://github.com/fogfactory/crown-and-borough/issues/196) (remplace
la dissolution automatique de #194).

- **Capitale du fief prise** : lorsqu'un autre joueur prend le contrôle de la
  case de la capitale (mise à jour du contrôle territorial, immédiatement
  après la résolution des mouvements), le fief entier passe à ce joueur,
  **vacant** (le titulaire perd son titre), et le contrôle de **tous** les
  autres membres du fief bascule vers ce même joueur dans la même passe
  (contrôle transitif, voir « Contrôle et occupation » ci-dessus) — y compris
  un membre occupé par une tierce armée, qui continue de l'occuper mais sous
  le nouveau contrôleur. Seule la capitale déclenche ce transfert : une autre
  case du fief occupée sans que la capitale ne tombe ne change jamais de
  contrôleur. Une révolte (armée `NEUTRAL`) ne prend jamais le contrôle d'une
  case : elle ne transfère donc jamais un fief, même en délogeant le
  titulaire de sa capitale.
- **Mort du titulaire** (par exemple de la peste) : le fief reste au joueur
  qui le détient mais devient vacant.
- **Capture du titulaire** (otage ou donjon) : aucun effet sur le fief ; le
  noble existe toujours, seul son statut change.
- **Château de la capitale détruit** (pillage, y compris le pillage
  automatique de famine) : le fief est **dissous immédiatement**, quelle que
  soit la saison. Contrairement à une première intuition, il n'y a ni
  suspension du bonus de cité ni délai d'attente : la perte du château qui
  fait la capitale met fin au fief sur-le-champ. C'est la **seule** cause de
  dissolution d'un fief.
- **Fief de la couronne** : un fief dont le titulaire meurt sans héritier ni
  noble vivant chez son joueur passe à la couronne et n'est jamais attribué
  par défaut ; voir [politique.md](politique.md#héritage-dun-fief-sans-héritier).
- **Fief vacant** : il continue d'exister, de produire et de compter son
  point de score jusqu'à son attribution ou la dissolution de sa capitale. Un
  ordre d'hiver (`T A`) l'attribue à un noble du joueur qui le détient, sous réserve de la
  ligne de succession.
  En fin d'hiver, après résolution des ordres d'hiver (y compris une
  éventuelle attribution du même tour) et avant la conservation des stocks,
  tout fief encore vacant est **attribué par défaut** au **premier noble de la
  ligne de succession** du joueur, quel que soit son statut (un otage ou un
  prisonnier n'est pas écarté), avec un avertissement dans le rapport invitant
  le joueur à reprendre la main sur l'attribution au tour suivant. Si le
  joueur n'a plus aucun noble vivant, le fief reste simplement vacant
  (**plus jamais dissous** faute d'attribution) : il continue de produire et
  de compter son point de score jusqu'à ce qu'un noble soit disponible ou que
  sa capitale soit dissoute.

## Taxe seigneuriale

Issue : [#189](https://github.com/fogfactory/crown-and-borough/issues/189).

La carte **Impôts** (kind `seigneurial_tax`, code d'ordre `TX`, jouée depuis le
deck d'ordres spéciaux, voir [ordres-speciaux.md](ordres-speciaux.md)) se joue
par `P TX HHH XXX` : `HHH` est le **noble émetteur**, `XXX` la cible, au
printemps, en été ou en automne. L'émetteur et la cible déterminent la nature
de l'impôt :

- émetteur **seigneur titré** (baron, comte, duc) et cible la capitale d'un de
  ses fiefs, vacant compris : **taxe seigneuriale**. Elle double, pour le tour,
  le revenu territorial de ce fief, village inclus. L'ordre est rejeté si
  l'émetteur ne détient pas le fief ;
- émetteur **roi** et cible la capitale de n'importe quel fief constitué :
  **taxe royale** (voir ci-dessous) ;
- émetteur **évêque, cardinal ou pape** et cible un évêché : **dîme** (voir
  [religieux.md § Dîme](religieux.md#dîme)).

`XXX` est donc, par exception à la règle « `TER` est le village seed d'une
région », la capitale d'un fief ou le village seed d'un évêché. La production
des moulins n'est jamais touchée par une taxe. Deux cartes de taxe jouées sur
le même fief le même tour ne se cumulent pas : la seconde est consommée sans
effet, avec un rapport explicite. Dans tous les cas, jouer un impôt autorise la
Révolte (voir [ordres-speciaux.md](ordres-speciaux.md)) sur **tout territoire
du fief ou de l'évêché ciblé**, pas seulement sa capitale, la saison où il est
joué et la saison suivante, indépendamment de toute famine.

Le **roi** peut taxer n'importe quel fief constitué, mais seulement celui
qui n'est pas déjà taxé par son seigneur ce tour-là (priorité au titulaire
local) ; le supplément est alors détourné vers la capitale du roi au lieu de
la capitale du fief. Cette taxe royale est suivie dans le milestone
[Royauté](politique.md#pouvoirs-royaux).

Le détail du calcul (montant par territoire, avec et sans village) est défini
dans [economie.md](economie.md).

## Points et victoire

**Dépend aussi de :** [Succession](succession.md), pour l'application des
mariages et alliances au score.

L'état actuellement livré (issue
[#251](https://github.com/fogfactory/crown-and-borough/issues/251)) remplace
intégralement l'ancien barème du GDD §9 (territoire, village, moulin,
château, noble, troupe, ressource) et son complément fief ([#194](https://github.com/fogfactory/crown-and-borough/issues/194))
par le seul score de titres ci-dessous : `gdd.md` §9 a été réécrit en
conséquence par anticipation sur le reste de la section « Évolution du
document » du GDD, puisque les titres de roi et
[dignité](dames.md#dignités) n'existent pas encore côté moteur et que la
pondération par mariage (`succession.md § Mariages et alliances`) reste à
livrer. Le score se limite donc pour l'instant au nombre de fiefs détenus
(baronnie, comté, marquisat, duché) ; une partie à durée fixe se termine fréquemment
sur une égalité 0-0 sans vainqueur tant que ces sources manquent, ce qui est
accepté comme transitoire.

### Score de titres

Chaque titre détenu rapporte **1 point, quel qu'il soit** : baronnie, comté, marquisat,
duché, roi, ou [dignité](dames.md#dignités). Aucun titre religieux
(évêque, cardinal, pape) ne rapporte de point de score. Il
n'y a pas de pondération par rang — un baron et un roi comptent chacun pour
1 point de score, quelle que soit la différence de pouvoir en jeu par
ailleurs (voix, revenus, bonus de titre).

Le score d'un joueur est la somme des titres qu'il détient, ajustée par ses
mariages : voir [succession.md § Mariages et alliances](succession.md#mariages-et-alliances)
pour le calcul du poids d'alliance et des catégories tête/secondaire.

Un mariage qui est la tête active des deux familles est une alliance complète :
pas de bonus individuel, les scores s'additionnent pour la victoire commune.
Tout autre mariage est un mariage d'influence : chaque famille gagne en bonus
le nombre de titres de la famille du conjoint (jamais son score bonifié), ce qui
compte pour gagner seul ou via une autre tête. Le bonus est affiché à part
(`alliance`). Les seuils de victoire de `assets/balance.yaml` (50 %/66 % du
territoire) ont été relevés à 75 %/90 % pour tenir compte de ces bonus.

### Seuil de victoire et fin de partie

Une partie se termine immédiatement dès qu'un joueur franchit un seuil de
suprématie sur son score de titres. Deux seuils distincts existent, fixés
dans `assets/balance.yaml` en fonction du nombre de joueurs :

- **seuil solo** : score individuel requis pour une victoire majeure sans
  alliance ;
- **seuil d'alliance** : score combiné (les deux époux d'une tête active,
  voir succession.md) requis pour une victoire majeure commune. Le seuil
  d'alliance est strictement supérieur au seuil solo — une victoire à deux
  doit coûter plus cher que réussir seul, pas seulement cumuler deux scores
  plus faciles à atteindre séparément.

**Un joueur qui a une tête active ne peut jamais gagner seul**, même si son
score individuel atteint ou dépasse le seuil solo : tant qu'une tête est
active, seul le score combiné contre le seuil d'alliance est évalué pour lui.
Un joueur sans tête active (aucune alliance, ou seulement des alliances
secondaires parce que l'autre maison a une alliance mieux valorisée
ailleurs) reste évalué contre le seuil solo.

Si aucun seuil n'est atteint à la durée maximale de la partie (1 à 50 années,
GDD §2), la partie se termine sur le score de titres le plus élevé à cet
instant, selon les mêmes règles de seuil et d'alliance.

Les seuils se calculent dans `assets/balance.yaml` (bloc `victory`) à partir
de la taille du plateau : part des territoires de jeu (8 par joueur) que le
joueur (solo) ou l'alliance doit tenir au travers de fiefs, divisée par la
taille moyenne d'un fief (`reference_fief_size`, 4) et arrondie au supérieur.
Valeurs de départ : la moitié des territoires pour gagner seul, les deux tiers
(66 %) à deux, soit 3 titres (solo seul : une alliance à deux joueurs réunirait tous les joueurs) à 2 joueurs, 5/6 à 3 joueurs, 6/8 à 4 joueurs et 9/11 à 6 joueurs.
Le chiffre est volontairement approximatif, la taille des fiefs variant : un
joueur qui ne tient que des baronnies (3 territoires) atteint le seuil avec
moins de territoires, un joueur de duchés avec davantage. Le seuil d'alliance
reste toujours strictement supérieur au seuil solo. La valeur plus élevée du
titre de roi est différée à l'issue de calibrage.

### Victoire majeure, victoire mineure, échec

- **Victoire majeure** : un joueur sans tête active qui franchit le seuil
  solo est déclaré vainqueur majeur seul. Un joueur avec tête active ne peut
  être vainqueur majeur qu'avec son conjoint, et seulement si leur score
  combiné franchit le seuil d'alliance ; les deux époux sont alors vainqueurs
  à égalité, sans hiérarchie entre eux.
- **Victoire mineure** : parmi tous les joueurs reliés au vainqueur majeur
  par une chaîne de mariages d'alliance (tête ou secondaire, y compris
  transitive à travers plusieurs maisons), seul celui dont le lien a le
  poids d'alliance le plus élevé obtient une victoire mineure. Les autres
  membres de la chaîne n'obtiennent rien de cette victoire.
- **Échec** : tout joueur restant, éliminé ou non, qui n'obtient ni victoire
  majeure ni victoire mineure.

Lorsque plusieurs joueurs ou alliances remplissent la condition de victoire
au même hiver, ils sont départagés dans cet ordre : une victoire solo l'emporte
sur une victoire d'alliance ; puis le score le plus élevé ; puis celui (ou
l'alliance) qui détient le titre de roi ; puis celui qui cumule le plus de
territoires (les deux époux additionnés pour une alliance). Si l'égalité persiste
sur tous ces critères, les joueurs ou alliances restants sont vainqueurs
majeurs ex æquo. Une partie où personne ne détient de titre (0-0) n'a aucun
vainqueur.

### Titres de courtoisie

Le conjoint d'un titulaire de fief (baron, comte, duc) ou de couronne (roi)
porte un titre de courtoisie assorti — baronne, comtesse, duchesse, reine —
quel que soit son sexe et quelle que soit la catégorie du mariage (tête
ou secondaire). Ce titre est **strictement d'affichage** : il n'entre
dans aucun calcul de score, de poids d'alliance, de vote ou de rang de
succession, et ne confère aucun accès aux titres réservés aux hommes
(évêque, cardinal, pape, roi lui-même). Il suit le titulaire réel du fief ou
de la couronne et change ou disparaît avec lui (remariage, mort, perte du
titre).

### Lisibilité et simulateur

Le calcul combine poids d'alliance, tête active, catégorie de mariage et deux
seuils distincts : il n'est pas raisonnable de demander aux joueurs de le
recalculer de tête. Le front doit exposer un simulateur — à la manière de
`POST /api/games/{id}/orders/preview` pour les ordres — qui projette, à la
demande, le score de titres et le statut de victoire (majeure/mineure/échec)
d'un joueur pour un état hypothétique (avant de conclure un mariage, après un
Claim, etc.), sans engager l'action : `POST /api/games/{id}/victory/simulate`
(voir `architecture.md`). Le simulateur est un mode de la vue « Lignée et alliances » : le bouton « Simuler » permet de cliquer un noble (tuer, marier à…, prétendre à…) ou un lien de mariage (rompre), avec annuler/rétablir/réinitialiser. L'arbre, les liens, les scores et le panneau d'état de victoire de chaque maison (score avant/après, seuil, mode, statut) reflètent l'état hypothétique, jusqu'à réinitialisation.
