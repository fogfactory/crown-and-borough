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

Le socle porte `2 x N + 1` villages neutres : `N + 1` chefs-lieux et un village
dédié par territoire de départ. Comme une case ne porte qu'une seule
infrastructure, les territoires de départ des joueurs sont choisis hors des
territoires marqués comme villages, séparés d'au moins quatre étapes dans le
graphe franchissable les uns des autres, et associés chacun à un village
dédié à exactement deux étapes de leur territoire de départ, à au moins trois
étapes de tout autre territoire de départ et à au moins deux étapes de tout
autre village. Les chefs-lieux restent séparés d'au moins trois étapes de tout
territoire de départ et d'au moins deux étapes de tout village dédié. Les
territoires de départ et les villages qui satisfont conjointement toutes ces
contraintes sont choisis par une recherche avec retour arrière conjointe sur
les deux ensembles : un territoire de départ n'est retenu que si un village
dédié compatible existe pour lui, et réciproquement.

## Carte élargie

La carte du socle contient `8 x N` territoires de jeu et `(N + 1) x 4`
territoires supplémentaires, et porte `2 x N + 1` villages au total : `N + 1`
chefs-lieux et un village dédié par territoire de départ. Le placement reste
déterministe, connexe et compatible avec les contraintes de degré et de
terrain existantes. Le graphe franchissable est également sans point
d'articulation : chaque paire de territoires livrés, notamment chaque paire
de lieux-dits, reste reliée par deux chemins sans territoire intermédiaire
commun. La génération retient une probabilité de 50 % pour la suppression des
frontières difficiles (montagne/montagne et montagne/marécage), sans accepter
une suppression qui introduirait un point d'articulation.

Un chef-lieu et un village dédié sont des infrastructures identiques en jeu ;
seul le chef-lieu identifie la seed d'une région dans `regions[].seed`. Les
chefs-lieux sont placés par un algorithme de propagation maximin sur la
distance des centroïdes parmi les sites restés éligibles, pour maximiser leur
étalement sur la carte.

Les lacs et la mer sont conservés comme possibilités de conception, mais ne font
pas partie du périmètre minimal des rivières et des ponts.

## Lien religieux

Le découpage en `N + 1` évêchés de taille comparable sera défini dans
[`religieux.md`](religieux.md), en réutilisant les limites et les lieux-dits de
la carte.
