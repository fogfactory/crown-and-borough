# AGENTS.md — Crown & Borough

Instructions pour les agents travaillant sur ce dépôt.

## Contexte projet

Jeu de stratégie asynchrone par tours sur carte (MVP). Backend Go monolithique,
front React/Vite/TypeScript. Toute la conception vit dans `specs/` :

- `specs/gdd.md` — règles du jeu (la source de vérité du gameplay)
- `specs/architecture.md` — stack, structure du repo, contrats map.json/state.json
- `specs/roadmap.md` — plan d'implémentation (paliers P0→P3) et décisions actées
- `specs/prompts/` — un fichier markdown par tâche du roadmap, prêt à copier
  dans un agent ; à tenir à jour quand une tâche change

Avant d'implémenter une règle de jeu, lire le GDD. Avant de toucher la stack
ou les contrats d'API, lire l'architecture. Toute décision de conception
nouvelle ou modifiée doit être reportée dans les specs correspondantes.

## Conventions de code

- **Code exclusivement en anglais** : identifiants, noms de fichiers,
  commentaires, messages d'erreur/log, valeurs d'enums.
- Seules les chaînes de **contenu de jeu** sont en français (noms de communes,
  prénoms, qualificatifs, labels UI) — le jeu est en français.
- Backend Go : stdlib-first, pas d'ORM (sqlc prévu), structure
  `cmd/server` + `internal/{api,engine,db,models}`, module
  `github.com/fogfactory/crown-and-borough`.
- Front : TypeScript strict, composants shadcn/ui, carte en SVG.
- Enums alignés : `Terrain` = plain/forest/hill/mountain/swamp,
  `Season` = spring/summer/autumn/winter,
  `InfraType` = mill/post_relay/watchtower/supply_depot/castle.

## Conventions git

- **Conventional Commits** : `type(scope): description` — types usuels :
  `feat`, `fix`, `docs`, `chore`, `test`, `refactor`, `ci`, `build`.
  Messages en anglais, minuscules, pas de point final.
  Exemples : `docs(prompts): add p1.2 map generation prompt`,
  `feat(engine): resolve combat and retreats`.
- **Un commit = une chose précise** : ne pas mélanger des changements sans
  rapport (une tâche de roadmap = un commit ; un prompt = un commit).
- Ne **jamais commiter sans demande explicite** de l'utilisateur.
- Pas de force-push sans demande explicite.
- Avant de commiter : vérifier `git status`/`git diff`, ne staguer que les
  fichiers concernés, ne jamais committer de secrets.

## Commandes

- `make build` / `make run` / `make test` / `make vet` / `make clean`
- `make web-dev` — front Vite en dev (dossier `web/`)
- `npm run dev|build|lint` dans `web/`

## Décisions de conception actées

- Latence d'information (messagers, vision T-x) différée au palier P2 ;
  le MVP démarre en vision T0 partout.
- Carte : Voronoï seedé, graphe connexe **sans cul-de-sac sur aucune cellule,
  bords compris** (degré ≥ 2 partout) ; adjacences géométriques non toutes
  franchissables + routes reliant des territoires non adjacents.
- Noms de territoires générés à la carte à partir des communes adjacentes et
  des qualificatifs (`qualificatifs.csv`) ; codes 4 lettres (préfixe +
  trigramme commune) ; codes communes/prénoms : trigrammes stricts 3 lettres,
  uniques par catégorie.
- Persistance JSON d'abord (1 fichier par partie), Postgres/sqlc ensuite.
- Contrats front : `map.json` statique/public vs `state.json` dynamique/privé
  (vue par joueur, horodatage `asOf` par territoire en P2).
- Moteur de résolution : fonction pure `Resolve(state, orders) -> (state, report)`.
