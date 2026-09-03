# Ordres spéciaux et calamités

**Milestone lié :** [Ordres spéciaux & Calamités](https://github.com/fogfactory/crown-and-borough/milestone/4)

**Dépend de :** la boucle hiver, la résolution simultanée et les rapports du socle actuel.

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
calamités. L’augure du printemps révèle pour chaque calamité son kind, sa saison
et sa région ; les augures futures restent cachées.

## Syntaxe des ordres

Les ordres du deck sont soumis dans un champ `special` distinct des chaînes de
nobles et des investissements d'hiver. Aucun noble n'est requis :

- `D C BT` ou `D C RA` abandonne une carte bonus, en hiver uniquement ;
- `T C` tire une carte, en hiver uniquement ;
- `P BT TER` joue Beau temps au printemps, en été ou en automne ;
- `P RA TER` joue Récolte abondante au printemps, en été ou en automne ;
- `P RE TER` joue Révolte pendant ces saisons si une famine affecte la région cible.

Les cartes jouées sont consommées puis leurs effets sont appliqués avant le
ravitaillement et la résolution simultanée des ordres d'armée.

Les aliases français et anglais sont acceptés quelle que soit la langue de
l’interface. `TER` est obligatoirement le village seed d’une région. Les kinds
de calamité ne peuvent pas être joués comme ordres de joueur. Lorsqu’un ordre
`P` ou `J` est appliqué, il consomme la première carte du kind demandé dans la
main du joueur et la place dans la défausse ; l’effet est ensuite enregistré pour
une résolution simultanée par région.

## Cartes prévues

Le deck pourra accueillir notamment des impôts, des mariages, des assassinats,
des nominations de cardinaux et des Claims. Les règles propres aux cartes de
succession, politique et religion restent dans leurs spécifications thématiques.
