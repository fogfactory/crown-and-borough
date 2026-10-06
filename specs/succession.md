# Succession et mariages

**Milestone lié :** [Succession & Couronnement](https://github.com/fogfactory/crown-and-borough/milestone/24)

**Dépend de :** [Titres & Victoire](titres.md), [Religieux](religieux.md),
[Politique royale](politique.md) et les cartes du deck d'ordres spéciaux.

## Lignée

L'arbre généalogique doit permettre de représenter les filiations, les décès,
les exécutions, les assassinats, les mariages et les Claims. Tous les nobles
d'un joueur, quel que soit leur sexe, appartiennent à la ligne de succession
dans leur ordre d'achat.

Un titre de seigneur ne peut être octroyé à un noble que si tous les nobles
placés au-dessus de lui dans la ligne de succession possèdent déjà un titre de
niveau supérieur ou équivalent. Un [bâtard](#bâtard) est toujours placé en
dernier de la ligne et ne reçoit un titre que s'il est le dernier de sa lignée.

## Sexe des nobles

Chaque noble a un sexe, masculin ou féminin, déterminé à son recrutement.
Seul un noble masculin peut devenir évêque, cardinal, pape ou roi ; les titres
séculiers (baron, comte, duc) et la ligne de succession restent ouverts aux
deux sexes, dans leur ordre d'achat commun. Les mécaniques additionnelles
propres aux seigneurs femmes — au-delà du rôle de pièce de mariage — sont
définies dans [dames.md](dames.md).

## Recrutement des nobles

**Dépend aussi de :** [Ordres spéciaux](ordres-speciaux.md) pour la structure
de deck réutilisée, [Religieux](religieux.md) pour les cardinaux et
[dames.md](dames.md) pour les cartes de dot.

### Plafond de nobles

Chaque joueur ne peut détenir plus d'un nombre fixe de nobles vivants et
libres, otages ou au cachot à la fois : `noble_limit` (4) dans
`assets/balance.yaml`. Certains effets de jeu peuvent relever ce plafond,
sans jamais dépasser `noble_limit_max` (6) ; le premier est la dignité de
[bâtard](#bâtard), dont chaque porteur relève le plafond de 1 (deux bâtards :
+2), quels que soient son statut (libre, otage ou au cachot) et son état
civil. Un
noble au cachot compte dans ce plafond ; un noble mort ou définitivement
retiré (bâtard placé en bas de ligne après annulation papale d'un Claim, voir
« Claims ») libère une place. `R N CCC XXX` est rejeté avec le motif
`noble_limit_reached` si le plafond est déjà atteint. Si le plafond baisse
(mort du porteur d'une dignité de bâtard), un joueur peut se retrouver
au-dessus : aucun noble n'est expulsé, le recrutement est seulement bloqué
jusqu'à repasser sous le plafond.

### Deck de nobles

Chaque joueur pioche dans un **deck de nobles unique**, partagé par
tous les joueurs, généré déterministiquement à partir de la seed de partie
comme le deck d'ordres spéciaux. Il contient trois sortes de cartes :

- des **cartes de noble**, chacune portant un nom, un trigramme et un sexe
  tirés de `assets/prenoms.csv`, sans trait de départ. Elles sont réparties
  à parts égales entre hommes et femmes (la seed tranche la carte restante
  quand leur nombre est impair) ;
- des **cartes de prétention** (code `CLM`), qui ne recrutent personne : elles
  se jouent avec `C N HHH CCC` pour qu'un noble du joueur réclame les titres
  d'un noble d'une autre famille, voir [Prétentions](#prétentions-claims) ;
- des **cartes de dignité**, qui ne recrutent personne : elles se jouent sur
  un noble déjà en jeu pour lui conférer une [dignité](dames.md#dignités).
  La seule dignité du deck est le [bâtard](#bâtard) ; d'autres s'y ajouteront
  avec les issues des cardinaux et des figures féminines.

**Taille.** Le deck compte `joueurs × (noble_limit_max + 1)` cartes (28 à
4 joueurs), réduit au besoin pour que les cartes de noble puissent recevoir
un nom encore libre (les nobles de départ en consomment aussi). Parmi elles,
les dignités duplicables représentent une carte sur `joueurs − 1`, sans
dépasser une carte sur quatre : `max(1, deck / max(joueurs − 1, 4))` cartes,
arrondies à l'entier inférieur. Ce quota garantit au moins un bâtard par
partie ; le reste du deck est constitué de cartes de noble. Les cartes de
prétention s'ajoutent à cette taille sans remplacer de carte de noble : une
carte sur `2 × max(joueurs − 1, 4)` de la taille de base, au moins une (31
cartes à 4 joueurs, dont 3 de prétention).

La **pioche** (`T N`) et la **défausse** (`D C CCC`) sont des ordres d'hiver. Les
ordres qui **jouent** une carte (`R N`, `C N`, `D N`) se soumettent à
n'importe quelle saison, d'action ou d'hiver : ils s'appliquent au début de la
résolution, avant les ordres d'armée, et un noble recruté ne participe au tour
qu'à partir du suivant. Le recrutement se déroule en deux temps, chacun une
entrée d'ordre distincte :

- **pioche** (`T N`, un ordre gratuit) : ajoute la carte du dessus du deck à
  la main de cartes de noble du joueur, au plus une fois par joueur et par
  tour d'hiver (`noble_draw_already_used` au-delà). La limite de main est
  partagée avec la main d'ordres spéciaux : la main de cartes de noble
  (nobles et dignités) et la main d'ordres spéciaux ne peuvent ensemble
  dépasser `special_orders.hand_limit` cartes, sinon l'ordre est rejeté
  (`hand_limit_reached`). Un joueur pioche au plus `special_orders.draw_orders_limit`
  cartes par hiver, tous decks confondus, dont au plus une dans le deck de
  personnages (nobles et dignités) : une pioche de noble réduit d'une carte
  le remplissage automatique de la main d'ordres spéciaux. Quand la pioche est vide,
  la défausse est mélangée pour la reconstituer ; si la pioche et la défausse
  sont toutes deux vides, l'ordre est rejeté (`noble_deck_empty`) ;
- **jeu d'une carte de noble** (`R N CCC XXX`, gratuit, sans coût en R) : joue
  la carte `CCC` de la main — son trigramme — pour faire apparaître le noble
  correspondant sur le château ou village `XXX`, sous réserve des conditions
  de contrôle, d'armée présente et de plafond ;
- **jeu d'une carte de dignité** (`D N XXX CCC`, gratuit) : joue la carte de
  dignité `CCC` de la main (son code, `BAS` pour le bâtard) sur le noble
  `XXX`, qu'il appartienne au joueur ou à un adversaire — **toute dignité se
  joue sur n'importe quel noble, secrètes comprises** ; ce n'est pas une
  propriété qui varierait d'une dignité à l'autre. Deux cartes de même
  dignité sont interchangeables ;
- **défausse d'une carte de la main** (`D C CCC`, gratuit, sans limite par
  hiver) : retire de la main la carte `CCC` (trigramme d'un noble ou code d'une
  dignité) sans la jouer ; elle va à la défausse du deck de nobles telle
  quelle et y reste jusqu'au mélange de la défausse. La place libérée peut
  servir à un `T N` plus loin dans la même feuille. Une carte absente de la
  main est rejetée (`card_not_in_hand`). Le rapport public ne nomme pas la
  carte défaussée. `D C` suivi d'un code à deux lettres reste la défausse
  d'une carte d'ordre spécial.

Cette limite, combinée au plafond de nobles, fait du recrutement une
ressource rare plutôt qu'une action économique libre.

**Défausse et remélange.** Une carte jouée reste suivie, liée au noble sur
lequel elle agit, tant que ce noble vit.

- Quand un noble recruté par une carte meurt ou quitte définitivement le jeu,
  sa carte de noble sort du deck et une **nouvelle carte de noble du même
  sexe**, portant un nom et un trigramme encore inutilisés (ni sur une carte,
  ni sur un noble, ni sur un noble mort), rejoint la défausse ; s'il n'en reste
  aucun, rien n'est ajouté. Le trigramme du défunt reste réservé. Un noble de
  départ, qui n'a pas de carte, ne laisse rien.
- Une carte de dignité retourne à la défausse quand son porteur meurt ou quand
  la dignité est retirée par un effet de jeu (point d'entrée unique du moteur,
  `removeDignity`).
- Quand une pioche trouve la pile vide, la défausse est mélangée de façon
  déterministe (graine dérivée de la seed de partie et d'un compteur de
  remélanges stocké dans l'état) pour former la nouvelle pioche.

### Bâtard

Le bâtard est une dignité permanente, attribuable à tout noble du joueur par
une carte de dignité (ou par l'annulation papale d'un Claim, voir
« Claims »). Elle se cumule avec toute autre dignité, quel que soit le sexe,
l'état civil ou le statut du noble ; elle ne s'attribue qu'une fois au même
noble (`noble_already_bastard`) et compte comme un titre dans le score.

- **Plafond.** Chaque noble du joueur qui est bâtard relève `noble_limit` de 1
  (deux bâtards : +2), sans jamais dépasser `noble_limit_max`, même si le
  porteur est marié, otage ou au cachot. Le bonus disparaît à sa mort.
- **Succession.** Un bâtard est toujours dernier de la ligne de succession,
  quel que soit son ordre d'achat. Il ne peut recevoir un nouveau titre que s'il
  est le dernier membre de sa lignée (tous les autres nobles du joueur étant
  déjà morts ou bâtards, ou le joueur n'ayant aucun noble non bâtard). Un
  noble qui devient bâtard conserve les titres qu'il détient. Un bâtard ne
  peut pas être roi.
- **Mariage.** Un bâtard peut se marier, mais son mariage n'est pas
  constitutif d'une alliance : il est enregistré dans la lignée sans poids,
  sans catégorie, sans partage de score et sans compter dans le bonus de
  densité.
- **Capture.** Un bâtard capturé en combat est placé directement au cachot,
  jamais en otage. Il reste susceptible d'être otage s'il est remis à un
  autre joueur par un effet autre que la capture (otage volontaire) ou si son
  propriétaire change son statut.
- **Claims.** Un bâtard ne peut pas être l'héritier d'un Claim ; jouer une
  carte de bâtard sur l'héritier — y compris celui d'un adversaire, comme
  toute carte de dignité — annule son Claim (voir « Prétentions »). Sans
  condition d'autorité (ce n'est pas une prérogative royale), c'est ce qui
  rend inutile un ordre séparé d'annulation de Claim par le roi ou le pape.
- **Deck d'ordres spéciaux.** Les cartes du deck d'ordres spéciaux peuvent
  cibler ou reconnaître un bâtard (prédicat de ciblage `is_bastard`).

Les effets d'une dignité sont déclarés en un seul endroit du moteur (table de
dignités) que chaque règle concernée interroge (plafond, succession, titres,
mariage, capture, ciblage) : une nouvelle dignité ajoute une entrée plutôt
qu'un test dispersé.

## Mariages et alliances

Il n'existe pas de limite au nombre de mariages d'un joueur : chaque noble en
âge de se marier peut contracter le sien, indépendamment des autres nobles du
même joueur. Un joueur peut ainsi être simultanément allié, à des degrés
différents, avec plusieurs autres joueurs.

### Conclusion d'un mariage

Un mariage se conclut en hiver par un ordre **symétrique**, gratuit et sans
noble émetteur (il n'est pas une chaîne) :

- le joueur qui possède le noble `XXX` soumet `M N XXX YYY` ;
- le joueur qui possède le noble `YYY` soumet `M N YYY XXX`.

Le consentement est **simultané** : le mariage n'existe que si les deux ordres
sont soumis le même hiver, chacun nommant l'autre noble. Il n'y a ni
proposition en attente ni acceptation différée : une demande sans
réciproque est rejetée (`marriage_not_reciprocated`) et doit être soumise à
nouveau l'hiver suivant.

Le refus est visible des deux côtés. Le rapport du joueur qui a soumis l'ordre
porte le rejet ; celui de l'autre joueur, qui n'a pas donné l'ordre
réciproque, signale le mariage refusé. Les autres joueurs reçoivent une
**rumeur** (section `marriages` du rapport, résultat `failure`) : « la
négociation de mariage entre XXX et YYY a échoué ». La rumeur est systématique
et nomme les deux nobles, sans préciser qui avait fait la demande.

Conditions, vérifiées pour chacun des deux nobles ; un ordre qui n'en
respecte pas une est rejeté avec le motif indiqué :

- `XXX` appartient au joueur qui soumet l'ordre (`noble_not_owned`) ;
- `XXX` et `YYY` appartiennent à deux joueurs distincts
  (`marriage_same_owner`) ;
- `XXX` et `YYY` sont de sexe différent, un homme et une dame
  (`marriage_same_sex`) ;
- les deux nobles sont libres : ni otage ni au cachot (`noble_not_free`) ;
- aucun des deux n'est déjà marié (`noble_already_married`).
- aucun des deux ne porte une dignité qui interdit le mariage
  (`marriage_forbidden`). Cette condition est un point d'extension : tant que
  les dignités n'existent pas ([dames.md](dames.md#dignités)), elle ne rejette
  rien ; l'issue des dignités (#259) précise lesquelles interdisent le
  mariage, la règle posée dans dames.md étant qu'une dame titulaire d'une
  dignité ne peut pas se marier tant qu'elle la détient.

Un noble n'a qu'un seul mariage à la fois. Le mariage prend fin à la mort de
l'un des époux (il reste enregistré dans la lignée) et le survivant peut se
remarier. Il perd alors, le cas échéant, le titre obtenu par alliance, et
ses bonus de score s'éteignent avec le mariage (voir les sections
suivantes). Si un
même noble figure dans plusieurs couples d'ordres réciproques, seul le
premier dans l'ordre de résolution (joueurs par identifiant, puis ordre de la
feuille) est conclu ; les autres sont rejetés `noble_already_married`.

Les ordres de mariage sont résolus après tous les autres ordres d'hiver
individuels du tour : un noble libéré (`L N`) le même hiver peut donc être
marié. La conclusion enregistre le mariage sans en calculer les effets : le
poids, la catégorie et les bonus relèvent des sections suivantes.

**Annonce.** Un mariage conclu est public : le rapport de tous les joueurs
(et des spectateurs) mentionne les deux nobles et leurs maisons dans la
section `marriages` du rapport (résultat `success`) et l'état de partie les
expose à tous.

### Poids d'alliance

Chaque mariage reçoit un poids d'alliance, qui détermine sa catégorie et sert
de départage quand un joueur a plusieurs mariages tête :

```
poids(noble)   = rang_succession(noble) + rang_titre(noble)
poids(couple)  = min(poids(épouxA), poids(épouxB)) + bonus_densité

rang_succession : tête de ligne = 3, second = 2, troisième et suivants = 1
rang_titre      : sans titre = 0, baron = 1, comte = 2, marquis = 3, duc = 4,
                  roi/pape = 5 (à ajouter avec la couronne)
bonus_densité   : +1 par mariage supplémentaire déjà existant entre les deux
                  mêmes maisons, non plafonné
```

Les rangs de succession, les rangs de titre et le bonus de densité sont
configurés dans `assets/balance.yaml` (section `alliance`).

`min(...)` retient le maillon le plus faible du couple : un duc qui épouse une
cadette obscure n'obtient pas une tête au même titre qu'un double mariage de
premiers héritiers. Le bonus de densité récompense à l'inverse la
concentration de plusieurs mariages entre les deux mêmes maisons plutôt que
leur dispersion.

Un mariage dont l'un des époux est [bâtard](#bâtard) n'est pas une alliance :
il n'a ni poids, ni catégorie, et ne compte pas dans le bonus de densité.

Le poids du couple classe le mariage en catégorie, par seuil
(`head_min_weight` et `mixed_min_weight`, section `alliance` de
`assets/balance.yaml`) :

| Catégorie | Poids | Effet |
|---|---:|---|
| Tête | ≥ 5 | Alliance complète : score additionné, victoire commune possible (voir [titres.md § Victoire majeure, victoire mineure, échec](titres.md#victoire-majeure-victoire-mineure-échec)) |
| Mixte | 2 à 4 | La moitié des titres du conjoint s'ajoute au score, sans victoire commune |
| Secondaire | ≤ 1 | +1 point par titre du conjoint, sans partage de score |

> À trancher : seuils indicatifs, à valider avec la balance générale du score
> de titres.

### Tête active

Un joueur peut détenir plusieurs mariages tête simultanément, portés par des
nobles différents. Un seul est actif à la fois pour la victoire commune :
celui au poids d'alliance le plus élevé. Les autres têtes du même joueur
comptent alors comme des mixtes (50 % des titres du conjoint, sans victoire
commune) tant qu'une tête supérieure reste active.

La tête active change dynamiquement, sans jamais nécessiter de déclaration du
joueur :

- au décès, à l'exécution ou à l'assassinat du noble qui la porte, l'alliance
  bascule au mariage tête suivant dans l'ordre de poids ;
- un Claim, une annulation par le pape, ou un changement de titre (élection,
  succession) peut modifier le poids d'un mariage et donc reclasser sa
  catégorie ou son statut d'activité.

Tant qu'un joueur a une tête active, il ne peut plus gagner seul : sa
victoire majeure ne peut être que commune avec son conjoint de tête, contre
un seuil plus élevé que le seuil solo (voir [titres.md § Seuil de victoire et fin de partie](titres.md#seuil-de-victoire-et-fin-de-partie)).
Épouser une tête est donc un engagement stratégique, pas seulement un bonus
de score.

### Droits, exécutions et joueurs éliminés

La règle devra encore préciser (issue de milestone) :

- les droits transférés par un mariage tête sur les décisions de l'autre
  maison (annexion, taxation, élections) ;
- les conséquences d'une exécution du conjoint sur le poids et la catégorie
  du mariage restant ;
- le traitement d'un mariage entre deux joueurs éliminés, ou entre un joueur
  éliminé et un joueur actif — notamment si le score du joueur éliminé
  continue de compter pour son ex-conjoint.

## Otage volontaire

**Dépend aussi de :** la réception des nobles et le statut `hostage` du GDD
§6.

Un nouvel ordre d'hiver permet à un joueur de remettre volontairement un de
ses propres nobles libres à un autre joueur, qui le reçoit en statut
`hostage` sans combat ni capture. Contrairement à une capture, l'envoi est
unilatéral et ne demande pas le consentement du destinataire (comme le
transfert de ressources `G XXX YYY N` du GDD §2, qui n'exige pas non plus que
la destination appartienne au donneur).

Cet ordre combine deux mécaniques déjà en place :

- **connaissance de chaîne** : comme tout détenteur d'otage, le joueur
  receveur connaît les chaînes émises par le noble reçu tant qu'il reste
  hostage (GDD §4) — envoyer un noble volontairement, c'est donc aussi
  choisir de rendre son activité visible à un tiers, un geste diplomatique
  lisible plutôt qu'un pur repli défensif ;
- **effets passifs des dames** : lorsque le noble envoyé est une dame
  porteuse d'un effet de cour ou d'une carte de dot (voir
  [dames.md](dames.md)), l'effet passif s'applique désormais au **joueur qui
  la détient**, pas à son propriétaire d'origine. Envoyer une dame savante
  en gage devient un instrument diplomatique à part entière — prêter un
  savoir plutôt qu'un simple otage de valeur.

> À trancher dans l'issue de milestone : syntaxe exacte de l'ordre et
> condition de ciblage (territoire portant une armée du destinataire,
> capitale du destinataire, ou autre point d'ancrage) ; si le propriétaire
> d'origine conserve un droit de rappel ou de rançon au-delà de la libération
> standard `L N NNN` par le détenteur ; et si un noble envoyé volontairement
> peut refuser d'émettre des chaînes pour se soustraire à l'observation
> (probablement non, pour rester cohérent avec le statut `hostage` existant).

## Prétentions (Claims)

Un **Claim** (prétention) permet à un noble « héritier », né d'une alliance,
de réclamer les titres d'un seigneur de l'autre famille. Quand ce seigneur
meurt, les titres de fief qu'il détient reviennent à l'héritier. Il n'existe
pas d'ordre dédié d'annulation du Claim par le roi ou le pape : jouer une
carte de bâtard sur l'héritier, y compris celui d'un adversaire, annule sa
prétention (voir « Bâtard » ci-dessus et « Bâtard » plus bas) ; le pape
dispose par ailleurs de l'excommunication (voir [dames.md § Carte de
procès](dames.md#carte-de-procès)) pour les cas que le bâtard ne couvre pas.

**Ordre.** `C N HHH CCC` (gratuit) consomme une **carte de prétention** de la main de
cartes de noble du joueur (`card_not_in_hand` sinon ; la carte reste sur
l'héritier tant que sa prétention vit et retourne à la défausse quand elle
s'éteint) : `HHH` est un noble du
joueur, l'héritier, et `CCC` un noble d'un autre joueur dont il réclame les
titres. Conditions, un ordre qui n'en respecte pas une est rejeté avec le
motif indiqué :

- `HHH` appartient au joueur (`noble_not_owned`) et `CCC` à un autre joueur
  (`claim_on_own_noble`) ;
- `HHH` a été posé (recruté) pendant un mariage entre `CCC` et l'un des
  nobles du joueur : le mariage était conclu au tour de la pose et n'avait pas
  pris fin avant elle par la mort d'un époux (`claim_requires_marriage`). Le
  mariage peut avoir pris fin depuis. Un noble de départ n'a jamais été posé
  pendant un mariage. Le moteur retient, pour cela, le tour de pose de chaque
  noble et les tours de conclusion des mariages et de décès ;
- `HHH` n'est pas bâtard (`claim_by_bastard`) et ne porte pas déjà une
  prétention (`claim_already_staked`) : un héritier ne réclame qu'un seul
  noble.

**Empilement.** Les Claims s'empilent : plusieurs héritiers, de la même
famille ou de l'autre, peuvent réclamer les titres d'un même noble, sur un
même couple ou non. Ils sont classés par ancienneté (le Claim joué le plus
tôt d'abord) ; parmi les Claims du même hiver, celui de la famille de
l'épouse passe en premier, puis l'ordre de jeu départage.

**Effet.** Quand `CCC` meurt, chaque fief dont il est titulaire passe tout
entier, avec ses territoires, au propriétaire du premier héritier vivant dans
le classement, et cet héritier en devient le titulaire, sans condition de rang
de succession (événement public de fief changeant de main, motif `claim`).
Sans fief à `CCC` ou sans héritier vivant, les prétentions s'éteignent sans effet ; une fois le fief transmis, les autres prétentions sur `CCC` s'éteignent aussi.
Les titres royaux ne sont pas encore implémentés : ils suivront la même règle.

**Bâtard.** Jouer une carte de bâtard sur l'héritier annule sa prétention.
Une carte de bâtard jouée sur un noble « parent » (le noble visé ou son
conjoint) n'annule pas la prétention.

**Visibilité.** Une prétention est publique : elle figure dans le rapport de
tous les joueurs dès qu'elle est jouée.
