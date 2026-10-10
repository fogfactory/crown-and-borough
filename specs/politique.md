# Politique royale

**Milestone lié :** [Royauté](https://github.com/fogfactory/crown-and-borough/milestone/25) (parent : [#17](https://github.com/fogfactory/crown-and-borough/issues/17)). Le commandement noble relève de l'ancien milestone [Politique royale](https://github.com/fogfactory/crown-and-borough/milestone/6).

**Dépend de :** [Titres & Victoire](titres.md), [Religieux](religieux.md),
[Succession](succession.md) (mariages, Claims) et [Hiver](hiver.md) : les titres
fournissent les voix, le clergé donne l'onction, et tous les ordres royaux sont
des ordres d'hiver.

## Commandement noble

Une armée commandée par un noble gagne `+1` de puissance. Le bonus vaut une
seule fois par armée, même lorsque plusieurs nobles libres sont présents sur la
même case.

Le commandement est reconnu lorsqu'au moins un noble libre appartient au même
joueur que l'armée et se trouve sur sa case. Un noble adverse détenu par
l'armée, ainsi qu'un noble otage ou au cachot, ne fournit pas ce bonus.

Le bonus s'applique à la force de l'attaque, à la force fournie par un soutien
et à la défense de l'armée. Une armée en famine conserve une force de combat et
de déplacement nulle : le bonus de commandement ne l'en sort pas.

## Roi

Le roi est un **noble masculin** (un [bâtard](succession.md#bâtard) ne peut pas
l'être) qui détient la couronne. Il n'y a jamais plus d'un roi. La couronne est
un titre du noble : elle compte pour 1 point de score
([titres.md](titres.md#score-de-titres)) et pour le rang de titre 5 du poids
d'alliance ([succession.md](succession.md#poids-dalliance)). Elle se cumule
avec les fiefs et, le cas échéant, les titres religieux du même noble.

### Élection

**Ouverture.** Le trône est vacant au début de l'hiver (registre figé à
l'étape 0 de [hiver.md](hiver.md#principes), comme les élections religieuses) et
une élection est possible : au moins 3 titres votants existent et le clergé
peut donner l'onction (au moins deux évêques ou un cardinal en titre). Tant
que ces conditions restent vraies et que le trône est vacant, l'élection est
réévaluée à chaque hiver, sans ordre dédié pour la relancer. Un trône vacant
dès le début de la partie suit la même règle.

**Candidature.** `K R NNN` (gratuit) : le joueur présente le noble `NNN`. Il
doit être un homme, libre (ni `hostage` ni `dungeon`), non excommunié, non
bâtard, et détenir au moins un titre de fief (baronnie, comté, marquisat,
duché) à l'instantané ; son état civil est indifférent. Un joueur ne présente
qu'une candidature par élection (la première valide de la feuille).

**Voix.** `V R NNN` (gratuit) : le joueur vote pour un candidat déclaré, quel que
soit son propriétaire ; premier vote valide de la feuille. **Chaque noble
titré donne une voix à son propriétaire**, sans pondération par rang : un
baron, un évêque, un cardinal ou le pape valent chacun une voix. Un noble qui
cumule plusieurs titres n'en donne qu'une. Sont titrés les nobles qui
détiennent un fief (vacant exclu : un fief vacant n'a pas de titulaire), un
évêché, le cardinalat ou la papauté, dames comprises pour les fiefs. Les voix
se lisent sur l'instantané des titres : un titre gagné cet hiver ne vote pas
(principe 3 de [hiver.md](hiver.md#principes)) ; un noble excommunié a perdu
ses titres religieux ; un noble au cachot ne vote pas ; un otage vote pour
son propriétaire.

**Victoire.** Le candidat arrivé en tête est élu si **toutes** les conditions
suivantes sont vraies :

- il reçoit la majorité relative des voix, sans égalité au sommet ;
- il reçoit au moins **3 voix** ;
- l'onction : parmi les voix qu'il reçoit, au moins deux proviennent de nobles
  évêques (ou plus) ou une d'un cardinal ou du pape.

Le candidat arrivé en tête sans l'onction n'est pas élu et le deuxième ne
prend pas sa place : le trône reste vacant jusqu'à l'hiver suivant. Les
bulletins individuels restent privés, le rapport publie le total de voix par
candidat et la mention de l'onction.

L'élection royale est résolue après le conclave, au sein de l'étape 6 de
[hiver.md](hiver.md#ordre-de-résolution). L'élu est couronné à l'investiture
(étape 7) : sa couronne ne donne ni voix ni pouvoir avant l'hiver suivant, et
il n'est plus candidat à aucune autre élection pour ce motif.

### Mandat, capture et mort

- **Mandat viager.** Le roi règne jusqu'à sa mort. Aucune destitution, aucune
  perte de la couronne par excommunication (l'excommunication ne retire que ses
  titres religieux).
- **Capture.** Un roi `hostage` garde ses pouvoirs. Un roi au `dungeon` les voit
  **suspendus** : il ne peut émettre aucun ordre royal ni jouer de taxe royale,
  mais conserve la couronne, son score et sa rente. S'il y a une reine libre,
  celle-ci exerce alors la [régence](#reine-et-régente).
- **Mort.** La couronne passe à l'héritier d'un [Claim](succession.md#prétentions-claims)
  sur le couple royal s'il y en a un (voir ci-dessous), sinon la reine devient
  régente et le trône est vacant : une élection est organisée dès que le
  registre le permet.

### Claim sur le couple royal

Un Claim visant le roi **ou sa reine** cible, en plus des fiefs, la couronne. À
la mort du roi, ses fiefs et la couronne passent au premier héritier vivant
dans le classement habituel des Claims (ancienneté, famille de l'épouse
d'abord, puis ordre de jeu) qui peut être roi (homme, non bâtard) ; un héritier
qui ne le peut pas est sauté pour la couronne mais conserve son droit sur les
fiefs. L'héritier couronné est roi sans élection ni onction, et sans condition
de rang de succession. Une prétention sur la reine est donc aussi une
prétention sur le trône : elle profite à la maison de l'héritier, y compris
celle de la reine. La mort de la reine seule ne transmet que ses éventuels
fiefs. Sans héritier éligible, la couronne reste vacante (reine régente,
élection).

## Pouvoirs royaux

Tous les pouvoirs ci-dessous sont exercés par le roi (noble `KKK` du joueur,
libre ou otage, pas au cachot) ou, à défaut, par la [régente](#reine-et-régente).
Les ordres royaux d'hiver portent le préfixe `W` et sont gratuits. Ils se
résolvent avant les ordres de gestion (étape 2a de
[hiver.md](hiver.md#ordre-de-résolution)), car les ordres de gestion qu'ils
autorisent les lisent. Un ordre royal qui ne trouve pas l'ordre qu'il
autorise cet hiver est consommé sans effet (rapport explicite).

| Ordre | Pouvoir | Règle |
|---|---|---|
| `P TX HHH XXX` | **Taxe royale** | Carte Impôts du roi sur la capitale de n'importe quel fief constitué, vacant compris, non taxé par son seigneur ce tour ; le supplément va à la capitale du roi ([titres.md](titres.md#taxe-seigneuriale)). |
| `W X KKK XXX YYY` | **Accord d'annexion** | Autorise le fief de capitale `XXX` à annexer le territoire `YYY` d'un autre fief ([Annexion](#seigneurie-et-annexion)). |
| `W R KKK XXX` | **Accord de rang** | Accorde gratuitement au fief `XXX` le titre correspondant à sa taille. |
| `W F KKK XXX` | **Octroi gratuit de fief** | Le fief dont la capitale est `XXX`, constitué cet hiver par `T F`, ne coûte rien à son joueur. |
| `W G KKK XXX NNN` | **Octroi d'un fief de la couronne** | Donne le fief de la couronne de capitale `XXX` au noble `NNN`, de n'importe quel joueur, sous réserve de la ligne de succession. |

Un roi émet au plus `royal.orders_per_winter` ordres royaux par hiver (valeur
de départ : un par type, à calibrer). Le roi peut octroyer à ses propres nobles
comme à ceux des autres joueurs.

### Rente

Chaque tour (à la fin de la résolution du tour, comme le revenu territorial),
le roi gagne **1 R par fief constitué** sur la carte, quel que soit son
propriétaire, vacants et fiefs du roi compris. La rente est créée, jamais
prélevée sur les fiefs, et livrée à la capitale du joueur du roi. Elle ne
dépend pas de l'état du roi : un roi capturé continue de la percevoir pour son
joueur. Aucune rente en l'absence de roi.

La reine gagne en plus la **moitié de la rente, arrondie au supérieur**, livrée
à la capitale du joueur de la reine. Sa rente ne diminue pas celle du roi. Le
rapport de fin de tour la mentionne séparément.

### Seigneurie et annexion

L'annexion agrandit un fief existant. Elle s'ajoute à la constitution
([titres.md](titres.md#constitution-dun-fief)) et la remplace pour
l'agrandissement différé qui y était mentionné. Le titre du fief ne change
**jamais** automatiquement : annexer ajoute des territoires, changer de rang
est un ordre distinct.

**Annexion.** `T X NNN XXX YYY`, ordre d'hiver de gestion (étape 2) : le
noble `NNN`, titulaire du fief de capitale `XXX`, annexe le territoire `YYY`.
Coût : `fief_per_territory` (2 R), payé comme un investissement d'hiver ciblé
sur `YYY`. Conditions, vérifiées avant tout paiement :

- `NNN` appartient au joueur et est le titulaire du fief `XXX`
  (`annex_not_titular`) ;
- `YYY` est adjacent, par une frontière franchissable, à un territoire du
  fief (`annex_not_adjacent`) ;
- le joueur **occupe** `YYY` : une de ses armées y stationne, et aucune armée
  adverse ni `NEUTRAL` (`annex_not_occupied`) ;
- `YYY` n'est la capitale d'aucun fief (`annex_capital`) ;
- si `YYY` est déjà membre d'un fief (le même joueur ou un autre), un
  `W X KKK XXX YYY` du roi ou de la régente est présent cet hiver
  (`annex_royal_accord_required`) et le fief cédant garde au moins 3
  territoires (`annex_donor_too_small`) ; sinon `YYY` est un territoire
  contrôlé hors fief.

Effet : `YYY` rejoint le fief, quitte son ancien fief le cas échéant (avec son
contrôle transféré au propriétaire du fief annexant), et son revenu va
désormais à `XXX`. Aucun titre n'est modifié, y compris celui du fief cédant.
Un fief sans roi ne peut donc annexer que des territoires hors fief.

**Changement de rang.** `T R NNN XXX` (étape 2) : le titulaire `NNN` du fief
`XXX` ajuste le titre au barème de [titres.md](titres.md#constitution-dun-fief)
pour sa taille actuelle. Le titre ne fait que monter (`rank_unchanged` si le
rang correspondant n'est pas supérieur). Deux voies :

- **accord royal** : si un `W R KKK XXX` est présent cet hiver (donc seulement
  tant qu'il y a un roi ou une régente), le changement est gratuit ;
- **paiement** : sans accord, le joueur paie `fief_per_territory` pour
  **chaque** territoire du fief (réseau de paiement d'hiver habituel), comme
  pour la constitution d'un fief de cette taille.

Dans les deux cas, la ligne de succession s'applique au titre atteint
(`succession_rank_blocked`).

**Octroi gratuit de fief.** À la constitution d'un fief par `T F`, le roi peut
prendre le coût à sa charge par `W F KKK XXX` : le joueur ne paie rien.
L'ordre `T F` reste soumis à toutes ses autres conditions.

## Reine et régente

**Reine.** La reine est la conjointe du roi : le titre de courtoisie de
[titres.md](titres.md#titres-de-courtoisie) devient mécanique pour elle seule.
Elle appartient à un autre joueur que le roi (un mariage lie toujours deux
joueurs distincts). Elle n'a **aucun pouvoir d'ordre** ; son seul effet est sa
[rente](#rente). Elle cesse d'être reine à la mort du roi, au veuvage (mariage
terminé), ou si le mariage est dissous. Le roi se marie selon les règles
ordinaires de [succession.md](succession.md#conclusion-dun-mariage) ; il n'est
pas tenu d'être célibataire.

**Régence.** La régente est la reine qui exerce les pouvoirs du roi en son
nom, sauf la rente du roi (elle conserve la sienne), dans deux cas :

1. **roi au cachot** : tant que le roi est au `dungeon` et que la reine est
   libre ou otage, elle exerce tous les pouvoirs royaux ; la régence cesse dès
   que le roi est libéré ;
2. **trône vacant à la mort du roi sans héritier éligible** : la reine devenue
   veuve (conjointe au moment de la mort) hérite des prérogatives du roi
   jusqu'au couronnement d'un nouveau roi. Elle les perd si elle meurt, est
   mise au cachot, ou se remarie. Le trône reste vacant : l'élection a lieu
   normalement, et elle ne peut pas se faire élire (sexe).

Les ordres royaux (`W …`) et la taxe royale sont émis par le noble de la
régente. Elle ne choisit pas le roi, ne cumule pas la couronne, et la rente
du roi n'est pas versée pendant la vacance.

## Héritage d'un fief sans héritier

Quand le titulaire d'un fief meurt, l'ordre existant de
[succession.md](succession.md#prétentions-claims) et de
[titres.md](titres.md#perte-et-vacance-dun-fief) s'applique : Claim d'abord,
sinon fief vacant chez son joueur puis attribué au premier noble de sa ligne.
Quand le joueur n'a **plus aucun noble vivant** (aucun noble à qui attribuer)
et qu'aucun héritier de Claim ne reprend le fief, le fief **retourne
immédiatement à la couronne** s'il y a un roi ou une régente :

- le fief passe au joueur de la couronne (roi ou régente), vacant, et le
  contrôle de tous ses territoires bascule dans la même passe (comme à la prise
  d'une capitale, [titres.md](titres.md#perte-et-vacance-dun-fief)) ;
- c'est un **fief de la couronne** : il n'est jamais attribué par défaut en fin
  d'hiver à la ligne du joueur de la couronne ; il reste vacant, continue de
  produire et de compter son point, jusqu'à un `W G` ;
- sans roi ni régente, le fief reste vacant chez son joueur. Au couronnement
  d'un roi, tous les fiefs vacants dont le joueur est sans noble vivant font
  retour à la couronne dans la même passe.

## À trancher

Points d'interprétation à confirmer ou à calibrer pendant l'implémentation :

1. Les dames titrées votent-elles au trône (fiefs ouverts aux deux sexes), en
   dépit de la piste « réservé aux titres masculins » de
   [dames.md](dames.md#réseau-et-influence-de-cour) ? Ce document suppose que
   oui, la piste visant les titres religieux.
2. Le coût du changement de rang payant (2 R × taille totale) et la limite
   `royal.orders_per_winter` sont à calibrer.
3. Un fief annexé à un autre joueur par accord royal est une expropriation :
   faut-il une contrepartie ou un droit de réponse ?
4. Le même noble peut-il être roi et pape ? Aucune exclusion n'est posée.
5. Les fiefs de la couronne offrent au joueur du roi une grande source de
   score et de revenu via la rente et la capitale : à valider en équilibrage.
6. Joueur éliminé : le sort d'une couronne, d'un fief de la couronne ou d'une
   régence d'un joueur éliminé n'est pas traité ici.
