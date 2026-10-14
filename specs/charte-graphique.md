# Charte graphique

**Suivi :** milestone [Refonte graphique](https://github.com/fogfactory/crown-and-borough/milestone/27),
une issue par phase (voir [Phasage](#phasage)).

**Dépend de :** [architecture.md](architecture.md) pour la stack front et les
contrats `map.json` / `state.json` ; [titres.md](titres.md#contrôle-et-occupation)
pour les notions de territoire contrôlé et occupé ;
[cartographie.md](cartographie.md) pour les futures rivières ;
[politique.md](politique.md) et [succession.md](succession.md) pour la
couronne, la lignée et les prétentions.

Les décisions 1 à 20 ont été actées lors de l'audit graphique du
27 septembre 2026, les décisions 21 à 32 lors de l'audit du 12 octobre 2026
qui a suivi le milestone Royauté (fiefs, couronne, ordres royaux, cartes de
fief). Elles ne sont pas encore implémentées : l'interface actuelle
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
3. **Une information, un canal visuel.** Contrôle : bannière. Fief : couronne
   sur la bannière. Occupation : écu de l'armée présente. Évêché : cadre et
   liseré. Terrain : symboles d'encre. Saison : voile de la carte.
   Deux informations ne partagent jamais un canal.
4. **Jamais la couleur seule.** Toute information portée par une couleur a un
   second repère : pièce héraldique, symbole de terrain, trait de frontière,
   forme ou texte.
5. **Des tokens, pas de valeurs en dur.** Couleurs, polices, tailles et rayons
   passent par les tokens du thème (`web/src/index.css`). Les composants
   n'écrivent aucune valeur hexadécimale ; les données de rendu de la carte
   (teintures, symboles) sont centralisées dans un module dédié.
6. **Le nom d'abord, le code ensuite.** Un trigramme ou un code seul n'a sa
   place que dans une feuille d'ordres ou, sur la carte, pour un territoire
   sans particularité. Partout ailleurs, on écrit le libellé complet suivi du
   code entre parenthèses, et ce libellé mène à la carte (voir
   [Identifiants et liens vers la carte](#identifiants-et-liens-vers-la-carte)).

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
| 21 | Fiefs sur la carte | Bannière couronnée sur la capitale ; trigramme de la capitale sur les bannières des autres territoires du fief | Cartouche gravé, contour du fief, capitale seule |
| 22 | Couronne et Église | Marque-page « Royaume » du tiroir | Médaillons sur la carte, boutons de la barre du haut, section de la lignée |
| 23 | Titres d'un noble dans les fiches | Pion et attributs, puis titres en texte | Étiquettes unifiées, texte seul, pictogrammes seuls |
| 24 | Scores et victoire | Popover actuel restylé | Piste de score, tableau compact, scores dans la barre |
| 25 | Textes d'aide | Une ligne de résumé et un « ? » qui ouvre le texte complet | Mode apprenti, ligne d'aide unique, renvoi aux règles |
| 26 | Ordres de titres | Groupés par titulaire | Menu « Ajouter un ordre », depuis la carte, autocomplétion seule |
| 27 | Réglage d'un ordre de titre | Formulaire déplié dans le tiroir, carte cliquable | Ciblage sur la carte, modale restylée, modèle de ligne |
| 28 | Main de cartes | Éventail de cartes à jouer | Liste compacte, jetons restylés, main repliée |
| 29 | Trésor | Bandeau d'une ligne, dépliable | Barre du haut, blocs repliés, livre de comptes |
| 30 | Fiche de territoire | Cartouche posé sur la carte | Sections repliables, marque-page Territoire, infobulle et tiroir |
| 31 | Lignée et alliances | Arbre généalogique | Maisons compactes, page dédiée, restyle de l'actuel |
| 32 | Résultats répétitifs du rapport | Carte de synthèse et encadré des rumeurs | Filtrage seul, « chez vous » d'abord, tableau |
| 33 | Identifiants hors des ordres | Libellé complet suivi du code entre parenthèses ; code seul uniquement dans les feuilles d'ordres et pour les territoires ordinaires de la carte | Codes seuls partout |
| 34 | Liens vers la carte | Un clic sur un territoire ou un noble cité dans le rapport ou le poste de commandement cadre la carte sur lui | Aucun lien |

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
| `brass-400` | `#c9a24a` | Fond du bouton principal, halo de sélection, couronnes des nobles titrés et des fiefs |
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
- Le moteur attribue la teinture du siège dans `players[].color` et publie
  les armoiries (champ, métal, pièce) dans `players[].arms` dès la création
  de la partie ([architecture.md](architecture.md#statejson)) ; le front ne
  les recalcule pas. Une partie sans armoiries montre un champ plein de la
  couleur du joueur.
- **Deux étapes.** D'abord, chaque joueur reçoit les armoiries générées
  ci-dessus. Ensuite, un joueur pourra choisir son propre blason (champ,
  pièce, meuble) dans son profil ; ce choix se superpose aux armoiries
  générées et reste soumis aux mêmes règles (teinture en champ, métal en
  pièce, pas d'or en teinture, pas de doublon dans une partie).

### Évêchés

Six teintes terreuses, hors de la palette des joueurs, remplacent les couleurs
Tailwind de `web/src/lib/region-color.ts`. Au-delà de six évêchés, les motifs
existants (`REGION_PATTERNS`) s'ajoutent, dessinés à l'encre à 30 %.

| | Ocre | Ardoise | Olive | Brique | Lavande | Sauge |
|---|---|---|---|---|---|---|
| Teinte | `#a9803a` | `#5f7d8c` | `#7b7d47` | `#9a5f4b` | `#7f7496` | `#6b8a74` |
| Texte du chef-lieu (4,6:1 sur `paper-map`) | `#80612c` | `#516a77` | `#67693c` | `#8f5846` | `#6b617e` | `#536c5a` |

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
symboles, posés sur une grille en quinconce au pas du terrain et décalés au
hasard, restent à 6,5 unités des frontières et hors d'une ellipse de demi-axes
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
  4. sinon, **escarpement** : une ligne de crête dont les dents de falaise
     tombent vers le territoire le plus bas (montagne > colline > forêt >
     plaine > marécage ; à égalité, vers le trigramme le plus grand).
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
ligne de base ; mât `ink-900` de 14 unités, drapeau de 12 × 8 unités. Chaque
joueur a un drapeau et un écu définis une fois (`<symbol>`), instanciés par
`<use>` sur chaque pièce.

### Fiefs : bannière couronnée

- La capitale d'un fief porte la bannière de son joueur surmontée de la
  couronne du rang du fief (baronnie, comté, marquisat, duché ; mêmes dessins
  que les couronnes des nobles), en `brass-400` cernée d'encre.
- Les autres territoires du fief gardent leur bannière ordinaire, marquée du
  trigramme de la capitale en petites capitales Cinzel sous le drapeau : on
  lit d'un coup d'œil à quel fief appartient une case.
- Un fief de la couronne ([politique.md](politique.md)) porte la couronne
  fermée du roi sur sa capitale.
- Le nom du fief (« Baronnie de Rochevent ») ne s'écrit plus sur la carte :
  il passe dans l'infobulle de la bannière et dans la fiche du territoire.
  L'étiquette ovale de `MapFiefLayer.tsx` et le mini-écu marqué du trigramme
  (`OwnershipBadge`) disparaissent.

**Précision.** La couronne de fief mesure 10 × 6 unités, posée sur le haut
du mât ; le trigramme de rattachement fait 6 unités. Un fief vacant garde sa
couronne au-dessus d'une bannière `neutral`.

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
- **Précision.** Un titre de courtoisie (baronne, comtesse, duchesse, reine ;
  [titres.md](titres.md#titres-de-courtoisie)) se montre par la couronne du
  titre en simple filet `brass-700`, sans remplissage : il se lit sans se
  confondre avec un titre réel. Un noble excommunié porte un socle cerné d'un
  pointillé `ink-900`. Les dignités (bâtard, sorcière, astrologue, espion…)
  ne figurent pas sur la carte : publiques, elles sont écrites dans les
  fiches ; secrètes, seul leur propriétaire les voit.
- Plusieurs nobles sur une case se placent côte à côte, espacés de 12 unités.
- **Précision.** La crosse se tient à gauche du pion : les pions voisins se
  peignent de gauche à droite et couvriraient une crosse tenue à droite. Le
  chapeau du cardinal et la tiare du pape coiffent la tête ; la couronne
  séculière se pose au-dessus. Le cachot dessine trois barreaux et deux
  traverses à l'encre par-dessus le pion.
- Le pion remplace le losange de `NobleMarker`.

Les titres de fief ([titres.md](titres.md)), religieux
([religieux.md](religieux.md)) et royaux ([politique.md](politique.md)) sont
livrés dans `develop`, marquisat compris : les attributs sont donc tous dus
dès la phase 4. Ils remplacent les pastilles de titre (`NobleHolder.tsx`,
fiches de nobles) partout où un noble est représenté : la fiche montre le pion
puis les titres en texte. L'arbre de la lignée garde son dessin jusqu'à sa
refonte (phase 7).

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
- **Territoire ordinaire et territoire particulier :** un territoire sans
  particularité peut ne montrer que son trigramme, en vue synthétique. Un
  territoire particulier (capitale d'un joueur, capitale de fief, chef-lieu
  d'évêché, fief de la couronne) montre toujours son nom complet au-dessus de
  son code, à tous les niveaux de zoom.
- Les autres pièces de la carte (bannières, écus, pions, bâtiments) ne
  s'étiquettent pas d'un code : leur infobulle donne le libellé complet et le
  code entre parenthèses (« Baron Rainaud de Rochevent (RAI) »). Seul le
  trigramme de rattachement d'un fief reste écrit sous la bannière, en vue
  synthétique.

**Précision.** Quand le nom d'un territoire ordinaire rendu fait moins de
10 px à l'écran, seul le code s'affiche ; le seuil suit le zoom. Sur la carte
à 4 joueurs (52 territoires), la vue d'ensemble montre les seuls territoires
particuliers et le premier cran de zoom tous les noms. Un nom coupé l'est au
trait d'union le plus proche de son milieu. La bannière se plante à gauche
de la ligne la plus large de l'étiquette (nom ou code), sur la dernière ligne
de base du nom, ou sur celle du code quand le nom est masqué.

### Légende

La légende montre des échantillons de symboles au lieu de pastilles de couleur :
les cinq terrains, les icônes de frontière infranchissable et le pointillé
franchissable, la bannière, la bannière couronnée d'un fief, l'écu, le pion
de noble (libre, otage, au cachot, excommunié, et ses attributs de titre) et
les bâtiments. Les boutons des calques (intentions, contrôle, évêchés,
calamités, cartes) sont conservés.

## Écrans et composants

### Identifiants et liens vers la carte

Les trigrammes de territoire et de noble, les codes de carte (`BT`, `TX`),
de dignité (`BAS`) et d'ordre (`W X`) sont la langue des feuilles d'ordres.
Ailleurs, ils accompagnent un nom, ils ne le remplacent pas.

- **Feuilles d'ordres :** les codes s'écrivent seuls. Toute ligne produite
  par une aide (formulaire d'ordre de titre, carte jouée depuis la main,
  modèle de saisie) porte un commentaire en libellés complets :
  `W X ZRK RVE RVN # Sieur Robert (ZRK) accorde à la baronnie de Rochevent
  (RVE) l'annexion de Ravinval (RVN)`.
- **Carte :** code seul admis pour un territoire ordinaire (voir
  [Étiquettes](#étiquettes)) ; libellé complet et code pour un territoire
  particulier et dans toutes les infobulles.
- **Partout ailleurs** (rapport, poste de commandement, fiche de territoire,
  marque-page Royaume, lignée, scores, menus et listes déroulantes des
  formulaires, messages d'erreur) : libellé complet, puis le code entre
  parenthèses en IBM Plex Mono `ink-500` : « Rochevent (RVE) »,
  « Baron Rainaud de Rochevent (RAI) », « Beau temps (BT) ». Un libellé
  long peut se réduire au nom propre (« Rainaud (RAI) ») dans une cellule
  étroite, jamais au code seul.
- **Lien vers la carte :** dans le rapport et dans le poste de commandement
  (hors zone de saisie), un territoire ou un noble cité est un lien. Le clic
  cadre la carte sur le territoire, ou sur le territoire où se trouve le
  noble, le sélectionne et ouvre sa fiche ; le pion du noble cliqué reçoit
  le halo de sélection. Sur téléphone, le toucher bascule sur l'onglet
  Carte.
- **Style du lien :** le texte garde sa couleur, souligné d'un pointillé
  `brass-700` ; au survol, le soulignement devient plein et le territoire
  visé s'éclaire sur la carte.

**Précision.** Un noble dont la position n'est pas connue du joueur (vue
privée, [information.md](information.md)) ou mort n'a pas de lien : son nom
s'écrit en texte simple, avec une infobulle « position inconnue » ou
« décédé ». Dans la zone de saisie, le comportement reste celui de
[Saisie des ordres](#saisie-des-ordres) : un trigramme complet et valide
recentre la carte, sans clic.

**Précision.** Sur ordinateur, suivre un lien laisse le tiroir sur son
marque-page : la carte se cadre sur le territoire et ses voisins, le
sélectionne, et la fiche reste au poste de commandement jusqu'au cartouche
([Fiche de territoire](#fiche-de-territoire)). Le territoire survolé par un
lien prend un liseré `brass-700` en tirets sur un voile `brass-400`. Dans
les listes déroulantes des formulaires, une région s'écrit comme un
territoire (« Alpilles (ALP) ») et un fief comme dans le marque-page Royaume
(« Baronnie de Rochevent (RVE) »). Les commentaires en libellés complets des
lignes produites par une aide arrivent avec les formulaires dans le tiroir
([Poste de commandement](#poste-de-commandement)).

### Disposition sur ordinateur

La carte occupe toute la largeur ; les panneaux passent dans un tiroir :

- **Barre du haut, fine et sur une ligne :** sceau et « Crown & Borough » en
  Cinzel ; saison, année et tour en texte (« Automne 1002 · tour 7 sur 40 ») ;
  scores (voir [Scores et victoire](#scores-et-victoire)) ; lignée ; pastilles de soumission ; joueur actif en hotseat ; navigation
  Règles et FAQ ; un menu « Partie » qui regroupe la création d'une partie
  (joueurs, graine, années), la résolution forcée et la langue.
- **Tiroir à droite :** posé sur la carte, avec des onglets en marque-pages
  (Ordres, Rapport, Royaume, Règles) qui dépassent de son bord gauche. Il se replie
  pour rendre toute la carte ; le marque-page actif est en `brass-400`. Les
  onglets tiennent lieu de titre : le titre qui répète l'onglet actif
  (`GamePanelCard.tsx`) disparaît.
- **Action principale :** le bouton d'envoi des ordres vit dans un pied de
  tiroir collant, toujours visible. Libellé proposé : « Sceller les ordres ».

**Précision.** Le tiroir mesure 384 à 416 px comme la colonne actuelle ; la
carte se recadre quand il s'ouvre ou se replie. Un bouton sous les
marque-pages le replie ; un marque-page le rouvre. Le bandeau « Carte
publique · vue commune » posé sur la carte disparaît ; l'aide aux gestes
rejoint la légende, dont le bouton passe en haut à gauche de la carte pour
laisser le bord droit aux marque-pages. Le bouton de navigation qui ramène à
la partie depuis les Règles ou la FAQ s'appelle « Carte », pour ne pas se
confondre avec le menu « Partie ». Une erreur d'envoi s'affiche dans le pied
collant, au-dessus de « Sceller les ordres ». En ligne, la résolution forcée
rejoint aussi le menu « Partie », avec sa confirmation.

### Téléphone

- Navigation d'application par une barre d'onglets en bas : Carte, Ordres,
  Rapport, Royaume, Règles. Chaque vue a tout l'écran ; le panneau glissant actuel
  (`web/src/components/ui/panel-sheet.tsx`) disparaît.
- **Carte :** plein écran, ouverte sur les territoires du joueur. La fiche
  d'un territoire touché s'ouvre en bas de l'écran, sur la même largeur que
  le cartouche de l'ordinateur (voir [Fiche de territoire](#fiche-de-territoire)).
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

**Précision.** Le téléphone s'ouvre sur l'onglet Carte. Le cadrage sur les
territoires du joueur garde la carte pleine : il ne montre jamais de papier
vide au-delà de son bord. La barre du haut écrit la saison et le tour en
abrégé (« Automne 1002 · 7/40 ») ; le joueur actif du hotseat, la navigation
et la langue passent dans le menu « Partie », la lignée s'ouvre depuis le
marque-page Royaume. En ligne, la partie ne garde que sa propre barre :
retour, saison et tour, scores, salon et menu ; l'état du réseau n'y apparaît
que hors ligne. Le pied « Sceller les ordres » n'accompagne que l'onglet
Ordres. Le lien entre la carte et la saisie suit la phase 6, le toucher d'un
événement du rapport la refonte du rapport.

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

### Textes d'aide

Les explications occupent aujourd'hui une bonne part du poste de
commandement (ordres d'hiver, cartes spéciales, ordres de titres, main,
chaînes d'ordres, en-têtes des fenêtres d'ordre).

- Chaque section garde **une seule ligne** de résumé sous son titre, en
  `ink-500`.
- Un bouton « ? » (cercle `line-strong`, 24 px, cible de 44 px sur téléphone)
  à droite du titre ouvre un popover avec le texte complet et un lien
  « Voir la règle » vers la section correspondante du marque-page Règles.
- Les syntaxes d'ordre (`D C KIND`, `R N CCC XXX`…) ne s'écrivent plus dans
  les phrases : elles apparaissent dans les modèles de saisie et dans
  l'autocomplétion.
- Une information d'état (calamités prévues par l'astrologue, deck restant,
  limite de main) n'est pas une aide : elle reste visible, sous forme de
  donnée courte (« Deck : 43 · Main : 5/6 »).

**Précision.** Le résumé tient en 80 caractères au plus. Le popover se ferme
par Échap ou un clic à l'extérieur ; il n'est jamais ouvert par défaut.

### Poste de commandement

De haut en bas, sous les marque-pages : trésor, ordres de titres, main de
cartes, saisie des ordres, pied collant d'envoi.

- **Trésor :** un bandeau d'une ligne remplace les blocs « Revenu prévu » et
  « Consommation prévue » : stock, revenu par tour d'action, consommation et,
  s'il y en a un, le risque de famine en `warning-700`
  (« ⚖ 20 R · +10 R / tour · −1 R · famine : aucune »). Un clic le déplie
  sur le détail actuel (hors fief, fiefs, moulins, rente royale, taxes…),
  où chaque fief et territoire s'écrit en toutes lettres
  (« Baronnie de Rochevent (RVE) : +3 R »).
- **Ordres de titres :** groupés par titulaire. Chaque noble qui peut émettre
  un ordre de titre a une ligne : son pion avec ses attributs, son nom et ses
  titres en texte court, puis ses actions en boutons secondaires au libellé
  court (« Excommunier », « Accorder une annexion »). Les ordres de fief vont
  sous le titulaire du fief, ceux de la couronne sous le roi ou le régent,
  ceux des dames sous la dame. Le code de l'ordre (`X E`, `W X`) reste dans
  l'infobulle du bouton.
- **Réglage d'un ordre :** le bouton déplie un formulaire dans le tiroir,
  sous la ligne du titulaire, au lieu de la fenêtre modale actuelle
  (`OrderDialog.tsx`). La carte reste visible : un clic sur un territoire ou
  un noble remplit le champ actif, signalé par un anneau `brass-700`. Le
  formulaire montre la ligne d'ordre produite en IBM Plex Mono, avec son
  commentaire en libellés complets, puis « Annuler » et « Ajouter l'ordre ».
  Ses listes déroulantes proposent « Ravinval (RVN) », pas « RVN ». Un seul formulaire est ouvert à la fois.
- **Main de cartes :** un éventail de petites cartes à jouer (56 × 80 px) :
  pictogramme à l'encre, nom, code en pied. Trois familles se distinguent
  par le filet et la pièce d'angle, pas par une couleur vive : cartes
  spéciales (filet `brass-700`), cartes de noble et de prétention (filet
  `ink-900`, pion en angle), dignités (double filet). Un clic sur une carte
  ouvre « Jouer » et « Défausser », qui insèrent la ligne dans la saisie ; la
  croix permanente disparaît. Une carte défaussée ce tour passe en grisé
  barré. Le deck et la limite de main s'écrivent sous l'éventail.
- **Calamités à venir :** l'encadré rouge actuel passe en `warning-100` et
  `warning-700` : une prévision n'est pas une erreur.
- **Saisie :** une zone par noble au printemps, en été et en automne, la zone
  des cartes de noble au-dessus ; une seule zone en hiver (voir
  [Saisie des ordres](#saisie-des-ordres)).

**Précision.** Sur téléphone, l'éventail défile horizontalement et chaque
carte mesure au moins 64 × 88 px.

**Précision.** En hiver, le bandeau de trésor n'écrit que le stock et le
revenu : il n'y a ni ravitaillement ni famine à prévoir. Le risque de famine
compte les armées menacées (« famine : 2 armées »). La main réunit cartes
spéciales et cartes de noble dans un seul éventail, en toute saison ; les
commandes de la carte choisie (« Jouer », « Défausser », ou les ordres d'une
carte spéciale en saison d'action) s'affichent sous l'éventail. Une carte
spéciale en plusieurs exemplaires porte un « ×2 » en coin.

### Marque-page Royaume

Le marque-page « Royaume » regroupe ce qui concerne tout le royaume et qui
flotte aujourd'hui sur la carte (`CrownPanel.tsx`,
`ReligiousHierarchyPanel.tsx`) :

- **Couronne :** roi, reine (ou reine douairière), régent, fiefs de la
  couronne avec leur nombre de territoires.
- **Église :** pape, cardinaux, puis un évêque par évêché, « vacant » en
  italique `ink-500`.
- Chaque titulaire s'écrit avec son pion à attributs, son nom complet suivi
  de son code entre parenthèses, et la teinture de son joueur en pastille ;
  un fief de la couronne s'écrit « Baronnie de Rochevent (RVE) · 4
  territoires ». Noms et fiefs sont des liens vers la carte.
- En pied : un bouton « Ouvrir la lignée et les alliances ».

Plus rien ne se pose sur la carte en dehors de la légende et des contrôles de
zoom.

### Fiche de territoire

Sélectionner un territoire ouvre un **cartouche** posé sur la carte, relié au
territoire par un trait d'encre ; le tiroir reste consacré aux ordres.

- **En-tête :** nom en Alegreya italique suivi du code entre parenthèses,
  bannière, terrain.
- **Une ligne par sujet,** absente quand elle est vide : fief et rang (avec la
  couronne), titulaire, capitale ou cité et sa défense, ressources et revenu
  du territoire, armée (écu, effectif, ordre), nobles présents (une rangée de
  pions à attributs, leurs noms au survol ou au toucher), bâtiments.
- Les doublons actuels fusionnent : un seul « revenu », sous la forme
  « 40 R · +3 R / tour ».
- Le cartouche se ferme par Échap, par la croix ou en sélectionnant ailleurs.

**Précision.** Le cartouche mesure 280 px de large au plus et se place du côté
du territoire le plus éloigné du tiroir ; il ne cache jamais le territoire
sélectionné. Le détail d'un noble (statut, propriétaire, dignités) s'ouvre en
popover depuis son pion.

### Scores et victoire

Le popover actuel des scores (`Scoreboard.tsx`) est conservé et restylé : une
carte par joueur, avec ses armoiries, son score de titres en Alegreya 700,
l'objectif, le mode (seul ou allié) et, en fin de partie, la victoire. Les
étiquettes « SOLO » deviennent du texte (« seul ») ; le score du joueur actif
est en `brass-700`.

### Lignée et alliances

La fenêtre « Lignée et alliances » (`LineageDialog.tsx`) devient un **arbre
généalogique** dessiné à l'encre sur parchemin.

- Chaque maison s'ouvre sous ses armoiries. Ses nobles sont des médaillons
  (pion à attributs, nom, rang de succession en chiffre) rangés dans l'ordre
  de succession.
- Un mariage est un double trait horizontal entre deux médaillons : le
  mariage de tête en `brass-700`, un mariage secondaire en `ink-700`, un
  mariage terminé en hachures `ink-500`.
- Une prétention descend du couple vers l'héritier qui la porte, en trait
  pointillé, comme une filiation.
- Les titres s'écrivent en texte sous le nom (décision 23) ; plus aucune
  étiquette n'est coupée.
- Le score et le statut de victoire de chaque maison restent en pied de
  fenêtre, sur une ligne par joueur.
- Le graphe d'alliances actuel devient un encart de l'arbre (les maisons et
  leurs alliances en vue d'ensemble), plus une vue séparée.
- **Simulation :** le mode « Simuler » garde son bandeau
  (annuler, rétablir, réinitialiser, quitter) en `warning-100` ; l'arbre
  simulé porte un filigrane « Hypothèse » en Cinzel `ink-500` à 15 %, pour
  qu'aucune capture ne se confonde avec l'état réel.

**Précision.** Le modèle ne connaît pas de filiation au-delà des prétentions
([succession.md](succession.md)) : l'arbre ne dessine que l'ordre de
succession, les mariages et les prétentions. Il défile dans les deux sens et
se cadre sur la maison choisie dans « Focus ».

### Rapport de tour

- Chaque événement devient une carte (pictogramme à l'encre, titre, détail),
  regroupée par catégorie ; les catégories vides sont masquées.
- Par défaut, rien n'est dessiné sur la carte. Le bouton « Voir » d'un
  événement le sélectionne, cadre la carte sur les territoires concernés et y
  dessine l'événement (déplacement, combat, retraite…) jusqu'à la sélection
  suivante.
- **Résultats répétitifs :** les résultats d'une même nature (conservation
  d'hiver, revenus, production…) forment une seule carte de synthèse
  (« Conservation : 4 stocks divisés par deux »), dépliable sur la liste
  (« Rochevent (RVE) : 40 → 20 R »). Les
  lignes sans effet (« 0 → 0 ») ne s'affichent pas.
- **Rumeurs :** un encadré « On dit au marché… » en Alegreya italique, sur
  `paper-100` avec un filet `line` ; les cartes jouées et calamités à venir
  forment un encadré voisin, en texte droit.
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

1. **Charte graphique** ([#443](https://github.com/fogfactory/crown-and-borough/issues/443)) **:** ce document.
2. **Tokens sans changement visible** ([#444](https://github.com/fogfactory/crown-and-borough/issues/444)) **:** créer les tokens avec les valeurs
   actuelles, remplacer les quelque 700 couleurs écrites en dur, puis refuser
   toute nouvelle valeur hexadécimale dans `web/src` hors du module des données
   de carte. Sortie : captures identiques et tests verts.
   **Précision :** chaque usage reçoit le token de son rôle dans la charte ;
   quand un rôle couvre aujourd'hui plusieurs nuances, des paliers
   intermédiaires (`paper-60`, `vermilion-150`…) et des familles d'attente
   (`gold`, `claim`, `steel`) les conservent, à fusionner en phase 3. Les
   couleurs des dessins SVG (carte, lignée, emblème) vivent dans
   `web/src/lib/map-palette.ts` ; une règle ESLint refuse tout littéral
   hexadécimal ailleurs dans `web/src`, hors tests et fixtures.
3. **Nouvelle identité** ([#445](https://github.com/fogfactory/crown-and-borough/issues/445)) **:** basculer les tokens sur cette charte, charger
   Cinzel, Alegreya et IBM Plex Mono, recolorer les composants shadcn, retirer
   Geist et le bloc `.dark`. Sortie : contrastes AA vérifiés.
4. **Carte gravée** ([#446](https://github.com/fogfactory/crown-and-borough/issues/446)) **:** terrain, icônes de frontière, évêchés, bannières, écus,
   pions de nobles et attributs de titre, bannières couronnées des fiefs,
   bâtiments, sélection, saisons, légende, teintures et
   armoiries générées des joueurs.
5. **Ergonomie** ([#447](https://github.com/fogfactory/crown-and-borough/issues/447)) **:** libellés complets et liens vers la carte, barre du haut
   et menu « Partie », tiroir à signets et action collante, marque-page
   Royaume, ambiance de saison, onglets du téléphone avec l'écran partagé Ordres, textes d'aide réduits, poste de
   commandement (trésor, ordres de titres groupés et formulaires dans le
   tiroir, main en éventail), cartouche de territoire, rapport en cartes
   d'événements, scores restylés.
6. **Saisie assistée des ordres** ([#448](https://github.com/fogfactory/crown-and-borough/issues/448)) **:** coloration, autocomplétion, lien avec la
   carte, validation en direct.
7. **Emblème, lignée et pages annexes** ([#449](https://github.com/fogfactory/crown-and-borough/issues/449)) **:** sceau, favicon, arbre
   généalogique de la lignée, accueil, connexion, règles et FAQ.

Viennent ensuite : le choix du blason par le joueur, puis, éventuellement,
les cartes d'ordres.

## Questions ouvertes

- **Arbre de la lignée :** la disposition automatique d'un arbre avec des
  mariages entre maisons et des prétentions croisées reste à prototyper sur
  le scénario de démonstration `crown` (`scripts/demo.sh crown`) avant la
  phase 7 ; la disposition manuelle actuelle (« Réinitialiser la
  disposition ») pourra être conservée si l'automatique ne suffit pas.
- **Rivières** ([cartographie.md](cartographie.md)) : le symbole de fleuve est
  prévu ; le dessin des ponts reste à définir.
- **Niveau de détail des étiquettes :** le seuil de 10 px convient à la carte
  à 4 joueurs ; il reste à vérifier sur une carte hotseat à 16 joueurs (près
  de 200 territoires, voir [Étiquettes](#étiquettes)).
