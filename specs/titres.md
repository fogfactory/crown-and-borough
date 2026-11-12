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

La prise de contrôle est positionnelle : une armée qui s'arrête sur une
case en prend le contrôle (pour la capitale d'un fief, voir « Conquête d'un
fief par un noble »). Le maintien de ce contrôle diffère selon
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
  `NEUTRAL`) s'y arrête. Seule la conquête de la capitale du fief transfère le
  fief et donc le contrôle de tous ses territoires (voir ci-dessous).
- **occupé** : une armée est présente sur la case. C'est une information
  dérivée, jamais stockée. Une case a un **contrôleur** (case contrôlée)
  et est **occupée contre son contrôleur** lorsqu'une armée y stationne dont
  le propriétaire diffère de ce contrôleur — une révolte `NEUTRAL` y compris.
  Cette notion s'applique à toute case contrôlée, en fief ou non (une
  capitale de joueur occupée par une révolte en relève tout autant), mais ne
  change jamais son contrôleur : seule la conquête de la capitale d'un fief (ou
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

La conquête de la capitale d'un fief (par une armée portant un noble libre,
voir « Conquête d'un fief par un noble ») transfère le contrôle de **tous** ses
membres au conquérant en une seule passe, même ceux occupés par une tierce
armée (l'occupation continue, seul le contrôleur change). Hors fief, la prise
de contrôle est positionnelle et son maintien éphémère (voir ci-dessus, [#215](https://github.com/fogfactory/crown-and-borough/issues/215)) :
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
la **ligne de succession**, lue séparément pour chaque sexe (voir
[succession.md](succession.md#lignée)) : tous les nobles de son sexe placés
au-dessus de lui (ordre d'achat) détiennent déjà un fief de titre équivalent ou
supérieur à celui qui est constitué, faute de quoi l'ordre est rejeté
(`succession_rank_blocked`). Un évêque, un cardinal ou le pape compte comme
déjà titré plus haut et ne bloque personne ; un bâtard ne reçoit un titre que
s'il est le dernier de sa lignée. Un même noble peut
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
  case de la capitale avec une armée portant un noble libre (mise à jour du
  contrôle territorial, immédiatement après la résolution des mouvements, voir
  « Conquête d'un fief par un noble »), le fief entier passe à ce joueur. Le
  noble conquérant devient titulaire usurpateur et l'ancien titulaire exilé ;
  un fief déjà vacant reste vacant. Le contrôle de **tous** les
  autres membres du fief bascule vers ce même joueur dans la même passe
  (contrôle transitif, voir « Contrôle et occupation » ci-dessus) — y compris
  un membre occupé par une tierce armée, qui continue de l'occuper mais sous
  le nouveau contrôleur. Seule la capitale déclenche ce transfert : une autre
  case du fief occupée sans que la capitale ne tombe ne change jamais de
  contrôleur. Une révolte (armée `NEUTRAL`) ne prend jamais le contrôle d'une
  case : elle ne transfère donc jamais un fief, même en délogeant le
  titulaire de sa capitale. Une armée sans noble libre ne transfère pas non
  plus le fief : elle occupe la capitale.
- **Mort du titulaire** (par exemple de la peste) : le fief reste au joueur
  qui le détient mais devient vacant, sauf Claim ou usurpation (voir
  « Conquête d'un fief par un noble » et
  [succession.md](succession.md#prétentions-claims)).
- **Capture du titulaire** (otage ou donjon) : aucun effet sur le fief ; le
  noble existe toujours, seul son statut change.
- **Château de la capitale pillé** (pillage, y compris le pillage
  automatique de famine) : le château est mis en ruine mais reste la capitale
  du fief, qui n'est pas dissous ; la capitale du joueur est conservée. Le
  château en ruine n'apporte aucune défense (pas de bonus de cité), reste
  une source de ravitaillement et un point d'ancrage pour les retraites, et
  peut être reconstruit par `C C`. Un château en ruine ne peut pas devenir la
  capitale d'un nouveau fief (`T F`). Le fief ne se dissout que si le château
  de sa capitale disparaît ; seul `C C` peut être construit sur sa ruine.
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

## Conquête d'un fief par un noble

Issue : [#610](https://github.com/fogfactory/crown-and-borough/issues/610).
**Livré.**

**Conquête.** La capitale d'un fief n'est conquise que par une armée de joueur
portant sur la case au moins un **noble libre** de son propriétaire (un otage
ou un prisonnier porté par l'armée ne compte pas), évaluée à la mise à jour
du contrôle territorial. Le fief passe tout entier, avec ses territoires, au
conquérant (contrôle transitif, voir « Contrôle et occupation »). Si le fief a un
titulaire, le premier noble libre de l'armée, dans l'ordre de la ligne de
succession du joueur, devient titulaire avec le statut **usurpateur** et
l'ancien titulaire reçoit le statut **exilé**. Il faut un titulaire légitime
pour qu'il y ait usurpation : un fief **vacant** conquis passe au conquérant
sans usurpateur ni exilé et reste vacant (`T A` permet de l'attribuer). Cette
condition de noble libre vaut aussi pour un fief vacant : sans noble libre,
l'armée occupe seulement. Un noble seul, sans armée, ne conquiert rien. Les armées sans
noble occupent la capitale avec les conséquences déjà décrites pour un membre
de fief occupé (revenu du fief inchangé, case ni source de ravitaillement ni
dépôt, investissements refusés). La capitale d'un joueur hors fief reste
positionnelle.

**Exilé.** Un titre exilé n'a aucun effet de jeu (ni rang, ni vote, ni
bonus) : c'est un rappel de rôle, affiché comme tel dans les rapports et dans
l'interface (arbre de lignée, panneau du territoire), pas dans les noms
composés par le serveur. Un fief n'a au plus qu'un exilé : le titulaire légitime
d'origine.

**Restauration par la reprise de la capitale.** Lorsque le propriétaire du
noble exilé reprend la capitale du fief avec une armée de lui, **sans condition
de noble**, le fief lui revient entièrement, l'exilé redevient titulaire (plus
d'usurpateur, plus d'exilé), sans condition de rang de succession. Une armée du
propriétaire de l'exilé déjà présente sur la capitale restaure le fief au
contrôle territorial suivant. Si l'exilé et l'usurpateur appartiennent au même
joueur, la reprise ne change rien : seule la mort de l'un ou `T D` résout la
situation.

**Durée de l'usurpation.** Une usurpation ne dure que tant que l'usurpateur et
l'exilé vivent. Si l'usurpateur meurt, l'exilé retrouve le titre et le fief
revient à son propriétaire. Si l'exilé meurt, l'usurpateur devient le titulaire
officiel (plus de statut usurpateur), sauf si un Claim vise l'exilé (voir
ci-dessous). Si les deux sont morts dans le même règlement, même avec un Claim vivant sur
l'exilé, le fief est vacant, reste à son propriétaire du moment (celui de
l'usurpateur) et suit les règles de « Perte et vacance d'un fief » ; le Claim
n'est pas honoré. Cette règle vaut pour des morts réglées ensemble ; des morts
réglées à des moments différents du même tour s'appliquent l'une après l'autre
(l'usurpateur meurt : l'exilé retrouve le fief ; l'exilé meurt ensuite : le fief
suit la règle d'un titulaire ordinaire, Claim compris).

**Claims sur un fief usurpé.** Un Claim ne permet pas de récupérer un titre
usurpé : un Claim sur l'usurpateur est sans effet sur ce fief, dont la mort
rend le titre à l'exilé. Un Claim sur le titre **exilé** s'applique : à la mort
de l'exilé, l'héritier du premier Claim vivant (classement habituel) devient le
nouvel exilé, et l'usurpateur reste usurpateur. Il n'y a pas de Claim vivant
sur l'exilé : l'usurpateur devient officiel. Le nouvel exilé peut appartenir à
n'importe quel joueur, y compris celui de l'usurpateur ; sa restauration suit
les règles de « Restauration par la reprise de la capitale » et de « Durée de
l'usurpation », le fief revenant au propriétaire du noble exilé. Les Claims qui visent
ces nobles restent valables pour leurs autres fiefs.

**Usurpé puis conquis.** Si un troisième joueur conquiert un fief déjà
usurpé, l'usurpateur déchu perd tout (il ne devient pas exilé) ; l'exilé
d'origine reste exilé.

**Claim.** À la mort du titulaire, si la capitale porte une armée avec un
noble libre, ce noble devient titulaire usurpateur et le fief passe à son
propriétaire, l'héritier du Claim recevant le titre exilé. Une garnison de
l'ancien propriétaire avec noble annule donc le Claim. Sans armée, avec une
armée sans noble, ou avec une armée de la maison de l'héritier, l'héritier prend
le titre officiellement (ni usurpation, ni exil).

**Renonciation.** Deux ordres d'hiver gratuits permettent de renoncer :
- à un Claim, par `T C HHH` : le joueur qui le détient éteint la prétention de
  son héritier `HHH` (issue « renoncée »), la carte de prétention retourne à la
  défausse et l'héritier peut réclamer à nouveau avec une autre carte, y
  compris par un `C N` plus bas sur la même feuille ;
- à un titre contesté, exilé ou usurpé, par un même ordre `T D NNN XXX` sur le
  noble `NNN` qui le porte et la capitale `XXX` du fief concerné (il ne vaut
  que pour ce fief), le moteur déterminant son statut :
  - exilé : son propriétaire abandonne le titre, l'usurpateur devient
    titulaire officiel et plus aucune restauration n'est possible ;
  - usurpé : son propriétaire le rend, le titre retourne à l'exilé et le fief
    à son propriétaire, comme à la mort de l'usurpateur.

Les deux ordres sont gratuits. Un noble qui ne porte ni titre exilé ni titre
usurpé sur ce fief voit l'ordre `T D` rejeté.

**Propriétaire d'un fief vacant.** Un fief vacant garde son propriétaire
courant : il ne revient à la couronne que par la règle des fiefs orphelins
(voir [politique.md](politique.md#héritage-dun-fief-sans-héritier)). Quand
plusieurs nobles libres sont présents dans l'armée conquérante, le titulaire
est le premier dans la ligne de succession du joueur. L'exilé n'est jamais un
titulaire pour le score, les votes, les rangs ou les bonus. Conquérir un fief
vacant efface sa marque de fief de la couronne.

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
- émetteur **évêque ou cardinal** et cible un évêché, ou émetteur **pape** sans
  cible (tous les évêchés) : **dîme** (voir
  [religieux.md § Dîme](religieux.md#dîme)).

`XXX` est donc, par exception à la règle « `TER` est le village seed d'une
région », la capitale d'un fief ou le village seed d'un évêché. La production
des moulins n'est jamais touchée par une taxe. Deux cartes de taxe jouées sur
le même fief le même tour ne se cumulent pas : la seconde est consommée sans
effet, avec un rapport explicite. Dans tous les cas, jouer un impôt autorise la
Révolte (voir [ordres-speciaux.md](ordres-speciaux.md)) sur **tout territoire
du fief ou de l'évêché ciblé**, pas seulement sa capitale, la saison où il est
joué et la saison suivante, indépendamment de toute famine.

Le **roi** (ou la régente), par son noble `KKK`, peut taxer n'importe quel fief
constitué, vacant compris, qu'il ne détient pas lui-même, à condition qu'il
ne soit pas déjà taxé par son seigneur ce tour-là : la taxe du seigneur a
priorité quel que soit l'ordre des soumissions, et la taxe royale est alors
consommée sans effet, avec un rapport explicite. Le fief garde son revenu de
base ; un supplément de même montant est créé pour la capitale du joueur du
roi, au lieu de doubler le revenu du fief. Voir
[politique.md](politique.md#pouvoirs-royaux).

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
duché, roi ou pape. Les [dignités](dames.md#dignités) (bâtard, dignités des
dames, carte de cardinal) et les titres d'évêque et de cardinal ne rapportent
aucun point de score. Il
n'y a pas de pondération par rang — un baron et un roi comptent chacun pour
1 point de score, quelle que soit la différence de pouvoir en jeu par
ailleurs (voix, revenus, bonus de titre).

Le score d'un joueur est la somme des titres qu'il détient, ajustée par ses
mariages : voir [succession.md § Mariages et alliances](succession.md#mariages-et-alliances)
pour le calcul du poids d'alliance et des catégories tête/secondaire.

Un mariage qui est la tête active des deux familles est une alliance complète :
pas de bonus individuel, les scores s'additionnent pour la victoire commune.
Tout autre mariage est un mariage d'influence : chaque famille gagne en bonus
les titres détenus par le conjoint de l'autre famille (un par fief dont il est
titulaire, +1 s'il est roi ; jamais son score bonifié), ce qui
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
(voir `architecture.md`). Le simulateur est un mode de la vue « Lignée et alliances » : le bouton « Simuler » permet de cliquer un noble ou un lien de mariage, avec annuler/rétablir/réinitialiser. L'arbre, les liens, les scores, les statuts usurpateur et exilé et le panneau d'état de victoire de chaque maison (score avant/après, seuil, mode, statut) reflètent l'état hypothétique, jusqu'à réinitialisation.

**Actions simulées.** Chaque action s'applique dans l'ordre à une copie de l'état :
- noble : tuer, marier à…, prétendre à…, couronner, destituer le roi (hypothèse sans équivalent en jeu), conquérir un fief, renoncer à une prétention (`T C`), renoncer au titre exilé ou usurpé qu'il porte sur un fief (`T D`) ;
- fief : monter d'un rang (`T R`), reprendre un fief dont le noble est l'exilé (restauration) ;
- lien de mariage : rompre.

La conquête, la restauration et les renonciations appliquent les mêmes règles que le moteur (« Conquête d'un fief par un noble ») : le simulateur et la résolution partagent la même implémentation. Une action impossible est refusée avec son motif (noble inconnu, fief inconnu, fief déjà au propriétaire du conquérant, exilé et usurpateur de la même maison pour une restauration, noble sans titre exilé ni usurpé pour `T D`, noble sans prétention pour `T C`).

**Hypothèses.** Les armées et les positions ne sont pas simulées : le noble choisi pour une conquête est supposé se tenir avec une armée sur la capitale, et l'armée de la maison de l'exilé qui reprend la capitale pour une restauration est supposée présente. Le coût, la limite d'un ordre par hiver, les cartes (la carte de prétention n'est pas défaussée par `T C`) et l'ordre des ordres d'une feuille ne sont pas simulés non plus.

**Variantes de mort.** Tuer un noble applique le règlement de « Durée de l'usurpation » et « Claim » sur l'état courant. Le menu propose chaque issue possible comme une option distincte, avec l'aperçu de qui détient ensuite chaque fief concerné, lu sur le résultat du serveur :
- mort simple : le titre suit les règles de l'état courant (l'héritier du Claim prend le titre, ou un noble libre d'une armée présente sur la capitale usurpe) ;
- mort avec l'autre titulaire d'un fief usurpé (« tuer avec ») : les deux morts sont réglées ensemble, le fief est vacant, reste à son propriétaire du moment et aucun Claim n'est honoré ; la mort simple de l'usurpateur restaure l'exilé, celle de l'exilé rend l'usurpateur officiel, ou donne l'exil à l'héritier d'un Claim sur l'exilé ;
- garnison supposée (noble d'une autre maison que l'héritier, choisi par un clic) ou exclue (« sans garnison ») quand un Claim vise le titulaire d'un fief : la garnison supposée usurpe le fief et l'héritier est exilé, sans garnison l'héritier prend le titre. L'hypothèse vaut pour tous les fiefs que cette mort transmet à un héritier. L'option « sans garnison » n'est proposée que si elle change le résultat.
