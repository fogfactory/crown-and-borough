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
terrain existantes. Le graphe franchissable compte au plus deux points
d'articulation (des verrous) ; hors d'eux, chaque paire de territoires
livrés, notamment chaque paire de lieux-dits, reste reliée par deux chemins
sans territoire intermédiaire commun. La génération retient une probabilité de
40 % pour la suppression des frontières difficiles (montagne/montagne et
montagne/marécage) et de 10 % pour les autres frontières non plain/plain, sans
accepter une suppression qui dépasserait deux points d'articulation. Le degré
franchissable maximal est de 3 en montagne, 4 en marécage et colline, 5 en
plaine et forêt.

Une frontière visible mesure au moins 24 px sur la carte de référence ; un
contact plus court est réduit à un point, sans frontière commune, et publié
dans `cornerContacts` pour que la carte y pose une icône de non-adjacence.

Un chef-lieu et un village dédié sont des infrastructures identiques en jeu ;
seul le chef-lieu identifie la seed d'une région dans `regions[].seed`. Les
chefs-lieux sont choisis parmi les sites restés éligibles en comparant de
nombreux placements : chaque évêché doit toucher le bord de la carte sur une
longueur suffisante, puis le moins possible de joueurs occupent seuls leur
évêché, puis l'écart minimal entre chefs-lieux est maximal. La croissance des
évêchés utilise un coût de franchissement bruité mais stable, pour que leurs
frontières s'imbriquent.

Les lacs et la mer sont conservés comme possibilités de conception, mais ne font
pas partie du périmètre minimal des rivières et des ponts.

## Lien religieux

Le découpage en `N + 1` évêchés de taille comparable sera défini dans
[`religieux.md`](religieux.md), en réutilisant les limites et les lieux-dits de
la carte.
