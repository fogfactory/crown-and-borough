# Cartographie

**Milestone lié :** [Cartographie](https://github.com/fogfactory/crown-and-borough/milestone/3)

**Dépend de :** la génération déterministe et le graphe de territoires du socle
actuel. Ce thème peut être développé en parallèle des règles politiques.

## Rivières et ponts

Toute suite de frontières infranchissables reliée à un bord de la carte devient
une rivière. Une rivière bloque le déplacement tant qu'aucun pont n'a été créé.
Un pont est une infrastructure ou un aménagement qui rétablit la connectivité
sur une frontière choisie.

L'issue devra préciser la construction, le coût, la durée de vie, le contrôle et
la représentation graphique des ponts.

## Villages neutres du socle

Le socle conserve `N + 1` villages neutres en plus des `N` châteaux de départ.
Comme une case ne porte qu'une seule infrastructure, les territoires de départ
des joueurs sont choisis hors des territoires marqués comme villages et sont
séparés d'au moins quatre étapes dans le graphe franchissable.

## Carte élargie

La carte du socle contient `8 x N` territoires de jeu et `(N + 1) x 4`
territoires supplémentaires dédiés aux villages neutres. Elle porte toujours
`N + 1` villages, placés uniquement dans cette plage dédiée. Le placement reste
déterministe, connexe et compatible avec les contraintes de degré et de terrain
existantes.

Les lacs et la mer sont conservés comme possibilités de conception, mais ne font
pas partie du périmètre minimal des rivières et des ponts.

## Lien religieux

Le découpage en `N + 1` évêchés de taille comparable sera défini dans
[`religieux.md`](religieux.md), en réutilisant les limites et les lieux-dits de
la carte.
