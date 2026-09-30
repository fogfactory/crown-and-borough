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
niveau supérieur ou équivalent.

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
libres ou otages à la fois (valeur indicative dans `assets/balance.yaml`, à
calibrer avec le nombre de joueurs). Un noble au cachot compte dans ce
plafond ; un noble mort ou définitivement retiré (bâtard placé en bas de
ligne après annulation papale d'un Claim, voir « Claims ») libère une place.
`R N XXX YYY` est rejeté avec le motif `noble_limit_reached` si le plafond
est déjà atteint.

### Deck de nobles

Le recrutement ne pioche plus librement dans le pool de prénoms de
`assets/prenoms.csv` : chaque joueur pioche dans un **deck de nobles
unique**, fini et partagé par tous les joueurs, généré déterministiquement à
partir de la seed de partie comme le deck d'ordres spéciaux. Le deck
contient :

- des **nobles anonymes** (majorité du deck), nom et sexe tirés de
  `prenoms.csv`, sans trait de départ ;
- des **cartes de personnage**, plus rares, qui recrutent un noble nommé
  avec un trait fixe : cardinaux (avancement facilité vers le cardinalat,
  résolu par [religieux.md](religieux.md)), figures féminines marquantes
  avec un bonus concret et spécifique (commandement renforcé, immunité
  partielle, etc. — distinctes des [titres féminins](dames.md#titres-féminins),
  attribués en cours de partie plutôt qu'au recrutement), et d'autres
  personnages à définir dans l'issue de milestone.

Le recrutement se déroule en deux temps, chacun une entrée d'ordre d'hiver
distincte plutôt qu'une pioche automatique :

- **pioche** (`T N`, un ordre gratuit) : ajoute la carte du dessus du deck à
  la main de cartes noble du joueur, au plus une fois par tour d'hiver
  (`noble_draw_already_used` au-delà) ;
- **jeu** (`R N XXX YYY`, gratuit également, sans coût en R) : joue la carte
  `XXX` présente dans la main du joueur — son trigramme — pour faire
  apparaître le noble correspondant sur le château ou village `YYY`, sous
  réserve des mêmes conditions de contrôle et d'armée présente qu'aujourd'hui.

Cette limite, combinée au plafond de nobles ci-dessus, fait du recrutement
une ressource rare plutôt qu'une action économique libre.

> À trancher dans l'issue de milestone : composition exacte du deck (nombre
> de cartes anonymes vs personnages, liste des personnages et de leurs
> traits) ; ergonomie de la pioche côté UX — la main de cartes noble
> s'ajoute à la main de cartes spéciales existante, donc le formulaire
> d'hiver devra sans doute exposer des boutons dédiés (piocher, jouer une
> carte de la main sur une cible) plutôt qu'une syntaxe brute à composer à
> la main.

## Mariages et alliances

Il n'existe pas de limite au nombre de mariages d'un joueur : chaque noble en
âge de se marier peut contracter le sien, indépendamment des autres nobles du
même joueur. Un joueur peut ainsi être simultanément allié, à des degrés
différents, avec plusieurs autres joueurs.

### Poids d'alliance

Chaque mariage reçoit un poids d'alliance, qui détermine sa catégorie et sert
de départage quand un joueur a plusieurs mariages tête :

```
poids(noble)   = rang_succession(noble) + rang_titre(noble)
poids(couple)  = min(poids(épouxA), poids(épouxB)) + bonus_densité

rang_succession : tête de ligne = 3, second = 2, troisième et suivants = 1
rang_titre      : sans titre = 0, baron = 1, comte = 2, duc = 3, roi/pape = 4
bonus_densité   : +1 par mariage supplémentaire déjà existant entre les deux
                  mêmes maisons, non plafonné
```

`min(...)` retient le maillon le plus faible du couple : un duc qui épouse une
cadette obscure n'obtient pas une tête au même titre qu'un double mariage de
premiers héritiers. Le bonus de densité récompense à l'inverse la
concentration de plusieurs mariages entre les deux mêmes maisons plutôt que
leur dispersion.

Le poids du couple classe le mariage en catégorie, par seuil (valeurs
indicatives, à caler dans `assets/balance.yaml`) :

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

## Claims

L'événement `Claim` permet de recruter un noble héritier qui réclame le titre
d'un seigneur marié à un membre de sa famille. À la mort du marié, le titre lui
revient. Le roi et le pape peuvent annuler le Claim. Une annulation par le pape
rend le seigneur bâtard et le place en bas de la ligne de succession.
