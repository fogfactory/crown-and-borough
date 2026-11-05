# CLAUDE.md — Crown & Borough

Ce fichier configure Claude pour ce dépôt. Les instructions de référence sont
partagées avec les autres agents (opencode) et importées ci-dessous : les
modifier là-bas, pas ici.

@AGENTS.md
@.opencode/AGENTS.md

## Spécificités Claude

- « ask_user » dans les instructions ci-dessus = l'outil **AskUserQuestion** :
  l'utiliser dès qu'une décision non triviale touche l'architecture ou le
  game design.
- Sous-agents disponibles (`.claude/agents/`) :
  - `explore-files` — repérage low-cost des fichiers pertinents avant un
    plan ou une implémentation (lecture seule).
  - `review` — revue du code, des plans et des specs au regard des
    conventions du projet.
- Serveur MCP `semble` (recherche sémantique de code) déclaré dans `.mcp.json`.
- Ne jamais commiter ni pousser sans demande explicite ; quand elle est
  faite, suivre les règles de commit/push de `AGENTS.md`.
