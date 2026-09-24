# Économie et prospérité

**Milestone lié :** [Économie & Prospérité](https://github.com/fogfactory/crown-and-borough/milestone/8)

**Dépend de :** [Ravitaillement](ravitaillement.md) et
[Titres](titres.md) pour la distinction occupé/contrôlé et les fiefs ; la
dîme religieuse dépend de [Religieux](religieux.md).

## Objectif de la refonte

L'économie v1 incite à entourer un unique château de moulins au niveau
maximal : la production dépend uniquement de l'infrastructure bâtie, jamais du
territoire lui-même. La refonte inverse cette incitation en faisant du
territoire contrôlé la source de revenu principale, et en réduisant la
capacité d'une case isolée à nourrir une grosse armée sur place.

## Rations de terrain

Remplace la table de rations locales de `gdd.md` §3 et §5. Objectif : réduire
le plafond d'armée soutenable sans logistique, et faire du siège un vrai
levier d'affamement plutôt qu'un détail négligeable.

| Terrain | Rations actuelles | Rations proposées |
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

## Flux de la ressource R

| Source | Bénéficiaire par défaut | Avec taxe |
|---|---|---|
| Territoire contrôlé (fief), sans village | 1 R → capitale du fief | Seigneur : 2 R → capitale du fief. Roi (si le seigneur ne taxe pas ce tour) : +1 R → capitale du roi, 1 R de base reste au fief |
| Territoire contrôlé (fief), avec village | 2 R → capitale du fief (1 territoire + 1 village) | Seigneur : 4 R → capitale du fief. Roi (si non taxé localement) : +2 R → capitale du roi, 2 R de base restent au fief |
| Territoire occupé, sans village | 1 R → capitale du joueur | — (pas de mécanique de taxe sur l'occupé) |
| Territoire occupé, avec village | 2 R → capitale du joueur | — |
| Moulin (niveau `N`) | `N` R → village/château adjacent s'il y en a un, sinon reste sur le moulin | Dîme religieuse jouée sur l'évêché : `N` R → capitale du joueur qui a joué la dîme, au lieu du village/château adjacent |

La taxe seigneuriale double toujours exactement le revenu de territoire,
village inclus ; elle ne touche jamais la production des moulins. La dîme
(voir [religieux.md](religieux.md)) fait l'inverse : elle ne touche que les
moulins d'un évêché, jamais le revenu de territoire. Les deux mécaniques
portent donc sur des flux disjoints et ne peuvent pas entrer en conflit sur le
même territoire le même tour.

Un moulin adjacent à la fois à un village et à un château verse sa production
au château par priorité. Une production restée sur un moulin isolé n'est pas
automatiquement acheminée : elle nécessite un ordre de transfert (`T`, voir
ci-dessous) porté par une armée.

### Amélioration d'un moulin

Le paiement d'une amélioration de moulin puise en priorité sur le stock
présent sur le moulin lui-même, puis sur le stock du village ou du château
adjacent (château en priorité si les deux sont adjacents) avant de recourir au
réseau de ravitaillement habituel.

## Village fortifié

Un village peut être amélioré en **village fortifié** pour le même coût qu'un
château (10 R). Contrairement à `C C` sur un village, qui remplace
l'infrastructure par un château (et conserve le stock de la case, règle
inchangée), l'amélioration en village fortifié ne détruit pas le village : il
conserve son statut et ses bonus de production propres, et gagne en plus le
bonus défensif d'un château (`+1` défense).

> À trancher : un village fortifié compte-t-il comme un village ou comme un
> château pour les règles qui distinguent les deux (score de fin de partie,
> comptage des voix d'évêché, remplacement par un château ultérieur) ? Une
> lecture cohérente avec « conserve son statut » est de le garder classé
> village partout sauf pour la défense.

## Portée de ravitaillement

Un château, un village ou un dépôt de vivres ne contribue au ravitaillement
(ancre, portée de base, bonus de portée) que s'il est **occupé ou contrôlé**
par le joueur qui l'utilise — pas seulement contrôlé. Sans cette précision,
toute conquête récente (occupée mais pas encore intégrée à un fief) perdrait
sa valeur logistique jusqu'à l'achat du fief, ce qui n'est pas l'intention.

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

Objectif : faire apparaître de nouveaux villages sans action directe d'un
joueur, tout en laissant sa gestion (occupation, sièges, fiefs) influencer où
et quand ça arrive. Le déclencheur reste un exode plutôt qu'un surplus : un
lieu-dit qui perd plus de `N` rations pendant l'hiver voit sa population fuir
et fonder un village ailleurs.

Piste retenue pour une première version — **exode élargi** :

1. le lieu-dit fuit vers la case libre la plus proche **non adjacente à un
   village ou un château existant**, en respectant l'ordre de priorité
   suivant : contrôlée par le joueur du lieu-dit d'origine (priorité au fief),
   sinon occupée par ce joueur, sinon n'importe quelle case libre restante, y
   compris neutre ou occupée par un autre joueur ;
2. si aucune case ne satisfait la contrainte d'adjacence à aucun niveau de
   priorité, la dégradation en cascade du brouillon initial s'applique : un
   dépôt de vivres est amélioré en village, à défaut un moulin est amélioré en
   village, sinon rien ne se passe.

La contrainte de non-adjacence évite qu'un nouveau village apparaisse collé à
une infrastructure existante ; elle s'applique à tous les niveaux de priorité
de l'étape 1, pas seulement au dernier.

> À trancher : le seuil `N = 5` a été calibré sur l'ancienne table de rations
> (plaine à 3, bonus château/village à +2). Avec la table réduite ci-dessus
> (plaine à 2, bonus supprimé), un déficit de 5 devient beaucoup plus facile à
> atteindre ; le seuil doit être recalibré une fois la nouvelle table en
> place — probablement par test plutôt que par calcul a priori.

Piste complémentaire, non retenue pour une première version mais à garder en
réserve si la carte reste trop statique en pratique : une croissance passive
et indépendante des joueurs, où un territoire neutre inoccupé et sans conflit
à proximité depuis plusieurs tours a une chance déterministe (seedée comme le
deck) de fonder un village chaque année. Contrairement à l'exode, cette piste
ne dépend d'aucune décision de joueur.

La règle devra préciser la définition exacte de « perd » (par rapport à la
production locale du lieu-dit, à sa consommation, ou aux deux), le choix
précis de la case en cas d'égalité de distance, et le contrôle de la nouvelle
structure (contrôlée si fondée sur un territoire de fief, occupée sinon).
