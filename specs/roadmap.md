# Roadmap d'Implémentation : Crown & Borough (MVP)

Plan d'implémentation pour atteindre le plus rapidement possible un prototype testable.
Sources : `specs/gdd.md` (règles) et `specs/architecture.md` (stack).

Suivi d'exécution : chaque ID est lié à son prompt dans `specs/prompts/`.
Statuts : `Prompt écrit`, `Plan prêt`, `En cours`, `Fait` et `Bloqué`.
Toute transition de statut doit être reportée ici.

## Stratégie globale

Trois paliers de "testable" successifs, chacun validé avant le suivant :

1. **Palier 1 — Moteur testable :** règles du GDD en Go pur, validées par `go test` (aucune infra).
2. **Palier 2 — Latence testable :** 2e pilier (messagers, vision T-x), toujours en Go pur.
3. **Palier 3 — Jouable entre amis :** API + persistance + front + déploiement Cloud Run.

**Décisions actées :**

- La latence d'information est différée au palier 2 : la boucle de jeu démarre en vision T0 partout.
- La carte est générée par **algorithme de Voronoï seedé** : un anneau de sites « cadre » absorbe les bords, puis les territoires intérieurs sont ré-ancrés avec un padding ; aucun territoire livré ne touche le bord. Elle compte **8 territoires par joueur**. Chaque frontière géométrique retenue est un arc qualifié franchissable ou infranchissable (crêtes, marécages) : le graphe est planaire et connexe, avec un degré franchissable compris entre 2 et le maximum du terrain (3 en montagne, marécage ou colline ; 5 en plaine ou forêt).
- Persistance : **JSON d'abord** (1 fichier par partie), migration **Postgres/sqlc** ensuite.
- Le **front est construit au fur et à mesure** du plan pour tester localement dès que possible.
- Les **infrastructures n'ont pas de propriétaire** : elles appartiennent à leur case (celui qui contrôle la case en bénéficie). Construire un château sur un village le **remplace** (jamais deux structures par case).
- **P1.2g — Armée unique par territoire :** le modèle conserve une armée `{owner, size}` au plus par territoire, avec un `ArmyID` interne global (`A1`, `A2`, …). Le front l'adresse par territoire et la caractérisation individuelle est différée.
- **P1.9 — Non-adjacence différée :** une non-adjacence n'est pas un motif de rejet atomique à la réception d'une chaîne. Elle est vérifiée à l'exécution : les ordres précédents restent valides, l'ordre non adjacent brise la chaîne et le suffixe postérieur est inexécutable. Les autres erreurs de réception restent bloquantes.
- **P1.12 — Villages neutres productifs :** un village produit et stocke la production de base qu'il soit neutre ou contrôlé. Son stock reste sur place à la capture et devient disponible au nouveau contrôleur ; il n'est pas disponible avant la capture.
- **Identité visible des territoires :** P1 affiche les trigrammes dans les messages, rapports et contrôles utilisateur, tout en conservant provisoirement les `TerritoryID` internes. Le refactor supprimant réellement le double identifiant est reporté à P3.0, avant P3.1.

## P0 — Fondations

| ID | Tâche | Livrable | Critère de test | Front | Statut |
|---|---|---|---|---|---|
| [P0.1](prompts/p0.1-squelette-repo.md) | Squelette repo Go | Structure standard, go.mod, Makefile, Dockerfile multi-stage, CI GitHub Actions | `make build` vert, CI verte | — | Fait |
| [P0.2](prompts/p0.2-assets-csv.md) | Assets CSV + loader Go | communes.csv (`code;nom;terrain`) et prenoms.csv (médiévaux français), codes trigrammes uniques par catégorie ; loader Go vérifiant les invariants | Tests unitaires : unicité par fichier, terrains valides, couverture des affinités | — | Fait |
| [P0.3](prompts/p0.3-scaffold-front.md) | Scaffold front | Vite + React + TypeScript + Tailwind + shadcn/ui | App dev se lance | MapViewer v0 sur fixtures `map.ts` (statique) + `state.ts` (dynamique) : fond + couche vivante superposés | Fait |

## P1 — Moteur de jeu (palier 1)

| ID | Tâche | Livrable | Critère de test | Front | Statut |
|---|---|---|---|---|---|
| [P1.1](prompts/p1.1-modeles-metier.md) | Modèles métier | Territoire, Armée (owner + taille, une par territoire — P1.2g), Noble, Infrastructure, Ressources, GameState | Compile + tests modèles | — | Fait |
| [P1.2](prompts/p1.2-generation-carte.md) | Génération de carte Voronoï | Arcs géométriques qualifiés, terrains, villages neutres rares, nommage des territoires | Carte déterministe par seed et nombre de joueurs, invariants (arcs géométriques, connexité, degré [2, max(terrain)], unicité trigrammes, nombre et répartition des villages) | Endpoint dev `/api/map?players=N` (2..5, défaut 4, cache mémoire) → la vraie carte s'affiche | Fait |
| [P1.3](prompts/p1.3-ordres-chaines.md) | Parser & modèles d'ordres | Parser texte des chaînes (format specs/architecture.md §6), 8 symboles A/S/H/J/P/D/O/K, liaison par transition (single/loop), réception (armée sur la position de la 1re ligne, remplacement de chaîne), capacité noble 1 émission/tour ; chaînes exposées sous `StateView.army.chain` dans le DTO dev sans filtrage | Tests : parsing, validation par type, réception, remplacement, capacité, round-trip JSON | — | Fait |
| [P1.4](prompts/p1.4-resolution-combat.md) | Résolution : progression, mouvement & combat | Progression de TOUTES les chaînes simultanément, mouvement, attaque, soutien, jonction (fusion), dispersion, pillage, combats & retraites | Scénarios de combat (égalité, supériorité, soutiens, retraites, destructions), progression single/loop appliquée | — | Fait |
| [P1.5](prompts/p1.5-ravitaillement-famine.md) | Ravitaillement & famine | Coût 2^(N-1), flux BFS depuis les châteaux/villages contrôlés, portée 3/5, stocks, algorithme famine | Tests : déficits, ordre d'épuisement, famine | — | Fait |
| [P1.6](prompts/p1.6-hiver-investissements.md) | Phase d'Hiver | Conservation 50 %, recrutement Nobles, construction Infrastructures, départ des joueurs sur villages distincts (château auto-construit) | Test : année complète sans perte | — | Fait |
| [P1.7](prompts/p1.7-rapport-tour-hotseat.md) | Rapport de tour + intégration | TurnReport (rapports des châteaux/villages contrôlés), simulation d'une année complète multi-joueurs | Test de bout en bout d'une année type | Endpoints dev `/api/map` + `/api/state` + `/api/orders` → boucle locale hotseat : ordres via le front, résolution, rapport affiché, et affichage de la chaîne courante, de son index et de son ordre courant pour l'armée sélectionnée ; `/api/state` est créé en avance par P1.2f comme état d'exemple statique et évolue ici vers l'état hotseat dynamique | Prompt écrit |
| [P1.8](prompts/p1.8-interaction-lisibilite-carte.md) | Interaction & lisibilité de la carte | Clic gauche sélection/désélection, fond SVG désélection, clic gauche maintenu + déplacement pour déplacer, lisérés intérieurs de sélection et de contrôle | Tests/manœuvres manuelles : gestes, frontières et contrôle entre joueurs | Carte : suppression du remplissage de contrôle, lisérés intérieurs, aides et légende cohérentes | Fait |
| [P1.9](prompts/p1.9-validation-partielle-chaines.md) | Validation partielle des chaînes | Non-adjacence différée à l'exécution ; ordres précédents conservés, ordre fautif et suffixe postérieur brisés | Tests moteur : chaîne O1/O2 valides, O3 non adjacent, erreurs bloquantes inchangées | Réception affichée comme reçue puis brisée, non comme perdue | Prompt écrit |
| [P1.10](prompts/p1.10-poste-commandement-rapport.md) | Poste de commandement & rapport lisible | Onglets partagés, nobles présents, pile complète d'ordres, DTO d'ordre complet, codes territoriaux visibles et marqueurs joueur/noble | Tests front/API : onglets, pile, labels complets, résultats et messages sans matricule | Poste de commandement et rapport dans une même zone ; ordre courant distingué ; chaque ordre possède label, noble, couleur et résultat | Prompt écrit |
| [P1.11](prompts/p1.11-hiver-signalisation-bilan.md) | Signalisation de l'hiver & bilan des coûts | Thème hivernal distinct, teinte neige de la carte, coût total de chaque ordre d'hiver réussi dans le rapport | Tests moteur/front : coûts simples, multi-sources, gratuits et rejetés ; lisibilité saisonnière | Panneau d'ordres d'hiver et carte immédiatement identifiables | Prompt écrit |
| [P1.12](prompts/p1.12-villages-neutres-productifs.md) | Villages neutres productifs | Production et stockage neutres, conservation, capture sans perte du stock et disponibilité au nouveau contrôleur | Tests multi-tours : accumulation, non-disponibilité avant capture, capture, hiver, ravitaillement | Rapport capable de distinguer une source neutre sans propriétaire joueur | Prompt écrit |

### P1.2 détaillé — Carte Voronoï

Les raffinements P1.2a à P1.2g sont également suivis individuellement :

| Sous-tâche | Prompt | Statut |
|---|---|---|
| P1.2a — Vocabulaire troupe/armée | [prompt](prompts/p1.2a-vocabulaire-troupe-armee.md) | Fait |
| P1.2b — Villages neutres | [prompt](prompts/p1.2b-lieux-dits-villages.md) | Fait |
| P1.2c — Nommage des communes | [prompt](prompts/p1.2c-nommage-communes.md) | Fait |
| P1.2d — Géométrie et graphe | [prompt](prompts/p1.2d-geometrie-graphe.md) | Fait |
| P1.2e — Production vivrière | [prompt](prompts/p1.2e-production-vivriere.md) | Fait |
| P1.2f — État d'exemple | [prompt](prompts/p1.2f-etat-exemple.md) | Fait |
| P1.2g — Armée : une par territoire (owner + taille) | [prompt](prompts/p1.2g-armee-unique-territoire.md) | Fait |

1. Cellules Voronoï déterministes (seed) → territoires de tailles inégales.
2. Extraction des arcs géométriques : toute paire de territoires intérieurs partageant au moins `minSharedEdges = 3` arêtes de grille forme un arc.
3. **Frontières infranchissables** (crêtes montagneuses, marécages) : les arcs géométriques sont qualifiés franchissables ou infranchissables, sans être supprimés.
4. **Pas de routes au MVP :** aucun arc ne relie des territoires non adjacents géométriquement. Le qualificatif futur `route` pourra rendre franchissable une frontière infranchissable (pont, col), hors MVP.
5. **Invariant :** le sous-graphe des adjacences franchissables est connexe et chaque territoire a un degré compris entre 2 et son maximum de terrain : 3 en montagne, marécage ou colline ; 5 en plaine ou forêt. L'union des arcs franchissables et infranchissables est exactement le graphe géométrique planaire.
6. Attribution des terrains (plaine, forêt, colline, montagne, marécage) + placement de villages neutres rares, bien répartis (maximisation de la distance minimale).
7. **Nommage des territoires :** chaque territoire reçoit une commune non encore utilisée ; l'affinité correspondant à son terrain dominant est privilégiée, puis `any`, avec un repli déterministe. Son code est le trigramme de cette commune, unique sur la carte.
8. Tests : seed et nombre de joueurs → même carte, absorption des bords et padding, arcs tous géométriques et classés de manière exclusive, connexité et degrés par terrain, unicité trigrammes et codes de territoires, nombre et répartition des villages.

## P2 — Latence d'information (palier 2)

| ID | Tâche | Livrable | Critère de test | Front | Statut |
|---|---|---|---|---|---|
| [P2.1](prompts/p2.1-ia-sans-ordre.md) | IA "sans ordre" | Soutien défensif auto à l'armée alliée la plus proche la moins soutenue (défense ou Sans Ordre uniquement, puissance = armée) | Tests sur lignes de front | — | Prompt écrit |
| [P2.2](prompts/p2.2-messagers-vision-tx.md) | Messagers & rapports (vision T-x) | Rapports des troupes/châteaux/villages/tours (cases adjacentes, fraîcheur = émission), T0 (noble libre ou otage, tour de guet + adjacentes), projection des troupes ; `state.json` devient la vue par joueur (`asOf` par territoire) | Tests : fraîcheur d'information par case | Carte affichant l'état stale (T-x) via `asOf` + projection distincte ; fraîcheur rendue par pois de taille croissante, maximum à 3 tours, sans recolorer le terrain | Prompt écrit |
| [P2.2a](prompts/p2.2a-otages-intel.md) | Renseignement des otages | Divulgation privée et datée de la chaîne de l'armée geôlière par les nobles `hostage`, filtrée selon les rapports et la vision T-x | Tests : aucune chaîne ennemie non observée ne fuit ; `dungeon` ne divulgue rien | Chaînes connues distinguées des observations et projections | Prompt écrit |
| [P2.3](prompts/p2.3-transmission-ordres.md) | Transmission différée des ordres | Ordres partant du noble émetteur vers le 1er territoire de la feuille (arrivée fixée à l'émission), 1re troupe du territoire dans la fenêtre jusqu'à l'hiver, chaîne perdue sinon ; pas d'interception | Tests : délais, réception en fenêtre, perte à l'hiver | — | Prompt écrit |

## P3 — Serveur jouable (palier 3)

| ID | Tâche | Livrable | Critère de test | Front | Statut |
|---|---|---|---|---|---|
| [P3.0](prompts/p3.0-identifiants-territoires.md) | Contrat territorial par trigrammes | Suppression du double `TerritoryID`/`Code` au profit du trigramme unique dans le modèle, les contrats et les références | Tests de génération, résolution, JSON et déterminisme sans `T<number>` | Front et API n'exposent plus de matricule territorial | Prompt écrit |
| [P3.1](prompts/p3.1-api-rest.md) | API REST complète (chi) | Parties multiples, inscription, état, ordres, résolution, rapports | Tests API | Migration des endpoints dev vers l'API réelle | Prompt écrit |
| [P3.2](prompts/p3.2-auth-sessions.md) | Auth & sessions | Code d'invitation par partie, session simple | Deux joueurs jouent une partie | Écran partie + code d'invitation | Prompt écrit |
| [P3.3](prompts/p3.3-persistance-json.md) | Persistance JSON | 1 fichier par partie, sauvegarde/restauration | Redémarrage sans perte | — | Prompt écrit |
| [P3.4](prompts/p3.4-polish-front.md) | Polish front | Correction des retours de test local, dont le rendu final de la fraîcheur T-x | Parcours complet fluide | Parcours complet, badges et pois de fraîcheur : diamètre croissant avec l'âge, maximum à 3 tours et information signalée non fiable | Prompt écrit |
| [P3.5](prompts/p3.5-deploiement.md) | Déploiement | Cloud Run + Artifact Registry + CI auto | Lien public jouable entre amis | — | Prompt écrit |

### Détail P3 — choix actés

- **Ordre des travaux :** P3.0 doit être terminé avant P3.1. Les contrats REST ne doivent pas être conçus sur le double identifiant territorial abandonné.
- **API (P3.1) :** router **chi** (1re dépendance tierce autorisée) ; parties en mémoire ; soumission **par joueur** ; résolution **synchrone à la dernière soumission** (pas de deadline au MVP) ; endpoints : POST/GET /api/games, GET /api/games/{id}, /map, /state (vue par joueur P2.2), POST /orders, /reports ; endpoints dev P1.7 conservés.
- **Élimination / victoire (P3.1 — acté, GDD §2) :** un joueur est éliminé quand il ne contrôle aucun territoire ET n'a plus aucune armée (les nobles, immortels, ne comptent pas) ; dernier joueur vivant = gagnant.
- **Auth (P3.2) :** inscription sans mot de passe (nom + token Bearer en localStorage), sessions en mémoire (perdues au restart — reprise de slot par nom + code d'invitation, même en partie commencée), code d'invitation 6 caractères par partie (le créateur = P1, join tant que la partie n'est pas commencée, 5 joueurs max), accès 403 hors membres, `?player=` retiré (identité par token).
- **Persistance (P3.3) :** 1 fichier JSON par partie (`game-<uuid>.json` dans `DATA_DIR`), écriture atomique (tmp + rename + fsync), sauvegarde après chaque mutation (soumissions comprises), restauration au démarrage, fichier corrompu → `.corrupt` (serveur démarre).
- **Déploiement (P3.5) :** un seul conteneur (front `go:embed web/dist` + API, same-origin, pas de CORS) ; Cloud Run + Artifact Registry + bucket GCS monté en gcsfuse sur `/data` (DATA_DIR) ; CI : tests/build → push AR (tag sha) → `gcloud run deploy` via workload identity federation ; pas de Terraform au MVP.
