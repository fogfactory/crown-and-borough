# Test live — Hotseat (session 1)

Notes prises pendant une partie en hotseat, au fil des observations.

> **Traçabilité :** ce test live a été réalisé après la finalisation du prompt
> P1.7, « Rapport de tour, cycle des saisons et boucle hotseat ». Ces notes
> recensent les écarts et améliorations observés après cette étape.

## Interaction

- Clic gauche sur un territoire : le sélectionne.
- Re-clic sur le territoire sélectionné : le désélectionne.
- Clic gauche en dehors de la carte : supprime la sélection.
- Déplacement de la carte : clic gauche maintenu puis glissé au-delà du seuil
  anti-drag, sans sélection du territoire sous le pointeur.

## Rendu

- Le cadre de sélection d'un territoire ne devrait pas masquer les frontières : cela empêche de voir si elles sont franchissables ou non.
- Rendu de la fraîcheur de l'information actuel (voile noir opaque par-dessus le territoire, `MapViewer.tsx` couche « Données anciennes ») : **pas satisfaisant** — il assombrit la couleur du terrain et nuit à la lisibilité de la nature du terrain.
  - Solution retenue : **pois de taille variable** (plutôt que des hachures) — des pois plus ou moins gros superposés au territoire (SVG pattern), sans changer la teinte du terrain :
    - plus les pois sont **gros**, moins l'information est fraîche ;
    - taille maximale atteinte à **3 tours** de retard — l'information est alors considérée **non fiable**.
  - À compléter éventuellement plus tard par un badge d'âge (« T-2 » / « il y a N tours », déjà prévu en P3.4) et/ou des frontières estompées.
- Rendu du contrôle territorial actuel (remplissage teinté de la couleur du joueur à 13 % d'opacité, `MapViewer.tsx` couche « Contrôle territorial ») : **pas satisfaisant** — mêmes raisons que la fraîcheur : cela modifie la couleur du terrain.
  - Solution retenue : **contour coloré** — le contour du territoire est dessiné avec la couleur du joueur (trait épais), le remplissage du terrain reste intact.
    - Le contour doit être un **liséré à l'intérieur du territoire** : il ne doit pas masquer la frontière (qui porte l'information franchissable/non franchissable), ni produire un effet bizarre à la frontière entre deux joueurs.

## Chaînes d'ordres

- Une chaîne d'ordres invalide **pour cause de non-adjacence** ne devrait pas invalider toute la chaîne : seul l'ordre invalide et les ordres **postérieurs** à celui-ci devraient être invalidés (les ordres antérieurs restent valides).
- Quand on sélectionne un territoire contenant une armée, on devrait pouvoir voir **l'ensemble des ordres empilés** (pas seulement l'ordre courant, aussi les suivants).
- En sélectionnant un territoire, on devrait voir également les **noms des nobles présents**.

## Panneau latéral

- Le **poste de commandement** et le **rapport de tour** devraient être dans **deux onglets partageant la même zone** (pas l'un au-dessus de l'autre en colonne).

## Phase hivernale

- Les ordres d'hiver sont très différents par nature (investissements, pas de chaînes, pas de combat) : il faut que la phase soit évidente **au premier coup d'œil** dans l'interface.
  - Solutions retenues :
    - **Panneau d'ordres distinctif** : thème hivernal propre (fond bleu glace, icône flocon, bouton « Soumettre les ordres d'hiver »…).
    - **Teinte de la carte** : voile neige (blanc/bleu très léger) ou palette saisonnière sur la carte entière en hiver.

## Économie

- Les **villages neutres** doivent produire des ressources et les **stocker** : prendre un village neutre revient donc à **récupérer ses ressources stockées**.

## Rapports

- Le **rapport d'hiver** doit préciser les **consommations de ressources pour chaque ordre réussi**.
- Chaque ligne d'ordre doit avoir une **pastille ou un marqueur coloré à la couleur du joueur**.
- La liste des **ordres exécutés** doit afficher :
  - le **label complet de l'ordre** (`XXX O YYY`) ;
  - une **pastille colorée** indiquant le noble (son trigramme, à la couleur du joueur) ;
  - son **résultat**.

## Panneau latéral

- La couleur du **joueur courant** doit aussi être affichée en **pastille sur le poste de commandement**.

## Identification des territoires

- Les messages d'erreur (ex. `Perdue · invalid_chain: chain validation failed: order O3: not_adjacent: A target "T24" is not adjacent to "T21"`) affichent le **matricule** du territoire au lieu du **trigramme** : il faudrait afficher le trigramme (ex. `ROS`). Depuis P1.9, la non-adjacence statique n'est plus rejetée à la réception : elle est reportée dans le rapport d'ordre à l'exécution (`non_adjacent_destination`, chaîne brisée) ; la réception affiche `Reçue`, pas `Perdue`.
- Plus généralement : le **matricule de territoire n'a aucun sens**, seul le **trigramme** devrait exister (remise en cause du modèle `TerritoryID` « T24 » face à `Code`/trigramme).
