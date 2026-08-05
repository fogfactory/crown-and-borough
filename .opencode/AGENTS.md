# AGENTS.md — Crown & Borough

Instructions pour les agents travaillant sur ce dépôt.

## Contexte projet

Jeu de stratégie asynchrone par tours sur carte (MVP). Backend Go monolithique,
front React/Vite/TypeScript. Toute la conception vit dans `specs/` :

- `specs/gdd.md` — règles du jeu (la source de vérité du gameplay)
- `specs/architecture.md` — stack, structure du repo, contrats map.json/state.json
- `specs/roadmap.md` — plan d'implémentation (paliers P0→P3), décisions actées
  et suivi d'exécution par statut
- `specs/prompts/` — phase de conception : un fichier markdown par tâche du
  roadmap, référencé dans `specs/roadmap.md` et à tenir à jour quand la tâche
  change
- `specs/plans/` — un plan d'implémentation détaillé par prompt de
  `specs/prompts/`, écrit au format `<palier>-<tâche>-implementation-plan.md`
  (ex. `p0.1-implementation-plan.md`)

Avant d'implémenter une règle de jeu, lire le GDD. Avant de toucher la stack
ou les contrats d'API, lire l'architecture. Toute décision de conception
nouvelle ou modifiée doit être reportée dans les specs correspondantes.

## Workflow : prompt → plan → évaluation → implémentation

Toute tâche suit ce workflow dans l'ordre. `specs/roadmap.md` est le document
de suivi d'exécution : chaque création, transition et clôture y est reportée.

1. **Prompt** (`specs/prompts/<id>-<slug>.md`) :
   - créer un fichier par tâche, avec un `<id>` identique à celui du roadmap ;
   - décrire le contexte, le périmètre, les étapes, les critères d'acceptation
     et les tests attendus ;
   - lister explicitement les specs à mettre à jour si la tâche les impacte,
     notamment `specs/gdd.md`, `specs/architecture.md` et
     `specs/roadmap.md` ;
   - plusieurs prompts peuvent être créés à la suite avant de passer à la
     phase de planification ;
   - mettre à jour `specs/roadmap.md` immédiatement : chaque prompt doit y
     être référencé et passer au statut `Prompt écrit`.
2. **Plan** (`specs/plans/<id>-implementation-plan.md`) :
   - partir du prompt et le raffiner avec les questions à l'utilisateur
     (`ask_user`) et le modèle de planification haut de gamme ;
   - écrire le plan TOUJOURS dans `specs/plans/`, jamais dans
     `specs/prompts/` ni ailleurs ;
   - inclure les fichiers concernés, les étapes d'implémentation, les tests,
     les mises à jour de specs et les risques ;
   - inclure une section « Difficulté et modèle recommandé » avec une note de
     difficulté sur 5 et un provider/modèle précis pour l'exécution ;
   - passer le statut du roadmap à `Plan prêt` quand le plan est exploitable.
3. **Évaluation et délégation** :
   - évaluer la difficulté, les risques et les critères du plan avant de
     déléguer ;
   - choisir le modèle le moins cher capable de réussir la tâche, plutôt que
     d'utiliser systématiquement le modèle le plus puissant ;
   - difficulté 1-2 : modèle léger ; difficulté 3 : modèle intermédiaire ;
     difficulté 4-5 : modèle haut de gamme ;
   - passer le statut du roadmap à `En cours` au début de l'implémentation.
4. **Implémentation** : utiliser le plan comme brief d'exécution. Toute
   déviation nécessaire doit d'abord être répercutée dans le plan et, si elle
   modifie une décision de conception, dans les specs concernées.
5. **Clôture** : vérifier les critères d'acceptation, les tests et les
   changements de documentation, puis passer la tâche à `Fait` dans
   `specs/roadmap.md`. En cas de blocage, utiliser `Bloqué` et documenter la
   cause dans le roadmap ou le plan.

### Statuts du roadmap

- `Prompt écrit` — le prompt existe dans `specs/prompts/`, sans plan finalisé
- `Plan prêt` — le plan existe dans `specs/plans/` et peut être délégué
- `En cours` — l'implémentation a été déléguée ou est en cours
- `Fait` — l'implémentation et ses vérifications sont terminées
- `Bloqué` — une décision, une dépendance ou une correction empêche la suite

## Conventions de code

- **Code exclusivement en anglais** : identifiants, noms de fichiers,
  commentaires, messages d'erreur/log, valeurs d'enums.
- Seules les chaînes de **contenu de jeu** sont en français (noms de communes,
  prénoms, labels UI) — le jeu est en français.
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
- Chaque territoire porte le nom d'une commune ; son affinité de terrain est
  privilégiée avec un repli déterministe. Le code territorial est le trigramme
  de la commune, à trois lettres et unique sur la carte ; les codes de communes
  et prénoms restent uniques dans leur catégorie.
- Persistance JSON d'abord (1 fichier par partie), Postgres/sqlc ensuite.
- Contrats front : `map.json` statique/public vs `state.json` dynamique/privé
  (vue par joueur, horodatage `asOf` par territoire en P2).
- Moteur de résolution : fonction pure `Resolve(state, orders) -> (state, report)`.
