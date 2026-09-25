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

- **Contrôlé** : le statut porté par la case (`OwnerID`). Hors fief, il est
  positionnel (dernier joueur dont une armée s'est arrêtée sur la case) ; dans
  un fief, il est transitif (joueur qui détient le fief). Voir
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
| Mauvaise récolte | Rations de terrain et production des châteaux et villages supprimées. |
| Récolte abondante | Annule la mauvaise récolte ; sinon, rations de terrain et production des châteaux et villages doublées. |

Les bonus additifs de la v1 (`bonus_mill_production`, `bonus_army_ration`)
sont retirés de la balance. Quand le revenu territorial (#192) remplacera
`base_production`, la récolte s'appliquera à ce revenu (territoires et
villages) de la même façon.

## Revenu territorial

Issues : [#192](https://github.com/fogfactory/crown-and-borough/issues/192)
(hors fief), [#196](https://github.com/fogfactory/crown-and-borough/issues/196)
(fiefs).

- À chaque tour d'action (printemps, été, automne ; jamais en hiver), chaque
  territoire contrôlé rapporte `territory_income` R (1 par défaut), plus
  `village_income` R (1 par défaut) s'il porte un village.
- Le revenu est crédité au moment où la production est créditée aujourd'hui,
  avant le ravitaillement. Il n'est pas acheminé par le réseau et ne peut pas
  être intercepté.
- Hors fief, il est versé au stock de la **capitale du joueur**. Sans
  capitale, il va au château contrôlé le plus proche (distance en frontières
  franchissables, départage par trigramme), sinon au village contrôlé le plus
  proche, sinon il est perdu.
- Dans un fief, il est versé au stock de la **capitale du fief**, y compris
  lorsqu'une armée adverse occupe le territoire.
- La production de base des châteaux et villages contrôlés
  (`base_production`) est supprimée : le revenu territorial la remplace.
- Un village **neutre** continue de produire 1 R par tour dans son propre
  stock, récupéré à sa capture.

> À trancher dans #192 : calibrer `territory_income` par des parties de test.

## Flux de la ressource R

| Source | Bénéficiaire par défaut | Avec taxe |
|---|---|---|
| Territoire d'un fief, sans village | 1 R → capitale du fief | Seigneur : 2 R → capitale du fief. Roi (si le seigneur ne taxe pas ce tour) : +1 R → capitale du roi, 1 R de base reste au fief |
| Territoire d'un fief, avec village | 2 R → capitale du fief (1 territoire + 1 village) | Seigneur : 4 R → capitale du fief. Roi (si non taxé localement) : +2 R → capitale du roi, 2 R de base restent au fief |
| Territoire contrôlé hors fief, sans village | 1 R → capitale du joueur | — (pas de taxe hors fief) |
| Territoire contrôlé hors fief, avec village | 2 R → capitale du joueur | — |
| Village neutre | 1 R → son propre stock | — |
| Moulin (niveau `N`) | `N` R → château adjacent, sinon village adjacent, sinon reste sur le moulin | Dîme religieuse jouée sur l'évêché : `N` R → capitale du joueur qui a joué la dîme, au lieu du village/château adjacent |

La taxe du seigneur est livrée par
[#189](https://github.com/fogfactory/crown-and-borough/issues/189) ; la taxe
royale et la dîme restent suivies dans les milestones Politique royale et
Religieux.

La taxe seigneuriale double toujours exactement le revenu de territoire,
village inclus ; elle ne touche jamais la production des moulins. La dîme
(voir [religieux.md](religieux.md)) fait l'inverse : elle ne touche que les
moulins d'un évêché, jamais le revenu de territoire. Les deux mécaniques
portent donc sur des flux disjoints et ne peuvent pas entrer en conflit sur le
même territoire le même tour.

## Moulins

Issue : [#195](https://github.com/fogfactory/crown-and-borough/issues/195).

Un moulin de niveau `N` produit `N` R (`2N` sous le Beau temps) et
verse sa production à **une seule** infrastructure : le château adjacent,
sinon le village adjacent, sinon la case du moulin elle-même. Entre plusieurs
candidats du même type, le départage se fait par trigramme. Un moulin ne
compte donc plus pour chaque château ou village adjacent. Sous le mauvais
temps, il ne produit rien.

Une production restée sur un moulin isolé n'est pas automatiquement
acheminée : elle nécessite un ordre de transfert (`T`, voir ci-dessous) porté
par une armée. En hiver, le stock d'un moulin est conservé à
`ceil(stock / 2)`, comme celui d'un château ou d'un village, et n'est pas
rapatrié.

> À trancher dans #195 : le bénéficiaire adjacent doit-il être contrôlé par le
> même joueur que le moulin (recommandé) ; un moulin sur une case neutre
> produit-il sur sa propre case (recommandé).

### Amélioration d'un moulin

Le paiement d'une amélioration de moulin puise en priorité sur le stock
présent sur le moulin lui-même, puis sur le stock du village ou du château
adjacent (château en priorité si les deux sont adjacents) avant de recourir au
paiement d'hiver habituel. Le stock d'un moulin fait ainsi exception à la
règle selon laquelle seuls les châteaux et villages paient les
investissements d'hiver.

## Village fortifié

Issue : [#193](https://github.com/fogfactory/crown-and-borough/issues/193).

`C C XXX` sur un village contrôlé le **fortifie** pour le coût d'un château
(10 R) au lieu de le remplacer par un château. Un village ne peut plus être
remplacé par un château. Le village fortifié conserve son stock, sa
production et son bonus de revenu, et gagne le bonus défensif d'un château
(`castle_defense_bonus`), avec la même exception d'auto-capture. Un `C C` sur
un village déjà fortifié est rejeté sans prélèvement.

> À trancher dans #193 : un village fortifié compte-t-il comme un village
> partout sauf pour la défense (recommandé, via un indicateur sur le village)
> ou comme un nouveau type d'infrastructure ? Conséquences à fixer : score,
> désignation comme capitale (`E C`), droit d'être capitale de fief, voix
> d'évêché.

## Portée de ravitaillement

Un château, un village ou un dépôt de vivres contribue au ravitaillement
(ancre, portée de base, bonus de portée) du joueur qui **contrôle** sa case.
Une conquête récente hors fief est contrôlée dès qu'une armée s'y arrête et
garde donc immédiatement sa valeur logistique.

> À trancher dans #196, pour une case d'un fief contrôlée par un joueur mais
> occupée par une armée adverse : qui peut consommer le stock ou piller
> (recommandé : l'occupant peut piller mais pas consommer ; pour le
> contrôleur, la case bloque le flux) ; les investissements d'hiver y sont-ils
> autorisés (recommandé : non) ; un dépôt y garde-t-il son bonus de portée
> (recommandé : inutilisable par les deux joueurs tant que la case est
> occupée).

## Transfert de ressources

Deux ordres dédiés permettent de transférer des ressources entre joueurs.

### Tours d'action

`XXX T YYY N` transfère `N` ressources depuis le stock de la case occupée par
l'armée émettrice vers `YYY`.

- `XXX` est la position de l'armée au tour d'exécution ; la chaîne peut contenir
  plusieurs transferts comme n'importe quels autres ordres.
- `YYY` doit être un château, un village ou la case d'une armée contrôlée par un
  autre joueur vivant. Un dépôt sans armée n'est pas une destination valide.
- Le stock source ne nécessite pas de château ni de village. Toute case
  contrôlée peut conserver un cache pendant un tour d'action.
- Le transfert suit le réseau de ravitaillement du donneur : portée de base de
  trois cases, bonus des dépôts contrôlés et blocage par les armées adverses.
  L'armée adverse du destinataire bloque également lorsqu'elle se trouve sur
  une case intermédiaire ; elle est autorisée sur la case destination.
- Le réseau peut être estimé à la soumission et dans l'overlay, puis est
  recalculé sur la position de début du tour lorsque l'ordre est exécuté. Une
  route bloquée rend l'ordre invalide.
- Une armée affamée ne peut pas transférer. Elle ne transporte pas plus que sa
  consommation brute : `2^(N - 1)` ressources pour une armée de `N` troupes,
  sans déduire les rations locales.
- L'ordre est traité après le ravitaillement. L'armée qui le porte ne réalise
  aucun autre ordre pendant ce tour.
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

- `XXX` doit être un château ou un village contrôlé par le donneur ; le débit
  suit les règles habituelles des paiements d'hiver.
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
