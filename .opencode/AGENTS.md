# AGENTS.md — Crown & Borough

Instructions pour les agents travaillant sur ce dépôt.

## Contexte projet

Jeu de stratégie asynchrone par tours sur carte (MVP). Backend Go monolithique,
front React/Vite/TypeScript. Toute la conception vit dans `specs/` :

- `specs/gdd.md` — règles du jeu (la source de vérité du gameplay)
- `specs/architecture.md` — stack, structure du repo, contrats map.json/state.json
- `specs/roadmap.md` — état de la v1 et évolutions suivies par issues GitHub
- `specs/prompts/` — prompts online `p3.x` conservés temporairement comme
  matériau de travail ; les prompts livrés ou abandonnés ont été supprimés

Avant d'implémenter une règle de jeu, lire le GDD. Avant de toucher la stack
ou les contrats d'API, lire l'architecture. Toute décision de conception
nouvelle ou modifiée doit être reportée dans les specs correspondantes.

## Workflow Git

- `main` est la branche de release protégée. Les correctifs, améliorations UX,
  promotions depuis `develop` et PRs release-please y sont intégrés après revue.
- `develop` est la branche d'intégration protégée. Les features et changements
  de maintenance y sont intégrés et elle n'est jamais déployée directement.
- Utiliser `fix/*` ou `bugfix/*` pour un correctif vers `main`, `ux/*` pour une
  amélioration UX vers `main`, et `feat/*` ou `feature/*` pour une feature vers
  `develop`.
- Ne jamais pousser directement sur `main` ou `develop`. Toujours passer par
  une PR avec revue et CI verte.
- Les titres de PR suivent `type(scope): description`. `fix` produit un patch,
  `feat` une mineure, et `!` ou un footer `BREAKING CHANGE` une majeure. Les
  commits de maintenance seuls ne créent pas de release sauf s'ils sont inclus
  dans le changelog release-please.
- Les PRs normales sont fusionnées en squash en conservant le titre de la PR
  comme sujet du commit; les promotions de `develop` vers `main` et la
  synchronisation font exception avec des merge commits pour que
  release-please voie les Conventional Commits individuels.
- Ajouter exactement un label `release:patch`, `release:minor` ou
  `release:major` aux changements destinés à une release. La policy vérifie la
  cohérence du label avec le titre Conventional Commit; release-please calcule
  le bump effectif.
- Ne jamais créer de tag de production manuellement. release-please crée le
  tag SemVer et la GitHub Release après fusion de sa PR.
- La synchronisation de `main` vers `develop` est automatisée. Ne pas créer de
  seconde PR de synchronisation; résoudre la PR générée en cas de conflit.

## Workflow : issue → spécification → implémentation

Les nouveaux bugs et fonctionnalités sont suivis dans GitHub. Avant de coder,
lire `specs/gdd.md` pour les règles et `specs/architecture.md` pour les
contrats. Toute décision qui modifie le cœur v1 doit être reportée dans ces
documents. Les prompts `p3.x` ne sont pas un nouveau système de suivi : ils
seront remplacés par des issues au fur et à mesure de leur réalisation.

À la clôture d'une issue, vérifier les critères d'acceptation, les tests et les
spécifications, puis mettre à jour `specs/roadmap.md` si le périmètre produit
change.

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
  `InfraType` = mill/supply_depot/castle/village.

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

- Carte : Voronoï seedé, graphe connexe **sans cul-de-sac sur aucune cellule,
  bords compris** (degré ≥ 2 partout) ; adjacences géométriques non toutes
  franchissables + routes reliant des territoires non adjacents.
- Chaque territoire porte le nom d'une commune ; son affinité de terrain est
  privilégiée avec un repli déterministe. Le code territorial est le trigramme
  de la commune, à trois lettres et unique sur la carte ; les codes de communes
  et prénoms restent uniques dans leur catégorie.
- La session v1 est en mémoire ; la persistance JSON et les parties multiples
  sont suivies par issue.
- Contrats front : `map.json` statique/public vs `state.json` dynamique ; les
  chaînes et combats auront une projection privée côté serveur selon le GDD.
- Moteur de résolution : fonction pure `Resolve(state, orders) -> (state, report)`.
