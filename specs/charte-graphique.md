# Charte graphique

**Suivi :** une issue par phase, à créer (voir [Phasage](#phasage)).

**Dépend de :** [architecture.md](architecture.md) pour la stack front et les
contrats `map.json` / `state.json` ; [titres.md](titres.md#contrôle-et-occupation)
pour les notions de territoire contrôlé et occupé ;
[cartographie.md](cartographie.md) pour les futures rivières.

Les décisions ci-dessous ont été actées lors de l'audit graphique du
27 septembre 2026. Elles ne sont pas encore implémentées : l'interface actuelle
reste la référence tant que la phase correspondante n'est pas livrée. Les
paragraphes marqués **Précision** traduisent une décision en règle applicable ;
ils peuvent être amendés en revue sans remettre en cause le choix qu'ils
servent.

## Objectifs

- Améliorer l'ergonomie : une carte lisible d'un coup d'œil, des actions
  toujours accessibles, un écran de téléphone utilisable dès l'ouverture.
- Donner au jeu une saveur médiévale de carte ancienne, cohérente de la carte
  aux panneaux et jusqu'à l'emblème.

## Principes

1. **La couleur vive appartient aux joueurs.** Les teintures héraldiques ne
   disent qu'une chose : « à qui ». Elles colorent les bannières, les écus, les
   pions, les bâtiments et les intentions. L'interface n'en emprunte aucune.
2. **L'encre décrit le monde, le laiton désigne l'action.** Le terrain, les
   frontières et le texte sont à l'encre sépia. Le bouton principal, l'onglet
   actif, le focus et la sélection sont en laiton.
3. **Une information, un canal visuel.** Contrôle : bannière. Occupation : écu
   de l'armée présente. Évêché : cadre et liseré. Terrain : symboles d'encre.
   Saison : voile de la carte. Deux informations ne partagent jamais un canal.
4. **Jamais la couleur seule.** Toute information portée par une couleur a un
   second repère : pièce héraldique, symbole de terrain, trait de frontière,
   forme ou texte.
5. **Des tokens, pas de valeurs en dur.** Couleurs, polices, tailles et rayons
   passent par les tokens du thème (`web/src/index.css`). Les composants
   n'écrivent aucune valeur hexadécimale ; les données de rendu de la carte
   (teintures, symboles) sont centralisées dans un module dédié.

## Décisions

| # | Sujet | Retenu | Écarté |
|---|---|---|---|
| 1 | Direction artistique | Carte de cartographe : gravure, encre sépia, noms de lieux en italique | Plateau imprimé et parchemin, manuscrit enluminé, plateau physique |
| 2 | Matière | Parchemin clair rationalisé | Table de bois, salle du conseil sombre, double thème |
| 3 | Typographie | Cinzel, Alegreya, IBM Plex Mono | Garamond et humaniste, gothique, imprimé ancien |
| 4 | Couleur d'action | Laiton ; vermillon réservé aux erreurs ; aucun joueur en or | Encre seule, vert-de-gris, rouge sceau |
| 5 | Contrôle | Bannière sur chaque territoire contrôlé ; l'occupation se lit à l'armée présente | Lavis, liseré intérieur, vues à bascule |
| 6 | Terrain | Carte gravée : symboles d'encre sur un papier commun | Aquarelle, aplats pastel, tuiles de plateau |
| 7 | Frontières infranchissables | Série d'icônes choisie selon les terrains adjacents : montagnes, escarpement, marais dense, rivière | Lignes stylisées (crête hachurée, rempart, fleuve dessiné en trait) |
| 8 | Évêchés | Affichage actuel restylé : cadre, liseré et calque | Tiret-point et blason |
| 9 | Armées | Écu chiffré | Bloc de régiment, meeple, pile de jetons |
| 10 | Nobles | Pion simple, enrichi d'attributs cumulatifs selon les titres | Sceau de cire, blason personnel, pastille |
| 11 | Identité des joueurs | Teinture et blason générés, puis blason choisi par le joueur (deuxième étape) | Hachures héraldiques, couleur seule |
| 12 | Bâtiments | Silhouettes actuelles recolorées | Tampons à l'encre, maquettes, jetons |
| 13 | Disposition sur ordinateur | Carte maximale et tiroir à signets | Disposition actuelle améliorée, trois colonnes, table de jeu |
| 14 | Saisie des ordres | Texte assisté ; cartes d'ordres possibles plus tard, le texte assisté restant disponible | Constructeur seul, texte seul |
| 15 | Saison | Ambiance seule : la teinte de la carte et de l'interface annonce la saison | Roue des saisons, piste du temps |
| 16 | Téléphone | Onglets plein écran ; l'onglet Ordres partage l'écran avec la carte, synchronisée avec la saisie | Panneau glissant corrigé, carte pivotée, paysage |
| 17 | Rapport de tour | Cartes d'événements ; la carte ne montre un événement qu'au clic | Liste nettoyée, chronique, rejeu animé |
| 18 | Emblème | Sceau de cire | Cartouche, blason écartelé, emblème actuel |
| 19 | Mouvement | Aucune animation | Micro-animations, sons, résolution animée |
| 20 | Phasage | Fondations d'abord | Carte, ergonomie ou écran complet d'abord |

## Couleurs

### Papiers, encres et filets

Cinq papiers et trois encres remplacent les nuances actuellement écrites en
dur dans les classes Tailwind.

| Token | Valeur | Usage | Remplace |
|---|---|---|---|
| `paper-50` | `#fbf7ee` | Surfaces : panneaux, cartes, menus, champs | `#fffaf0`, `#fff8e7`, `#fbf6ea` |
| `paper-100` | `#f3ead9` | Creux : piste d'onglets, lignes de liste, blocs de code | `#f3ead9`, `#f8f0e2` |
| `paper-200` | `#efe7d8` | Fond de page | `#efe7d8` |
| `paper-300` | `#e6d8bb` | Pourtour de la carte, séparateurs pleins | `#e6d8bb` |
| `paper-map` | `#f1e6cb` | Papier de la carte gravée | aplats de terrain |
| `ink-900` | `#30291f` | Texte principal, traits et symboles de la carte | `#30291f`, `#17120f` |
| `ink-700` | `#594b3c` | Texte secondaire | `#594b3c` |
| `ink-500` | `#74644d` | Texte atténué | `#806f57`, sous le seuil AA sur `paper-100` |
| `line-strong` | `#8f7d5c` | Bordure des champs et des boutons secondaires | — |
| `line` | `#b7a786` | Bordure des panneaux | `#b7a786` |
| `line-soft` | `#d9cfba` | Séparateurs discrets | `#b7a786` à 50 % |

### Laiton et couleurs d'état

| Token | Valeur | Usage |
|---|---|---|
| `brass-300` | `#e2c47a` | Filets décoratifs : cadre de la carte, cartouches |
| `brass-400` | `#c9a24a` | Fond du bouton principal, halo de sélection, couronnes des nobles titrés |
| `brass-500` | `#b8923f` | Survol du bouton principal |
| `brass-700` | `#7d5a1c` | Texte en laiton (onglet actif, liens), anneau de focus, trait de sélection |
| `vermilion-700` | `#a4301c` | Texte et icône d'erreur |
| `vermilion-100` | `#f6e1d6` | Fond d'une alerte d'erreur |
| `warning-700` | `#8a5216` | Diagnostic non bloquant (hiver) |
| `warning-100` | `#fbeedd` | Fond d'un diagnostic |
| `success-700` | `#376341` | Validation, statut « libre » |
| `winter-700` | `#2c5b7d` | Titre et texte du panneau d'hiver |
| `winter-100` | `#eaf1f7` | Fond du panneau d'hiver |

Le vermillon n'apparaît jamais sur la carte : il reste dans les panneaux, là
où il ne peut pas être confondu avec le gueules d'un joueur.

### Correspondance avec shadcn/ui

Les composants shadcn lisent ces variables ; aucune classe ne réécrit leurs
couleurs au cas par cas.

| Variable | Token |
|---|---|
| `--background` | `paper-200` |
| `--foreground` | `ink-900` |
| `--card`, `--popover` | `paper-50` |
| `--card-foreground`, `--popover-foreground` | `ink-900` |
| `--primary` | `brass-400` |
| `--primary-foreground` | `ink-900` |
| `--secondary`, `--muted`, `--accent` | `paper-100` |
| `--secondary-foreground`, `--accent-foreground` | `ink-900` |
| `--muted-foreground` | `ink-500` |
| `--destructive` | `vermilion-700` |
| `--border` | `line` |
| `--input` | `line-strong` |
| `--ring` | `brass-700` |
| `--radius` | `0.25rem` |

Le thème sombre n'est pas retenu : le bloc `.dark` de `web/src/index.css`
disparaît avec la phase 3.

### Joueurs

Chaque joueur reçoit une teinture et des armoiries simples : un champ de sa
teinture et une pièce en métal. Le champ donne la couleur du joueur sur la
carte ; la pièce le distingue encore en niveaux de gris et pour un daltonien.

| Siège | Teinture | Valeur | Métal | Pièce | Blasonnement |
|---|---|---|---|---|---|
| P1 | Gueules | `#b3322b` | Or | Bande | De gueules à la bande d'or |
| P2 | Azur | `#2b5c9e` | Argent | Fasce | D'azur à la fasce d'argent |
| P3 | Sinople | `#3f7d3c` | Or | Chevron | De sinople au chevron d'or |
| P4 | Pourpre | `#6e3f8f` | Argent | Croix | De pourpre à la croix d'argent |
| P5 | Sable | `#2f2b27` | Or | Sautoir | De sable au sautoir d'or |
| P6 | Tenné | `#c8651b` | Argent | Pal | De tenné au pal d'argent |
| P7 | Céleste | `#4fa3c7` | Or | Chef | De céleste au chef d'or |
| P8 | Mûre | `#8c2f5a` | Argent | Bordure | De mûre à la bordure d'argent |

- Métaux : or `#d8ab35`, argent `#f1ede3`. L'or n'est jamais une teinture de
  joueur ; il n'apparaît qu'en métal, dans une pièce.
- Au-delà de huit joueurs (le hotseat va jusqu'à seize), le siège `P(n)`
  reprend la teinture de `P(n − 8)`, inverse le métal et décale la pièce de
  quatre rangs : P9 porte de gueules au sautoir d'argent, P10 d'azur au pal
  d'or, et ainsi de suite.
- Les rebelles (`NEUTRAL`) sont cendrés (`#7d766a`), sur un écu plein sans
  pièce.
- L'ordre d'attribution suit l'ordre des sièges, comme la palette actuelle
  d'`internal/engine/game.go`.
- **Deux étapes.** D'abord, chaque joueur reçoit les armoiries générées
  ci-dessus. Ensuite, un joueur pourra choisir son propre blason (champ,
  pièce, meuble) dans son profil ; ce choix se superpose aux armoiries
  générées et reste soumis aux mêmes règles (teinture en champ, métal en
  pièce, pas d'or en teinture, pas de doublon dans une partie).

### Évêchés

Six teintes terreuses, hors de la palette des joueurs, remplacent les couleurs
Tailwind de `web/src/lib/region-color.ts`. Au-delà de six évêchés, les motifs
existants (`REGION_PATTERNS`) s'ajoutent, dessinés à l'encre à 30 %.

| Ocre | Ardoise | Olive | Brique | Lavande | Sauge |
|---|---|---|---|---|---|
| `#a9803a` | `#5f7d8c` | `#7b7d47` | `#9a5f4b` | `#7f7496` | `#6b8a74` |

Un évêché ne remplit jamais l'intérieur d'un territoire, et un joueur ne
dessine jamais de liseré : c'est ce qui évite de confondre les deux.

### Saisons

| Saison | Teinte d'interface | Voile de la carte |
|---|---|---|
| Printemps | `#9cc36b` | `#9cc36b` à 14 % |
| Été | `#e3b341` | `#e3b341` à 14 % |
| Automne | `#c0703a` | `#c0703a` à 17 % |
| Hiver | `#bcd3e6` | `#e3edf7` à 42 %, avec la neige actuelle |

La teinte d'interface est décorative : elle n'est jamais le fond d'un texte
et ne remplace pas le laiton des actions.

### Contrastes

Le texte vise au moins 4,5:1 (WCAG AA) ; les contours de composants et les
objets graphiques de la carte au moins 3:1. Paires vérifiées :

| Paire | Rapport |
|---|---:|
| `ink-900` sur `paper-50` | 13,4:1 |
| `ink-900` sur `paper-map` | 11,6:1 |
| `ink-700` sur `paper-300` | 6,0:1 |
| `ink-500` sur `paper-50` | 5,4:1 |
| `ink-500` sur `paper-100` | 4,8:1 |
| `ink-500` sur `paper-200` | 4,7:1 |
| `ink-900` sur `brass-400` (bouton principal) | 6,0:1 |
| `ink-900` sur `brass-500` (survol) | 4,9:1 |
| `brass-700` sur `paper-50` | 5,9:1 |
| `brass-700` sur `paper-100` | 5,3:1 |
| `brass-700` sur `paper-map` (trait de sélection) | 5,1:1 |
| `line-strong` sur `paper-50` (bordure de champ) | 3,7:1 |
| `vermilion-700` sur `vermilion-100` | 5,5:1 |
| `success-700` sur `paper-100` | 5,8:1 |
| `warning-700` sur `warning-100` | 5,6:1 |
| `winter-700` sur `winter-100` | 6,4:1 |

`ink-500` ne s'emploie pas sur `paper-300` (4,1:1) : on y écrit en `ink-700`.
`brass-400` n'atteint que 1,9:1 sur `paper-map` ; c'est pourquoi la sélection
double son halo d'un trait `brass-700`. `line` (2,2:1) reste réservé aux
panneaux, jamais aux champs.

## Typographie

### Familles

| Rôle | Police | Graisses | Paquet |
|---|---|---|---|
| Titres, étiquettes en capitales, codes de territoire, noms d'évêché | Cinzel | 500, 700 | `@fontsource-variable/cinzel` |
| Texte, interface, noms de lieux (italique), chiffres | Alegreya | 400, 400 italique, 500, 700 | `@fontsource-variable/alegreya` |
| Ordres : saisie, cartes d'ordres, extraits cités | IBM Plex Mono | 400, 500 | `@fontsource/ibm-plex-mono` |

Les polices sont embarquées avec Fontsource, sans requête vers Google Fonts ;
toutes trois sont sous licence SIL Open Font License. Geist
(`@fontsource-variable/geist`) est retiré, et `font-serif` (police système)
laisse la place à un token `font-display`.

### Échelle

| Token | Taille et interligne | Usage |
|---|---|---|
| `text-2xs` | 11 / 14 px | Étiquettes en capitales Cinzel ; jamais plus petit pour Cinzel |
| `text-xs` | 12 / 16 px | Métadonnées, légende |
| `text-sm` | 14 / 20 px | Interface : boutons, onglets, listes, champs |
| `text-base` | 16 / 24 px | Texte courant : règles, FAQ, rapport |
| `text-lg` | 18 / 24 px | Titres de section du panneau (Cinzel 500) |
| `text-xl` | 22 / 28 px | Nom du territoire sélectionné (Cinzel 700) |
| `text-2xl` | 28 / 34 px | Titres de page |
| `text-3xl` | 36 / 42 px | Accueil et connexion |

Ces tokens remplacent l'échelle par défaut de Tailwind, dont ils reprennent
les noms.

### Règles

- Cinzel s'emploie en capitales espacées de 0,08 em pour les étiquettes.
- Scores, compteurs et coûts utilisent des chiffres alignés
  (`font-variant-numeric: lining-nums tabular-nums`).
- Tout texte d'ordre est en IBM Plex Mono : zone de saisie, carte d'ordre,
  ligne citée dans une erreur.
- Les libellés d'interface sont en casse de phrase ; seules les étiquettes
  Cinzel sont en capitales.

## Carte

Les dimensions ci-dessous sont en unités de carte, multipliées par
`annotationScale` comme les annotations actuelles ; les épaisseurs de trait
notées en px ne varient pas avec le zoom (`vector-effect: non-scaling-stroke`).

### Ordre des calques

De bas en haut :

1. Papier (`paper-map`), qui remplace les aplats `TERRAIN_COLORS`
   (`web/src/components/MapLegend.tsx`).
2. Terrain gravé, qui remplace les motifs de `MapDecorations.tsx`.
3. Évêchés : cadre et liseré (`MapRegionLayers.tsx`).
4. Voile de saison, et neige en hiver.
5. Zone de ravitaillement.
6. Frontières franchissables, frontières infranchissables, contour extérieur.
7. Sélection, désormais au-dessus des frontières pour ne jamais être masquée.
8. Ligne de ravitaillement, calamités et cartes jouées.
9. Pièces : bâtiments, bannières, armées, nobles.
10. Étiquettes : noms, codes, chefs-lieux.
11. Intentions, ordres d'hiver et événement du rapport sélectionné.

### Terrain gravé

Tous les territoires partagent le même papier ; le terrain se lit à ses
symboles, dessinés à l'encre `ink-900` et éclairés en haut à gauche comme sur
une gravure.

| Terrain | Symbole | Pas de dispersion |
|---|---|---:|
| Forêt | Arbre rond sur un fût, moitié droite ombrée à 35 % | 12,5 |
| Colline | Taupinière : arc et deux hachures | 17 |
| Montagne | Profil triangulaire, versant droit ombré à 40 % | 16 |
| Marécage | Trois traits d'eau et une touffe de roseaux | 15 |
| Plaine | Touffe d'herbe clairsemée, à 70 % d'opacité | 23 |

**Précision.** La dispersion est déterministe, semée par le trigramme du
territoire avec le même générateur que `web/src/lib/chaotic-icons.ts`. Les
symboles restent à 6,5 unités des frontières et hors d'une ellipse de
30 × 27 unités autour du centroïde, réservée aux étiquettes et aux pièces.
Chaque symbole est défini une fois (`<symbol>`) et instancié par `<use>` pour
limiter la taille du DOM.

### Frontières

- **Franchissable :** pointillé rond `ink-900` à 75 %, 1,7 px, comme
  aujourd'hui.
- **Infranchissable :** une série d'icônes posées le long de la frontière,
  comme la chaîne de pics actuelle, jamais une ligne stylisée. L'icône dépend
  des deux territoires qu'elle sépare, dans cet ordre :
  1. si la frontière appartient à une rivière (voir ci-dessous), **rivière** :
     tronçons d'eau ondulés bleus (`#3d76a6`) ;
  2. sinon, si l'un des territoires est un marécage, **marais dense** :
     touffes de roseaux serrées sur des traits d'eau ;
  3. sinon, si l'un est une montagne, **montagnes** : la chaîne de pics
     actuelle (glyphe `peaks`), redessinée à l'encre `ink-900` ;
  4. sinon, **escarpement** : petits blocs de falaise en profil, tournés vers
     le territoire le plus bas (montagne > colline > forêt > plaine >
     marécage ; à égalité, vers le trigramme le plus grand). Si le dessin
     reste illisible à la taille des icônes, la chaîne de pics le remplace.
- **Contour extérieur :** `ink-900`, 2,2 px.

**Précision.** Les icônes gardent l'espacement, la taille et la rotation de
l'actuelle `ImpassableBorderChain` (`web/src/components/MapIconLayers.tsx`),
avec un halo `paper-map` pour rester lisibles sur les symboles de terrain.
Toutes sont monochromes à l'encre, sauf la rivière. Les rivières de
[cartographie.md](cartographie.md) (suites de frontières infranchissables
reliées à un bord) n'existent pas encore ; en attendant, la règle commence au
marais.

### Évêchés

L'affichage actuel est conservé et restylé :

- **Cadre :** les bandes autour de la carte (géométrie actuelle de
  `useRegionLayout`) prennent la teinte de leur évêché, bordées d'un filet
  `ink-900` et d'un filet intérieur `brass-300`. Le nom figure dans un
  cartouche `paper-50` à filet `ink-900`, en Cinzel 700 et capitales espacées ;
  il remplace le libellé « Évêché de … (XXX) » écrit dans la bande.
- **Liseré :** une bande de 10 unités longe l'intérieur de la limite de
  l'évêché, dans sa teinte à 50 %, découpée par l'évêché comme les anneaux
  actuels, sans halo lumineux.
- **Chef-lieu :** son nom s'écrit dans la teinte de l'évêché assombrie, avec
  un contraste d'au moins 4,5:1 sur `paper-map`.
- **Calque :** activable depuis la légende comme aujourd'hui
  (`cb.regionsOverlay`), affiché par défaut.

### Contrôle : bannières

- Tout territoire contrôlé (champ `owner` de `state.json`, qu'il appartienne
  ou non à un fief) porte une bannière à queue d'aronde aux armes de son
  joueur : champ de la teinture, pièce du métal.
- Aucun marqueur d'occupation : l'écu de l'armée présente suffit, puisqu'une
  case ne porte qu'une armée.
- Un territoire neutre n'a pas de bannière.
- La bannière remplace l'écu de contrôle (`OwnershipBadge`) ; le bouton de
  légende « Contrôle joueur » l'affiche ou la masque.

**Précision.** La bannière se plante à gauche du nom, le pied du mât sur sa
ligne de base ; mât `ink-900`, drapeau de 12 × 8 unités.

### Armées : écu chiffré

- L'armée est un écu de 15 × 19 unités aux armes de son joueur, avec son
  effectif au centre en Alegreya 700 blanc cerné d'encre
  (`paint-order: stroke`), lisible sur la teinture comme sur le métal.
- Une armée rebelle porte l'écu cendré plein.
- L'écu remplace le disque numéroté de `LiveLayer`
  (`web/src/components/MapTerritoryLayers.tsx`).

### Nobles : pion et attributs de titre

- Chaque noble est un pion simple de 11 × 17 unités (socle, corps, tête) à la
  teinture de son joueur.
- Les titres s'ajoutent comme attributs **cumulatifs** posés sur le pion :

  | Titre | Attribut |
  |---|---|
  | Baron | Couronne simple : cercle et trois perles |
  | Comte | Couronne à neuf perles |
  | Marquis | Couronne à fleurons et perles alternés |
  | Duc | Couronne à fleurons |
  | Roi | Couronne fermée surmontée d'une croix |
  | Évêque | Crosse tenue à côté du pion |
  | Cardinal | Robe rouge et chapeau de cardinal |
  | Pape | Robe blanche et tiare |

  Un noble porte la couronne de son plus haut titre séculier et y ajoute ses
  attributs religieux : un duc évêque porte la couronne de duc et la crosse.
  Les couronnes sont en `brass-400` cernées d'encre ; la robe du cardinal
  (`#a4231c`) et celle du pape (`#f1ede3`) remplacent la teinture du corps,
  qui reste visible sur le socle.
- Otage : pion grisé (cendré). Au cachot : pion grisé derrière des barreaux à
  l'encre. Les attributs restent visibles.
- Plusieurs nobles sur une case se placent côte à côte, espacés de 12 unités.
- Le pion remplace le losange de `NobleMarker`.

Les titres ne sont pas encore livrés dans `develop` : ceux des fiefs sont en
cours ([titres.md](titres.md)), les titres religieux ([religieux.md](religieux.md))
et royaux ([politique.md](politique.md)) viendront par itérations. Chaque
attribut arrive avec le titre qu'il représente ; tant qu'aucun titre n'existe,
tous les nobles sont des pions simples. La liste des titres séculiers suit
celle que les specs retiendront (le marquisat y est encore à trancher).

### Bâtiments

- Les silhouettes game-icons.net actuelles (`web/src/components/MapMarkers.tsx`)
  sont remplies de la teinture du joueur qui contrôle la case, ou de
  `paper-50` si la case est neutre ; cerne `ink-900`, halo `paper-map`.
- La couronne de la capitale passe en `brass-700` ; le niveau reste un chiffre
  en haut à droite, en Alegreya 700.

**Précision.** Le stock d'un château ou d'un village s'affiche « 10 R » à côté
du bâtiment, et la légende rappelle que R désigne les ressources ; le « ×10 »
isolé disparaît.

### Sélection, ravitaillement, intentions et calamités

- **Sélection :** halo `brass-400` de 8 unités à 30 % et trait `brass-700`
  de 2,5 px ; remplace l'orange `#d28b22`.
- **Ravitaillement :** zone en hachures `ink-900` à 35 % (au lieu de
  `#808080`) ; ligne en tirets à la teinture du joueur.
- **Intentions :** flèches à la teinture du joueur, cernées d'`ink-900` (au lieu
  de `#17120f`) ; un brouillon garde la même flèche à 45 % d'opacité, en
  tirets (au lieu de `#d4a39b`).
- **Calamités et cartes :** pictogrammes actuels à l'encre et au papier ; une
  calamité annulée est barrée d'un trait `ink-900`, pas de rouge.

### Étiquettes

- **Nom :** Alegreya italique 500, 12 unités, `ink-900`, halo `paper-map` de
  3 unités. Les noms de plus de 14 caractères se coupent au trait d'union
  (un seul sur 465 communes aujourd'hui : Fougères-du-Marais).
- **Code :** Cinzel 700, 8 unités, espacé de 0,08 em, sous le nom. Le
  trigramme reste visible, puisque les ordres s'écrivent avec lui.

**Précision.** Quand le nom rendu fait moins de 10 px à l'écran, seul le code
s'affiche. Les seuils sont à régler sur des cartes réelles : 52 territoires à
4 joueurs, près de 200 en hotseat à 16.

### Légende

La légende montre des échantillons de symboles au lieu de pastilles de couleur :
les cinq terrains, les icônes de frontière infranchissable et le pointillé
franchissable, la bannière, l'écu, le pion de noble (libre, otage, au cachot,
puis les attributs de titre à mesure qu'ils existent) et les bâtiments. Les boutons des calques (intentions, contrôle, évêchés, calamités,
cartes) sont conservés.

## Écrans et composants

### Disposition sur ordinateur

La carte occupe toute la largeur ; les panneaux passent dans un tiroir :

- **Barre du haut, fine et sur une ligne :** sceau et « Crown & Borough » en
  Cinzel ; saison, année et tour en texte (« Automne 1002 · tour 7 sur 40 ») ;
  scores ; pastilles de soumission ; joueur actif en hotseat ; navigation
  Règles et FAQ ; un menu « Partie » qui regroupe la création d'une partie
  (joueurs, graine, années), la résolution forcée et la langue.
- **Tiroir à droite :** posé sur la carte, avec des onglets en marque-pages
  (Ordres, Rapport, Règles) qui dépassent de son bord gauche. Il se replie
  pour rendre toute la carte ; le marque-page actif est en `brass-400`. Les
  onglets tiennent lieu de titre : le titre qui répète l'onglet actif
  (`GamePanelCard.tsx`) disparaît.
- **Action principale :** le bouton d'envoi des ordres vit dans un pied de
  tiroir collant, toujours visible. Libellé proposé : « Sceller les ordres ».

**Précision.** Le tiroir mesure 384 à 416 px comme la colonne actuelle ; la
carte se recadre quand il s'ouvre ou se replie. Le bandeau « Carte publique ·
vue commune » posé sur la carte disparaît ; l'aide aux gestes rejoint la
légende.

### Téléphone

- Navigation d'application par une barre d'onglets en bas : Carte, Ordres,
  Rapport, Règles. Chaque vue a tout l'écran ; le panneau glissant actuel
  (`web/src/components/ui/panel-sheet.tsx`) disparaît.
- **Carte :** plein écran, ouverte sur les territoires du joueur.
- **Ordres :** écran partagé, la carte en haut et la saisie en bas. La carte
  suit la saisie : taper ou compléter un trigramme la recentre sur ce
  territoire et le sélectionne ; toucher un territoire sur la carte insère
  son trigramme à la position du curseur.
- **Rapport :** toucher un événement bascule sur la carte, cadrée sur
  l'événement.
- L'en-tête tient sur une ligne : sceau, saison et tour, scores et un menu
  pour la langue et les réglages. Plus rien ne déborde.
- Toute cible tactile mesure au moins 44 × 44 px.

**Précision.** Dans l'onglet Ordres, la carte prend environ 40 % de la hauteur,
au-dessus du clavier virtuel ; la synchronisation ne se déclenche que sur un
trigramme complet et valide, pour ne pas faire sauter la carte à chaque
frappe.

### Saisie des ordres

La saisie reste textuelle, assistée :

- **Coloration :** trigrammes, verbes et quantités ont chacun leur style
  (IBM Plex Mono ; verbe en `brass-700`, trigramme sur fond `paper-100`).
- **Autocomplétion :** les trigrammes proposés sont ceux que la position
  rend plausibles (territoires adjacents d'abord), avec le nom de la commune.
- **Lien avec la carte :** cliquer un territoire insère son trigramme à la
  position du curseur ; un trigramme saisi recentre et sélectionne la carte
  (voir [Téléphone](#téléphone)).
- **Validation en direct :** les erreurs de
  `POST /api/games/{id}/orders/preview` s'affichent sur la ligne fautive, en
  soulignement `vermilion-700` et en message sous la zone. Le client ne
  réimplémente aucune règle d'ordre (voir [architecture.md](architecture.md)) ;
  l'autocomplétion ne fait que suggérer.
- Le même principe s'applique aux investissements d'hiver et aux cartes
  spéciales.

**Évolution possible.** Des cartes d'ordres (choisir un verbe, puis la cible
sur la carte, et réordonner les cartes) pourront s'ajouter plus tard, comme
une seconde vue de la même chaîne, sans retirer le texte assisté. Elles
s'appuieraient sur les ordres parsés que renvoie la prévisualisation, déjà
utilisés par la surcouche d'intentions (`web/src/lib/intent-overlay.ts`).

### Saison

- Aucun widget dédié : l'ambiance annonce la saison. La carte prend le voile
  de la saison (tableau [Saisons](#saisons)) ; l'interface en reprend la
  teinte d'interface sur un filet sous la barre du haut et sur le marque-page
  actif du tiroir.
- En hiver s'ajoutent la neige actuelle et le panneau d'ordres d'hiver en
  `winter-100` et `winter-700`.
- La saison, l'année et le tour restent écrits en toutes lettres dans la
  barre du haut, pour ne jamais dépendre de la couleur seule.

### Rapport de tour

- Chaque événement devient une carte (pictogramme à l'encre, titre, détail),
  regroupée par catégorie ; les catégories vides sont masquées.
- Par défaut, rien n'est dessiné sur la carte. Le bouton « Voir » d'un
  événement le sélectionne, cadre la carte sur les territoires concernés et y
  dessine l'événement (déplacement, combat, retraite…) jusqu'à la sélection
  suivante.
- Ces cartes remplacent les listes actuelles de `ReportPanel.tsx`.

### Composants de base

- **Boutons :** principal sur `brass-400`, texte `ink-900`, filet `brass-700`,
  survol `brass-500` ; secondaire sur `paper-50`, bordure `line-strong`, texte
  `ink-700` ; discret sans fond ; destructif en `vermilion-700`.
- **Champs :** fond `paper-50`, bordure `line-strong`, anneau de focus
  `brass-700`.
- **Alertes :** fond `vermilion-100` ou `warning-100`, texte et filet dans la
  teinture `-700` correspondante.
- **Rayons :** 4 px pour les contrôles, 6 px pour les panneaux.
- **Ombres :** une seule ombre douce, réservée aux éléments flottants (menus,
  popovers, tiroir).

### Emblème

- Un sceau de cire : cire rouge (dégradé `#c8433a` vers `#6d1210`), anneau
  intérieur, monogramme « C&B » en Cinzel 700 et légende circulaire
  « SIGILLVM · CROWN · ET · BOROUGH ».
- Déclinaisons : favicon (sceau et monogramme sans légende, lisible à 16 px),
  en-tête (36 à 44 px), accueil et connexion (96 px et plus, avec la légende).
- Le rouge du sceau n'est pas une couleur d'interface.
- Le sceau remplace `web/src/components/BrandMark.tsx` et
  `web/public/favicon.svg` ; `theme-color` (`web/index.html`) passe à
  `paper-200`.

### Mouvement

- Aucune animation décorative : ni apparition animée des menus et popovers
  (`tw-animate-css`), ni déplacement animé des pièces, ni son.
- **Précision.** Seules restent les transitions fonctionnelles qui évitent un
  saut brutal, comme l'ouverture du tiroir (150 ms au plus). Elles sont désactivées quand le système demande de réduire les
  animations (`prefers-reduced-motion`).

## Accessibilité

- Les contrastes suivent le tableau [Contrastes](#contrastes).
- Le focus est visible partout : anneau `brass-700` de 2 px, décalé de 2 px.
- La carte reste utilisable au clavier : territoires focalisables et
  sélection par Entrée ou Espace, comme aujourd'hui.
- Chaque pièce garde un libellé accessible (joueur, effectif, statut du
  noble, niveau du bâtiment).
- La couleur n'est jamais le seul repère (principe 4).

## Phasage

1. **Charte graphique :** ce document.
2. **Tokens sans changement visible :** créer les tokens avec les valeurs
   actuelles, remplacer les quelque 700 couleurs écrites en dur, puis refuser
   toute nouvelle valeur hexadécimale dans `web/src` hors du module des données
   de carte. Sortie : captures identiques et tests verts.
3. **Nouvelle identité :** basculer les tokens sur cette charte, charger
   Cinzel, Alegreya et IBM Plex Mono, recolorer les composants shadcn, retirer
   Geist et le bloc `.dark`. Sortie : contrastes AA vérifiés.
4. **Carte gravée :** terrain, icônes de frontière, évêchés, bannières, écus,
   pions de nobles, bâtiments, sélection, saisons, légende, teintures et
   armoiries générées des joueurs.
5. **Ergonomie :** barre du haut et menu « Partie », tiroir à signets et
   action collante, ambiance de saison, onglets du téléphone avec l'écran
   partagé Ordres, rapport en cartes d'événements.
6. **Saisie assistée des ordres :** coloration, autocomplétion, lien avec la
   carte, validation en direct.
7. **Emblème et pages annexes :** sceau, favicon, accueil, connexion, règles
   et FAQ.

Viennent ensuite, au rythme des specs : les attributs de titre des nobles,
le choix du blason par le joueur, puis, éventuellement, les cartes d'ordres.

## Questions ouvertes

- **Armoiries des joueurs :** les armoiries générées peuvent être dérivées
  côté front de l'ordre du siège, sans changement de contrat. Le blason
  choisi par le joueur, lui, doit être stocké (profil) et publié dans
  `state.json`, ce qui change le contrat. À trancher dans l'issue de la
  phase 4 : dériver d'abord côté front, ou introduire le champ dès la
  génération pour préparer le choix.
- **Palette du moteur :** `internal/engine/game.go` attribue aujourd'hui
  16 couleurs. Passer aux teintures change les valeurs de `players[].color`
  des nouvelles parties ; le schéma, lui, ne change pas.
- **Fiefs** ([titres.md](titres.md),
  [#194](https://github.com/fogfactory/crown-and-borough/issues/194)) : le
  liseré est réservé aux évêchés et la bannière au contrôle, donc la
  représentation d'un fief et de son titre reste à définir. Piste : une
  bannière couronnée sur la capitale du fief et le titre dans la fiche du
  territoire.
- **Rivières** ([cartographie.md](cartographie.md)) : le symbole de fleuve est
  prévu ; le dessin des ponts reste à définir.
- **Niveau de détail des étiquettes :** seuils à régler sur des cartes réelles
  (voir [Étiquettes](#étiquettes)).
