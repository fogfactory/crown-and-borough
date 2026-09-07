# Économie et prospérité

**Milestone lié :** [Économie & Prospérité](https://github.com/fogfactory/crown-and-borough/milestone/8)

**Dépend de :** [Ravitaillement](ravitaillement.md) ; les règles qui parlent de
seigneurie dépendent aussi de [Titres](titres.md).

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

Si un lieu-dit perd plus de cinq rations pendant l'hiver, un village est fondé
dans la même seigneurie. Si aucune case libre ne permet cette fondation :

1. un dépôt de vivres est amélioré en village ;
2. à défaut, un moulin est amélioré en village ;
3. si toutes les cases sont occupées, rien ne se passe.

La règle devra préciser la définition de « perd », le choix du lieu, la
compatibilité avec les infrastructures existantes et le contrôle de la nouvelle
structure.
