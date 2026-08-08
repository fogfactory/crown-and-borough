# Religieux

**Milestone lié :** [Religieux](https://github.com/fogfactory/crown-and-borough/milestone/5)

**Dépend de :** [Titres & Victoire](titres.md) pour les lieux et les points,
et de [Cartographie](cartographie.md) pour le découpage des évêchés.

## Évêchés et évêques

La carte est divisée en `N + 1` évêchés de taille approximativement égale. Quand
tous les lieux-dits d'un évêché sont contrôlés, une élection d'évêque est
organisée :

- 1 voix par lieu-dit contrôlé dans l'évêché ;
- 1 voix par évêque ;
- 2 voix par cardinal ;
- 3 voix pour le pape ;
- majorité relative.

La spécification devra définir les cas d'égalité, la conservation du contrôle et
la durée du mandat.

## Cardinaux et pape

- lorsque deux cardinaux sont présents, un pape peut être élu parmi eux à la
  majorité absolue ;
- il ne peut y avoir que `N - 1` cardinaux ;
- le rôle, le contrôle et les voix d'un cardinal ou du pape doivent être
  distingués des titres séculiers.

Les cartes de nomination sont planifiées dans
[`ordres-speciaux.md`](ordres-speciaux.md), mais leur résolution est définie ici.
