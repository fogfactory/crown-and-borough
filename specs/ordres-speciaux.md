# Ordres spéciaux et calamités

**Milestone lié :** [Ordres spéciaux & Calamités](https://github.com/fogfactory/crown-and-borough/milestone/4)

**Dépend de :** la boucle hiver et les rapports du socle actuel.

## Pioche et main

- chaque hiver, un joueur tire un ou deux ordres spéciaux selon la règle de la
  partie ;
- les cartes peuvent être conservées pour une résolution ultérieure ;
- le joueur peut abandonner des cartes existantes ;
- la main est limitée à quatre ordres spéciaux.

Chaque carte devra déclarer son coût, ses conditions, son moment d'utilisation,
son effet et les informations visibles par les autres joueurs.

## Calamités

Un tirage peut déclencher une calamité. Le nombre maximal de calamités est de
trois par année, même si plusieurs cartes sont tirées. La règle devra définir la
pioche, le renouvellement, les défausses et la résolution déterministe des
calamités.

## Syntaxe des ordres

Les ordres spéciaux sont des lignes de joueur distinctes des chaînes de nobles :

- `D C BT` ou `D C RA` abandonne une carte bonus ;
- `T C` tire une carte en hiver ;
- `P BT TER` / `J BT TER` joue Beau temps ;
- `P RA TER` / `J RA TER` joue Récolte abondante.

Les aliases français et anglais sont acceptés quelle que soit la langue de
l’interface. `TER` est obligatoirement le village seed d’une région. Les kinds
de calamité ne peuvent pas être joués comme ordres de joueur.

## Cartes prévues

Le deck pourra accueillir notamment des impôts, des mariages, des assassinats,
des nominations de cardinaux et des Claims. Les règles propres aux cartes de
succession, politique et religion restent dans leurs spécifications thématiques.
