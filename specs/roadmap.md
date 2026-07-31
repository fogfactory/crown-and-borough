# Roadmap d'Implémentation : Crown & Borough (MVP)

Plan d'implémentation pour atteindre le plus rapidement possible un prototype testable.
Sources : `specs/gdd.md` (règles) et `specs/architecture.md` (stack).

## Stratégie globale

Trois paliers de "testable" successifs, chacun validé avant le suivant :

1. **Palier 1 — Moteur testable :** règles du GDD en Go pur, validées par `go test` (aucune infra).
2. **Palier 2 — Latence testable :** 2e pilier (messagers, vision T-x), toujours en Go pur.
3. **Palier 3 — Jouable entre amis :** API + persistance + front + déploiement Cloud Run.

**Décisions actées :**

- La latence d'information est différée au palier 2 : la boucle de jeu démarre en vision T0 partout.
- La carte est générée par **algorithme de Voronoï seedé** : territoires inégaux, graphe **connexe et sans cul-de-sac sur toutes les cellules, bords compris** (degré ≥ 2 partout). Les adjacences géométriques ne sont pas toutes franchissables (frontières infranchissables : crêtes, marécages) et des **routes** relient des territoires non adjacents géométriquement (le graphe final n'est pas planaire).
- Persistance : **JSON d'abord** (1 fichier par partie), migration **Postgres/sqlc** ensuite.
- Le **front est construit au fur et à mesure** du plan pour tester localement dès que possible.

## P0 — Fondations

| ID | Tâche | Livrable | Critère de test | Front |
|---|---|---|---|---|
| P0.1 | Squelette repo Go | Structure standard, go.mod, Makefile, Dockerfile multi-stage, CI GitHub Actions | `make build` vert, CI verte | — |
| P0.2 | Assets CSV + loader Go | communes.csv, prenoms.csv (médiévaux français), qualificatifs.csv + codes trigrammes uniques par catégorie ; loader Go vérifiant les invariants | Tests unitaires : unicité par fichier, unicité des préfixes, terrains valides | — |
| P0.3 | Scaffold front | Vite + React + TypeScript + Tailwind + shadcn/ui | App dev se lance | MapViewer v0 sur fixtures `map.ts` (statique) + `state.ts` (dynamique) : fond + couche vivante superposés |

## P1 — Moteur de jeu (palier 1)

| ID | Tâche | Livrable | Critère de test | Front |
|---|---|---|---|---|
| P1.1 | Modèles métier | Territoire, Armée, Noble, Infrastructure, Ressources, GameState | Compile + tests modèles | — |
| P1.2 | Génération de carte Voronoï | Graphe d'adjacence seedé, terrains, ~25 % lieux-dits, nommage des territoires | Carte déterministe par seed, invariants (connexité, degré ≥ 2, unicité trigrammes, % lieux-dits) | Endpoint dev `/api/map` (map.json statique) → la vraie carte s'affiche |
| P1.3 | Ordres & chaînes | Parser texte des chaînes (format specs/architecture.md §6), 6 symboles A/S/H/J/P/D, liaison par transition (single/loop), chaîne reçue par l'armée sur le territoire FROM de la 1re ligne | Tests : parsing, progression O1→O2→O3, single/loop par transition, invalide brise toujours, capacité noble 1/tour, validation par type | — |
| P1.4 | Résolution mouvement & combat | Mouvement, attaque, maintien, soutien, jonction, séparation, retraites | Scénarios de combat (égalité, supériorité, retraites, destructions) | — |
| P1.5 | Ravitaillement & famine | Coût 2^(N-1), flux BFS portée 3/5, stocks, algorithme famine | Tests : déficits, ordre d'épuisement, famine | — |
| P1.6 | Phase d'Hiver | Conservation 50 %, recrutement Nobles, construction Infrastructures | Test : année complète sans perte | — |
| P1.7 | Rapport de tour + intégration | TurnReport, simulation d'une année complète multi-joueurs | Test de bout en bout d'une année type | Endpoints dev `/api/map` + `/api/state` + `/api/orders` → boucle locale hotseat : ordres via le front, résolution, rapport affiché |

### P1.2 détaillé — Carte Voronoï

1. Cellules Voronoï déterministes (seed) → territoires de tailles inégales.
2. Extraction du graphe d'adjacence géométrique.
3. **Frontières infranchissables** (crêtes montagneuses, marécages) → suppression d'adjacences.
4. **Routes** reliant des territoires non adjacents géométriquement (cols, ponts).
5. **Invariant :** le graphe final (adjacences franchissables + routes) est connexe, sans cul-de-sac sur aucune cellule, bords compris (degré ≥ 2 partout). Les cellules de bord forment un anneau ; toute feuille résiduelle est corrigée par ajout de route.
6. Attribution des terrains (plaine, forêt, colline, montagne, marécage) + répartition ~25 % lieux-dits.
7. **Nommage des territoires :** chaque territoire reçoit un qualificatif de `qualificatifs.csv` selon son terrain dominant (+ "Marches" en bordure) et le nom d'une commune adjacente → "Forêt de Rosemont" / code `FROS` (préfixe + trigramme commune, 4 lettres).
8. Tests : seed → même carte, connexité, degrés, unicité trigrammes et codes de territoires, % lieux-dits.

## P2 — Latence d'information (palier 2)

| ID | Tâche | Livrable | Critère de test | Front |
|---|---|---|---|---|
| P2.1 | IA "sans ordre" | Soutien auto à l'allié le plus proche le moins soutenu | Tests sur lignes de front | — |
| P2.2 | Messagers & rapports (vision T-x) | Propagation des rapports, vision T0 (capitale, noble, tour de guet) ; `state.json` devient la vue par joueur (`asOf` par territoire) | Tests : fraîcheur d'information par case | Carte affichant l'état stale (T-x) via `asOf` |
| P2.3 | Transmission différée des ordres | Ordres voyageant à vitesse terrain, poursuite ancienne chaîne | Tests : délais, interception de chaîne | — |

## P3 — Serveur jouable (palier 3)

| ID | Tâche | Livrable | Critère de test | Front |
|---|---|---|---|---|
| P3.1 | API REST complète (chi) | Parties multiples, inscription, état, ordres, résolution, rapports | Tests API | Migration des endpoints dev vers l'API réelle |
| P3.2 | Auth & sessions | Code d'invitation par partie, session simple | Deux joueurs jouent une partie | Écran partie + code d'invitation |
| P3.3 | Persistance JSON | 1 fichier par partie, sauvegarde/restauration | Redémarrage sans perte | — |
| P3.4 | Polish front | Correction des retours de test local | Parcours complet fluide | — |
| P3.5 | Déploiement | Cloud Run + Artifact Registry + CI auto | Lien public jouable entre amis | — |
