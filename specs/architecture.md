# Architecture : Crown & Borough (MVP)

Pour faire tourner un prototype rapidement avec une demi-douzaine d'amis tout en gardant une base propre, lisible et facile à faire produire par des agents, la meilleure approche est un **monolithe modulaire en Go** hébergé sur **GCP Cloud Run**.

Go est idéal ici : typage strict, compilation rapide, conteneurs ultra-légers et excellente gestion des structures de données (graphes pour la carte, algorithmes de résolution).

---

## 1. Architecture Globale

Le modèle de jeu est asynchrone à résolution par tours. Il n'y a pas besoin de WebSockets complexes à maintenir en état constant : une API REST/HTTP classique avec du *polling* léger ou des requêtes simples suffit amplement pour le MVP.

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (Web)                       │
│        React / Vite + SVG / Tailwind (PWA)              │
└────────────────────────────┬────────────────────────────┘
                             │ REST API / JSON
┌────────────────────────────▼────────────────────────────┐
│                  Backend (Go Monolith)                  │
│  ┌──────────────┐  ┌──────────────────┐  ┌───────────┐  │
│  │ API / Auth   │  │ State Machine    │  │ Resolution│  │
│  │ (Gin / Chi)  │  │ & Engine (Graphe)│  │ Engine    │  │
│  └──────────────┘  └──────────────────┘  └───────────┘  │
└────────────────────────────┬────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────┐
│              Persistance & Infra                        │
│   Fichiers JSON (1 fichier / partie, volume GCS)        │
│   + GCP Cloud Run — PostgreSQL en migration ultérieure  │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Stack Technique Recommandée

### Backend : Go

- **Router / API :** `chi` ou `gin`. `chi` est extrêmement minimaliste, idiomatique et lisible.
- **Persistance :** fichiers **JSON au MVP** (1 fichier par partie, écriture atomique) — migration **PostgreSQL/`sqlc` différée** (cf. roadmap). `sqlc` génère du code Go type-safe à partir de requêtes SQL brutes, sans magie noire d'ORM, quand le besoin s'en fera sentir.
- **Moteur de résolution (Core Engine) :**
  - Un package pur Go sans dépendances web (`pkg/engine`).
  - Les territoires sont représentés sous forme de **graphe d'adjacence**.
  - La résolution d'un tour est une fonction pure : `Resolve(CurrentState, Orders) -> (NextState, TurnReport)`. Cela facilite énormément les tests unitaires automatisés.

### Frontend : React + TypeScript + SVG Interactif

Puisque le front sera produit par des agents, TypeScript + React est le combo sur lequel les LLM sont le mieux entraînés.

- **Bundler & Framework :** **Vite + React (TypeScript)**.
- **Rendu de la Carte :** **SVG Interactif**. Pour une carte de type *Diplomacy/Fief* basée sur des graphes/territoires :
  - Pas de Canvas/Phaser/Three.js au début.
  - Une carte en SVG où chaque territoire est un composant (`<path id="TER_01" onClick={...} />`) est ultra simple à manipuler en React, réactive, zoomable nativement et parfaitement lisible pour un agent.
- **UI & Styles :** **Tailwind CSS** + **shadcn/ui**. Rendu propre, responsive (mobile/PC) et accessible sans écrire de CSS complexe.

---

## 3. Déploiement & Infrastructure (GCP)

Pour limiter les coûts et la complexité d'exploitation :

1. **Calcul : GCP Cloud Run**
   - Image Docker minimale (Go multi-stage build → conteneur `scratch` ou `alpine` de ~15 Mo).
   - **Scale to zero :** réduction des coûts à 0 € quand personne ne joue.

2. **Persistance : fichiers JSON**
   - **1 fichier par partie** (`DATA_DIR`, écriture atomique tmp + rename), monté sur un **bucket GCS** (gcsfuse) en production pour survivre aux restarts Cloud Run.
   - **Cloud SQL for PostgreSQL** (ou Neon / Supabase) : migration ultérieure, hors MVP.

3. **CI/CD : GitHub Actions**
   - Un workflow simple : à chaque push sur `main`, build du conteneur, push sur GCP Artifact Registry et déploiement automatique sur Cloud Run.

---

## 4. Organisation du Projet (Repository Go)

Une structure standard (*Standard Go Project Layout*) simple et efficace pour la lisibilité :

```text
.
├── cmd/
│   └── server/          # Main entrypoint (HTTP server)
├── internal/
│   ├── api/             # Handlers HTTP, middleware, routes
│   ├── engine/          # Moteur de jeu pur (Graphes, Résolution, Combat, Famine)
│   ├── db/              # Chargement des assets (CSV, balance.json) — sqlc/Postgres différé
│   ├── models/          # Structures de données métier (Ordres, Territoires, Armées)
│   └── server/          # Sessions de jeu, store des parties, persistance JSON (P3)
├── assets/              # Données statiques du monde et équilibrage
│   ├── communes.csv     # Noms de lieux-dits
│   ├── qualificatifs.csv # Qualificatifs de territoires (prefixe + terrain)
│   ├── prenoms.csv      # Prénoms de nobles
│   └── balance.json     # Toutes les constantes d'équilibrage du moteur
├── web/                 # Application Frontend React (Vite)
├── Dockerfile           # Multi-stage build (build front + build Go)
└── Makefile             # Commandes de dev (run, test, migrate)
```

### Assets (Données statiques du monde)

Les fichiers `assets/*.csv` fournissent les noms et codes identifiants des entités du monde, embarqués dans le conteneur et chargés au démarrage du serveur :

| Fichier | Contenu | Utilisation |
| --- | --- | --- |
| `communes.csv` | Noms générés de communes | Nommer les lieux-dits |
| `qualificatifs.csv` | Qualificatifs de territoires (lettre + terrain d'application) | Nommer les territoires, à la génération de carte |
| `prenoms.csv` | Prénoms générés | Nommer les Nobles |

**Codes identifiants uniques et humainement lisibles :** chaque entité reçoit un code court (trigramme strict de 3 lettres, ex. `VIL` pour Villeneuve) aligné sur le *départage alphabétique des trigrammes* du GDD. Les codes sont uniques **au sein de chaque catégorie** (communes entre elles, prénoms entre eux, préfixes de qualificatifs entre eux) ; le type d'entité lève toute ambiguïté entre catégories. Ces codes sont utilisés dans les ordres et les rapports pour que les joueurs (et les agents) puissent référencer une entité sans avoir à retenir un UUID ou un nom complet. Les UUID internes restent la clé primaire ; les codes servent d'identifiant d'usage dans l'UI et les messages.

**Noms de territoires générés à la carte :** les territoires ne sont pas nommés dans les assets. À la génération de carte (P1.2), chaque territoire reçoit un qualificatif selon son terrain dominant (`qualificatifs.csv`) et le nom d'une commune adjacente : *"Forêt de Rosemont"* → code `F` + `ROS` = `FROS`. Les codes des territoires non lieux-dits font donc 4 lettres (les lieux-dits gardent le trigramme de leur commune, 3 lettres) et restent uniques par construction (préfixes et trigrammes uniques).

**Format attendu (CSV simple, en-tête en première ligne) :**

```csv
# communes.csv / prenoms.csv
code;nom
VIL;Villeneuve
ROS;Rosemont
GRI;Griffecourt
```

```csv
# qualificatifs.csv — prefix:une lettre | terrain: plain|forest|hill|mountain|swamp|any
prefix;qualificatif;terrain
F;Forêt;forest
M;Monts;mountain
```

**Convention de code :** le code est exclusivement en anglais (identifiants, commentaires, messages, valeurs d'enums) ; seules les chaînes de contenu de jeu (noms, labels UI) sont en français.

---

## 5. Contrats Serveur → Front : map.json / state.json

Le front consomme **deux documents JSON distincts**, séparant l'immobile du vivant :

### map.json (statique, constant, public)

La géographie est une connaissance commune : aucun joueur ne peut ignorer le terrain. Ce document est identique pour tous et ne change jamais après la génération de la partie.

```json
{
  "territories": [
    {
      "id": "T01",
      "code": "FROS",
      "name": "Forêt de Rosemont",
      "terrain": "forest",
      "lieuDit": true,
      "points": [[x, y], [x, y], ...],
      "adjacencies": ["T02", "T05"]
    }
  ]
}
```

- `points` : polygone de la forme (géométrie du territoire)
- `adjacencies` : graphe d'adjacence franchissable (toutes les frontières ne sont pas franchissables)

### state.json (dynamique, privé, par joueur et par fraîcheur de rapport)

Ce qui vit sur la carte : armées, infrastructures, contrôle, ressources. C'est la **vue** d'un joueur : en P2, l'API ne sert que ce que ses messagers ont rapporté (vision T-x), chaque territoire est horodaté (fraîcheur).

```json
{
  "turn": 12,
  "season": "spring",
  "asOf": { "T01": 11, "T02": 9, ... },
  "territories": [
    {
      "id": "T01",
      "owner": "P1",
      "resources": 4,
      "armies": [{ "id": "A1", "owner": "P1" }],
      "infrastructures": [{ "type": "mill", "level": 2 }]
    }
  ],
  "nobles": [{ "id": "N1", "name": "Hugues", "owner": "P1", "location": "T01" }]
}
```

- `asOf` : tour auquel chaque territoire a été observé (permet d'afficher la fraîcheur de l'information en P2)
- `armies`, `infrastructures`, `nobles`, `owner`, `resources` : couche dynamique
- **Coûts de trajet** (rapports P2.2 et ordres P2.3) : coût par case traversée selon le terrain, lu dans `assets/balance.json` (section travel : plaine/route 0,5 — 2 cases/tour ; forêt/colline 1 ; montagne/marécage 2 ; divisé par 2 sur un Relais de Poste)

### Rendu côté front

Le front combine les deux documents : `map.json` fournit géométrie et fond (terrains, lieux-dits, labels), `state.json` fournit la couche vivante (armées, constructions, contrôle, stocks). Cette séparation permet de **rendre plusieurs versions d'un même rapport** (comparer la vue fraîche et la vue datée), le calcul d'affichage restant entièrement côté client.

---

## 6. Format des chaînes d'ordres (texte)

Les joueurs (et les agents) expriment les chaînes d'ordres en texte brut, facile à taper. Une chaîne = un en-tête (le noble qui émet) + une ligne par ordre.

### Syntaxe

```
JEA # Jean                     <- en-tête : code trigramme du noble (commentaire ignoré)
BRI A ATL # attaque single
BRI S ATL - NOR # soutien offensif (la pile de BRI soutient l'attaque ATL → NOR), single
(BRI S ATL) # soutien défensif (soutient la tenue de la pile d'ATL), boucle
(ATL A NOR) # déplacement en boucle
H BRI      # maintien, single
BRI J ROS  # jonction, single — toujours en DERNIER ordre d'une chaîne
P BRI      # pillage de la case de l'armée, single
BRI D BRI ATL NOR  # dispersion : 1 destination par armée de la pile
```

- **En-tête (1re ligne)** : code trigramme du noble émetteur. Une chaîne = une **émission** (capacité : 1 émission par noble et par tour). **Pas de modification de chaîne** : une pile qui reçoit une chaîne remplace la précédente. Une chaîne s'applique à une **pile entière** (jamais d'ordres mixtes) ; elle est portée par l'armée au plus petit matricule de la pile.
- **Ordres** : chaque ligne explicite la **position** de l'armée (le territoire où elle doit se trouver à l'exécution) — jamais implicite. Formats par symbole :
  - `XXX A YYY` — atteindre/attaquer depuis XXX vers YYY
  - `XXX S YYY` — soutien **défensif** depuis XXX vers YYY (YYY adjacente, YYY ≠ XXX)
  - `XXX S YYY - ZZZ` — soutien **offensif** : la pile de XXX soutient l'attaque de la pile de YYY vers ZZZ (ZZZ adjacente à XXX, YYY–ZZZ adjacentes)
  - `H XXX` — maintien sur XXX (position de l'armée)
  - `XXX J YYY` — jonction depuis XXX vers YYY (**toujours en dernier ordre d'une chaîne**)
  - `P XXX` — pillage, XXX = case où l'armée se trouve
  - `XXX D XXX YYY ZZZ ...` — dispersion : XXX = position, puis une destination par armée de la pile (le nombre de destinations = taille de la pile ; la case de l'armée est une destination valide, listée explicitement)
- **Liaison par transition** : ordre **entre parenthèses** = boucle (`loop`) ; **sans parenthèses** = unique (`single`). Chaque ligne a sa propre liaison.
- **Position exigée** : si l'armée n'est pas sur la position indiquée quand l'ordre s'exécute → **failure** (single : chaîne brisée ; loop : retente).
- `#` : commentaire (ignoré jusqu'à fin de ligne). Lignes vides ignorées. Insensible à la casse (normalisé en MAJUSCULES). Codes invalides → erreur de parsing (avec numéro de ligne).

### Symboles et sémantique

| Symbole | Ordre | Syntaxe | Réussite | Échec (single) | Boucle (loop) |
|---|---|---|---|---|---|
| `A` | Attaque / atteindre | `XXX A YYY` | Déplacement vers une case **adjacente** (combat si occupée — résolution P1.4). **Les attaques de piles d'origines différentes ne se combinent pas** : chacune est un contendant distinct (à égalité au sommet → statu quo), **y compris entre piles d'un même joueur** (deux piles convergeant par A sur une case vide : la plus grosse entre, l'autre reste ; à égalité, statu quo — pour converger vraiment, il faut J) ; pour cumuler des forces, un soutien S | Chaîne brisée | Retente jusqu'à réussite |
| `S` | Soutien | `XXX S YYY` (défensif) ou `XXX S YYY - ZZZ` (offensif) | **Soutien explicite, toute nationalité** (même contre soi). *Défensif* : +taille de la pile à la défense de la pile en YYY (adjacente, ≠ XXX) qui **tient** (ordre sans déplacement H/S/P ; sans effet — gaspillé mais réussi — si elle se déplace ou est absente). *Offensif* : +taille de la pile à l'attaque de la pile de YYY vers ZZZ (ZZZ adjacente à XXX, YYY–ZZZ adjacentes ; sans effet — gaspillé mais réussi — si elle n'attaque pas ZZZ). **Coupé** si la pile soutenante est attaquée depuis une case **différente** de celle vers laquelle elle soutient (ZZZ en offensif, YYY en défensif) | Chaîne brisée | Offensif : index figé tant que la pile de YYY attaque ZZZ, avance sinon ; défensif : index figé tant que YYY est attaquée, avance sinon |
| `H` | Maintien | `H XXX` | L'armée reste sur XXX (sa position) | Chaîne brisée | **Garde indéfinie** : chaîne en veille jusqu'à réception d'un nouvel ordre |
| `J` | Jonction | `XXX J YYY` | **Déplacement pacifique** (pas une attaque, puissance 0) vers YYY **adjacente**, toujours en **dernier ordre d'une chaîne** (sinon chaîne invalide). **Fusionne** si une armée alliée est déjà sur YYY, ou si exactement une armée alliée y arrive au même tour **sans contestation** (aucune autre armée n'y converge — deux jonctions mutuelles J+J fusionnent) ; la **chaîne de l'hôte est conservée** (celle de la jonctionnante est consommée par ce J ; l'arrivant par A est l'hôte ; un rendez-vous J+J laisse la pile **Sans Ordre**) ; case occupée par l'ennemi → échec ; case **contestée** par une attaque ce tour, ou **convergence de plusieurs armées** → **repoussé** (échec) | Chaîne brisée | Retente jusqu'à réussite (puis chaîne consommée) |
| `P` | Pillage | `P XXX` | XXX = case où l'armée se trouve : détruit l'infrastructure de SA case (une seule par case, GDD §3) + bonus R (balance.json) **crédité au lieu-dit contrôlé le plus proche** du joueur (perdu s'il n'en contrôle aucun) ; **aucune infrastructure → ordre invalide** | Chaîne brisée | Retente jusqu'à destruction |
| `O` | Otage | `XXX O YYY` | YYY = code du noble prisonnier détenu par la pile de XXX (la case de l'armée) : il passe à l'état **otage** (état par défaut — il produit des rapports pour son propriétaire et compte en T0) ; noble non prisonnier, d'un autre joueur ou absent → **invalide** | Chaîne brisée | Retente |
| `K` | Cachot | `XXX K YYY` | YYY = code du noble prisonnier détenu par la pile de XXX : il passe **au cachot** (plus aucun rapport, ni récepteur, ni T0 pour son propriétaire) ; mêmes conditions d'invalidité que O | Chaîne brisée | Retente |
| `D` | Dispersion | `XXX D XXX YYY ZZZ ...` | Une destination **par armée de la pile** (nombre = taille de la pile) ; chaque destination est **adjacente** ou égale à la position ; à la résolution (P1.4), chaque destination **libre et non ciblée par une attaque** → l'armée s'y déplace pacifiquement ; la case de l'armée est valide ; **la chaîne reste sur l'armée d'origine, qui prend la première destination listée** (les autres armées sont créées). **Répartition des nobles par astérisque** : `XXX D YYY XXX*` = tous les nobles en XXX ; `YYY*JEA` = Jean en YYY ; `YYY*JEA*ANN` = Jean et Anne en YYY ; `*` seul = tous les nobles restants ; chaque noble au plus une fois ; **si des nobles chevauchent la pile et que l'ordre ne les assigne pas tous → ordre INVALIDE** (aucune armée d'origine par défaut) | Avance même si partielle | Retente jusqu'à résolution **intégrale** |

**Ordre invalide** (physiquement ou mécaniquement impossible : cible inexistante, cible non adjacente, armée détruite, pillage sans infrastructure...) : **brise immédiatement** la chaîne, quel que soit le mode de liaison.

**Armée sans chaîne** : une armée sans chaîne associée est *Sans Ordre* (pas de statut dédié : `army.chain == nil` suffit).

**Réception de la chaîne (P2.3)** : la chaîne est émise par le **noble de l'en-tête** depuis SA position ; elle voyage à la vitesse du terrain (coûts des assets d'équilibrage, cf. §5 de l'architecture) vers le **premier territoire de la feuille** (XXX de `XXX A YYY`, le territoire de `H XXX` ou `P XXX`). **L'arrivée est calculée à l'émission** (temps de trajet fixé). Entre l'arrivée et l'hiver suivant, la **première armée du joueur émetteur** présente sur ce territoire reçoit la chaîne — appliquée à **toute sa pile** — et **remplace** la sienne (départage : plus petit matricule). Une chaîne qui échoue la validation de réception (D ≠ taille de la pile, nobles non couverts) est **perdue**. Aucune armée requise à l'émission (l'armée peut arriver plus tard). Chaîne jamais reçue → **perdue** à la fin de l'hiver. Pas d'interception au MVP.

**Progression** : la progression des chaînes prend en compte **toutes les chaînes d'ordres simultanément** (résolution des combats, retraites, jonctions) — traitée dans le moteur de résolution P1.4.

### Ordres d'Hiver (phase de gestion)

L'hiver, le joueur soumet une **liste d'ordres** (même mécanique de soumission que les chaînes ; une ligne = un investissement ; traités dans l'ordre saisi) :

- `R N XXX` — recruter un **Noble** sur XXX (nom = prénom tiré + "de \<nom du territoire\>", ex. "Jacques de Notombes")
- `R A XXX` — recruter une **Armée** sur XXX
- `C M XXX` — construire un **Moulin** sur XXX (améliore le moulin existant si déjà présent)
- `C C XXX` — construire un **Château** sur XXX (rend la case lieu-dit)
- `C R XXX` — construire un **Relais de Poste** sur XXX
- `C T XXX` — construire une **Tour de Guet** sur XXX
- `C D XXX` — construire un **Dépôt de Vivres** sur XXX
- `E C XXX` — désigner le château de XXX comme **Capitale** du joueur (remplace la désignation actuelle ; exige de contrôler la case ; par défaut, la capitale est le premier château construit)
- `L N XXX` — libérer le noble prisonnier XXX (le noble réapparaît à la capitale de son propriétaire ; requis : noble prisonnier appartenant au joueur)

XXX = code du territoire ciblé (3 lettres pour un lieu-dit, 4 sinon). Pas de chaîne, pas de position, pas de liaison : un ordre d'hiver s'applique directement ou est **rejeté** (événement explicite ; ordre rejeté = investissement perdu). Coûts et métriques d'équilibrage : `assets/balance.json` (toutes les constantes du moteur, éditables sans recompiler le code).

---

## 7. Stratégie d'Exécution avec les Agents

Pour faire produire ce projet par des agents sans perdre le contrôle :

1. **Découper le moteur (`internal/engine`) en premier :** faire écrire les règles métier en Go pur avec des tests unitaires systématiques (`engine_test.go`).
2. **API REST (chi) :** contrats d'interface JSON explicites (DTO), testés via `httptest` (pas d'OpenAPI au MVP).
3. **Frontend SVG :** demander à l'agent de créer un composant `MapViewer` qui prend un JSON de territoires/unités et affiche le SVG avec gestion de la sélection de cases pour donner les ordres.
