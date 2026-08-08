# Spécifications

Les fichiers de ce dossier sont organisés par thème, pas par version de
livraison. Les versions pourront être attribuées plus tard aux milestones
GitHub. Chaque thème possède une spécification de référence et un milestone
correspondant ; les dépendances sont indiquées ci-dessous pour faciliter le
réordonnancement.

## Socle actuel

| Thème | Référence | Suivi GitHub |
|---|---|---|
| Ravitaillement et famine | [`gdd.md`](gdd.md), sections 3 et 5 ; [`ravitaillement.md`](ravitaillement.md) | [Milestone v1](https://github.com/fogfactory/crown-and-borough/milestone/1) |
| Architecture | [`architecture.md`](architecture.md) | [Milestone v1](https://github.com/fogfactory/crown-and-borough/milestone/1) |
| Online | [`online.md`](online.md), [`prompts/`](prompts/) | [Issue #2](https://github.com/fogfactory/crown-and-borough/issues/2), [Milestone v1](https://github.com/fogfactory/crown-and-borough/milestone/1) |
| Tests live | [`hotseat-live-test.md`](hotseat-live-test.md) | [Milestone v1](https://github.com/fogfactory/crown-and-borough/milestone/1) |

## Thèmes futurs

| Thème | Spécification | Milestone | Dépendances indicatives |
|---|---|---|---|
| Titres et victoire | [`titres.md`](titres.md) | [Titres & Victoire](https://github.com/fogfactory/crown-and-borough/milestone/2) | Socle actuel |
| Cartographie | [`cartographie.md`](cartographie.md) | [Cartographie](https://github.com/fogfactory/crown-and-borough/milestone/3) | Indépendant, mais utile au thème religieux |
| Ordres spéciaux | [`ordres-speciaux.md`](ordres-speciaux.md) | [Ordres spéciaux & Calamités](https://github.com/fogfactory/crown-and-borough/milestone/4) | Socle actuel |
| Religieux | [`religieux.md`](religieux.md) | [Religieux](https://github.com/fogfactory/crown-and-borough/milestone/5) | Titres, Cartographie |
| Politique royale | [`politique.md`](politique.md) | [Politique royale](https://github.com/fogfactory/crown-and-borough/milestone/6) | Titres, Religieux |
| Succession | [`succession.md`](succession.md) | [Succession](https://github.com/fogfactory/crown-and-borough/milestone/7) | Titres, Religieux, Politique, cartes spéciales |
| Économie et prospérité | [`economie.md`](economie.md) | [Économie & Prospérité](https://github.com/fogfactory/crown-and-borough/milestone/8) | Socle actuel ; Titres pour les seigneuries |
| Information | [`information.md`](information.md) | [Brouillard de guerre](https://github.com/fogfactory/crown-and-borough/milestone/9) | Issue #2 et vue privée online |

## Dépendances proposées

```text
Socle actuel
├── Cartographie ───────────────┐
├── Économie et prospérité      │
├── Titres et victoire ─────────┼── Religieux ─── Politique royale ─── Succession
└── Ordres spéciaux             │       │              │                    │
                                └───────┴──────────────┴────────────────────┘

Issue #2 / vue privée online ─── Brouillard de guerre
```

Ce graphe est une recommandation de planification, pas un ordre de sortie
définitif. Les milestones restent réordonnables dans GitHub.

## Documents différés

La transmission différée des ordres a été retirée du périmètre actif. Les
anciens prompts associés restent supprimés ; [`transmission-differee.md`](transmission-differee.md)
conserve uniquement la décision et le périmètre à reprendre éventuellement.

## Conventions

- Une règle déjà livrée reste décrite dans `gdd.md` et `architecture.md`.
- Une règle future est décrite dans son fichier thématique et planifiée dans
  GitHub.
- Une issue doit renvoyer vers sa spécification thématique ; le milestone doit
  renvoyer vers cette même spécification.
- Les questions encore ouvertes doivent rester visibles dans la spécification
  ou dans l'issue, plutôt que d'être résolues implicitement dans le code.
