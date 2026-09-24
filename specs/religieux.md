# Religieux

**Milestone lié :** [Religieux](https://github.com/fogfactory/crown-and-borough/milestone/5)

**Dépend de :** [Titres & Victoire](titres.md) pour les lieux et les points,
et de [Cartographie](cartographie.md) pour le découpage des évêchés ; le flux
de ressource de la dîme dépend de [Économie et prospérité](economie.md).

## Évêchés et évêques

La carte est divisée en `N + 1` évêchés de taille approximativement égale. La
partition régionale déterministe déjà générée pour les calamités et les cartes
bonus (`internal/engine/mapgen/regions.go`, une région par village neutre
seed, couverture totale et connexité garanties) est directement réutilisable
comme découpage d'évêchés : même cardinalité `N + 1`, mêmes garanties de
connexité. Un territoire appartient à exactement une région/évêché,
indépendamment de son appartenance ou non à un fief séculier — les deux
découpages sont disjoints dans leur origine (fief : dynamique, acheté ;
évêché : statique, fixé à la génération de la carte).

Quand tous les lieux-dits d'un évêché sont contrôlés, une élection d'évêque est
organisée :

- 1 voix par lieu-dit contrôlé dans l'évêché ;
- 1 voix par évêque ;
- 2 voix par cardinal ;
- 3 voix pour le pape ;
- majorité relative.

La spécification devra définir les cas d'égalité, la conservation du contrôle et
la durée du mandat.

> À trancher : « tous les lieux-dits contrôlés » utilisait le sens positionnel
> unique du socle actuel. Avec la distinction occupé/contrôlé introduite par
> [titres.md](titres.md), préciser si l'élection exige que chaque lieu-dit de
> l'évêché soit contrôlé (en fief) par des joueurs, simplement occupé, ou l'un
> ou l'autre.

## Cardinaux et pape

- lorsque deux cardinaux sont présents, un pape peut être élu parmi eux à la
  majorité absolue ;
- il ne peut y avoir que `N - 1` cardinaux ;
- le rôle, le contrôle et les voix d'un cardinal ou du pape doivent être
  distingués des titres séculiers.

Les cartes de nomination sont planifiées dans
[`ordres-speciaux.md`](ordres-speciaux.md), mais leur résolution est définie ici.

## Dîme

Une carte de dîme (jouée depuis le deck d'ordres spéciaux, voir
[ordres-speciaux.md](ordres-speciaux.md)) détourne la production des moulins
d'un évêché vers la capitale du joueur qui la joue, au lieu du village ou du
château normalement bénéficiaire (voir le flux de ressource dans
[economie.md](economie.md)). Elle ne touche jamais le revenu de territoire des
fiefs, qui relève exclusivement de la taxe seigneuriale.

- l'**évêque** peut poser une dîme sur son propre évêché ;
- le **cardinal** peut la poser sur n'importe quel évêché, mais seulement celui
  qui n'est pas déjà tenu par son évêque ce tour-là (priorité au titulaire
  local, symétrique à la règle roi/seigneur de [titres.md](titres.md)) ;
- le **pape** peut la poser sur n'importe quel évêché, avec la même priorité
  au titulaire local (évêque, ou cardinal s'il a déjà posé une dîme ce tour).

La règle devra préciser le cas où plusieurs cardinaux ciblent le même évêché le
même tour, et si un évêché sans évêque élu reste taxable par un cardinal ou le
pape.
