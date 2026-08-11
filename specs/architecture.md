# Architecture : Crown & Borough v1

Cette architecture décrit le système réellement livré en v1 : un serveur Go
avec une session de partie en mémoire, un moteur pur, une API HTTP et un front
React. La persistance, l'authentification, les parties multiples et le
déploiement public sont des évolutions suivies dans GitHub.

## 1. Vue d'ensemble

Le jeu est résolu par saisons. Les joueurs préparent leurs ordres, les
soumettent au serveur, puis le moteur résout un tour complet de manière
déterministe. Le protocole HTTP suffit au mode online actuel : il n'y a pas de
simulation temps réel ni de canal permanent à maintenir.

```text
┌──────────────────────────────────────────────────────────┐
│ Frontend Vite / React / TypeScript                        │
│ Carte SVG, poste de commandement, ordres, rapport         │
└───────────────────────────┬──────────────────────────────┘
                            │ HTTP + JSON
┌───────────────────────────▼──────────────────────────────┐
│ Serveur Go                                                │
│ net/http ServeMux · Session · endpoints de partie         │
│                                                            │
│ ┌─────────────────────┐  ┌─────────────────────────────┐ │
│ │ Projection API       │  │ Moteur pur                   │ │
│ │ map/state/orders     │  │ résolution, carte, rapports  │ │
│ └─────────────────────┘  └─────────────────────────────┘ │
└───────────────────────────┬──────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────┐
│ Assets statiques                                          │
│ communes.csv · prenoms.csv · balance.json                 │
└──────────────────────────────────────────────────────────┘
```

La session actuelle contient une seule partie en mémoire. Chaque joueur
soumet séparément ses ordres avec son identifiant. La résolution intervient
quand tous les joueurs ont soumis, ou lorsqu'un client utilise `force`.

## 2. Stack technique

### Backend

- Go 1.26 ;
- `net/http` et `http.ServeMux` avec les patterns de routes du standard ;
- aucune dépendance Go tierce au runtime ;
- moteur sans dépendance web dans `internal/engine` ;
- JSON pour les contrats HTTP et les assets de balance ;
- mutex de session autour de l'état et de la carte correspondante.

La résolution est organisée autour des fonctions pures du moteur, notamment
`Resolve`, `ResolveWinter`, `ResolveTurn`, `CreateGame` et `GenerateMap`. Les
tests couvrent les invariants du modèle, la génération déterministe, les
scénarios de combat, la logistique, l'hiver, les rapports et les handlers HTTP.

### Frontend

- Vite ;
- React et TypeScript strict ;
- Tailwind CSS et composants shadcn/ui ;
- carte SVG interactive ;
- appels HTTP vers le serveur Go.

Le front est lancé séparément en développement avec le proxy Vite. Le serveur
Go v1 expose l'API de partie ; l'intégration d'un bundle frontend dans le
conteneur et le déploiement public restent à traiter.

## 3. Organisation du dépôt

```text
.
├── assets/
│   ├── balance.json       # paramètres numériques v1
│   ├── communes.csv       # communes et affinités de terrain
│   └── prenoms.csv        # prénoms de nobles
├── cmd/server/
│   └── main.go            # démarrage HTTP et session par défaut
├── internal/
│   ├── api/               # handlers, DTO, session et CORS de développement
│   ├── db/assetgen/       # chargement et validation des assets
│   ├── engine/            # résolution, rapports, logistique et carte
│   │   ├── demo/           # état de démonstration pour les outils de dev
│   │   ├── mapgen/         # génération géométrique et graphe
│   │   └── orders/         # parsing et validation des ordres
│   └── models/            # modèles et invariants métier
├── web/
│   ├── src/                # application React et tests front
│   └── dist/               # sortie Vite locale quand elle est générée
├── Dockerfile
└── Makefile
```

Les identifiants internes restent distincts des codes d'usage. Une carte porte
actuellement un `TerritoryID` séquentiel interne (`T01`) et un trigramme public
de commune (`ROS`). Les ordres et les labels utilisent le trigramme ; la
suppression du double identifiant est suivie comme une évolution online.

## 4. Contrats JSON

### `map.json`

La carte est statique pour une partie et commune à tous les clients :

```json
{
  "territories": [
    {
      "id": "T01",
      "code": "ROS",
      "name": "Rosemont",
      "terrain": "plain",
      "village": true,
      "points": [[0, 0], [100, 0], [100, 80]],
      "adjacencies": ["T02"],
      "impassable": ["T03"]
    }
  ]
}
```

- `points` décrit le polygone SVG ;
- `adjacencies` contient les frontières géométriques franchissables ;
- `impassable` contient les frontières géométriques infranchissables ;
- les deux listes sont triées, symétriques et disjointes ;
- `village` décrit le point de génération ; l'état courant des infrastructures
  est porté par `state.json`.

La génération utilise `8 x joueurs` territoires de jeu et `(joueurs + 1) x 4`
territoires supplémentaires dédiés aux `joueurs + 1` villages. Le serveur de
partie sert la carte courante après `POST /api/game`.

### `state.json`

L'état projeté sépare la couche dynamique du `GameState` de stockage :

```json
{
  "turn": 5,
  "season": "spring",
  "players": [
    {
      "id": "P1",
      "name": "Joueur 1",
      "color": "#a84632",
      "capitalTerritory": "T01"
    }
  ],
  "territories": [
    {
      "id": "T01",
      "owner": "P1",
      "resources": 4,
      "army": {
        "owner": "P1",
        "size": 2,
        "chain": {
          "noble": "HUG",
          "currentIndex": 0,
          "orders": [
            {
              "type": "attack",
              "position": "ROS",
              "targets": ["BOI"],
              "liaison": "single"
            }
          ]
        }
      },
      "infrastructures": [{ "type": "castle", "level": 1 }]
    }
  ],
  "nobles": [
    {
      "id": "N1",
      "code": "HUG",
      "name": "Hugues de Rosemont",
      "owner": "P1",
      "location": "T01",
      "status": "free"
    }
  ]
}
```

`army` vaut `null` lorsqu'aucune armée n'occupe la case. Les identifiants
d'armée, de chaîne et d'ordre internes ne sont pas exposés dans cette vue. Les
positions et les cibles des ordres utilisent les trigrammes territoriaux.
`capitalTerritory` désigne le territoire du château actuellement choisi comme
capitale par le joueur ; le champ est absent lorsqu'il n'a pas de capitale.

La v1 actuelle expose cette projection globalement, ce qui permet au front
hotseat d'afficher la partie complète. La politique de divulgation cible est
la suivante :

- la carte et les valeurs dynamiques chiffrées restent communes ;
- un joueur voit le détail des chaînes qu'il a émises, ainsi que celles émises
  par un noble qui lui appartient et qui est devenu otage, tant que la chaîne
  reste compatible avec la progression de l'armée ;
- un joueur voit les forces et le résultat exact d'un combat s'il intervient
  comme attaquant, défenseur ou soutien ;
- pour un combat auquel il ne participe pas, il voit le traitement général des
  ordres, mais pas le détail des puissances.

Le filtrage de cette vue doit être fait côté serveur, avec l'identité du joueur.
Il n'est pas délégué au front et constitue une tâche de l'issue online.

## 5. API v1

Toutes les réponses sont JSON. Le serveur est configuré par `ASSETS_DIR`,
`SEED`, `PLAYERS` et `PORT` ; la partie initiale est créée au démarrage.

| Méthode | Route | Comportement |
|---|---|---|
| `GET` | `/healthz` | Vérifie que le serveur répond. |
| `GET` | `/api/map` | Renvoie la carte de la session courante. |
| `GET` | `/api/state` | Renvoie l'état projeté de la session courante. |
| `GET` | `/api/supply?territory=T01` | Calcule la ligne ou la zone de ravitaillement sélectionnée. |
| `POST` | `/api/game` | Remplace la session par une nouvelle partie en mémoire. |
| `POST` | `/api/orders` | Enregistre la soumission d'un joueur et résout si tous ont soumis. |
| `POST` | `/api/reset` | Recrée la partie initiale configurée au démarrage. |

`POST /api/game` accepte une seed et une liste de joueurs. Les joueurs peuvent
être des objets `{id,name,color}`, des noms ou un nombre. Le moteur refuse les
parties hors de la plage 2–16 joueurs.

`POST /api/orders` accepte :

```json
{
  "player": "P1",
  "chains": [
    { "player": "P1", "noble": "HUG", "text": "HUG\nROS A BOI" }
  ],
  "winter": [],
  "force": false
}
```

Une soumission remplace la soumission précédente du même joueur pour le tour.
Tant que des joueurs manquent, la réponse contient `status: "pending"`, les
listes `submitted` et `remaining`, ainsi que l'état courant. Lorsque le tour
est résolu, la réponse contient `status: "resolved"`, le rapport et le nouvel
état. `force: true` permet de résoudre avec les soumissions déjà présentes.

Le serveur v1 ne fournit pas encore d'identité fiable : `player` est une
identité de développement déclarée par le client. L'authentification, les
sessions, les codes d'invitation et les parties multiples font partie des
évolutions online.

## 6. Moteur et résolution

Les modèles métier sont dans `internal/models`. Ils valident notamment :

- l'unicité des joueurs, territoires, trigrammes, armées, nobles et
  infrastructures ;
- la symétrie du graphe et l'existence des références ;
- une seule armée et une seule infrastructure par territoire ;
- la cohérence entre les index de `GameState` et les entités ;
- la saison calculée à partir du tour absolu.

`ResolveTurn` choisit la résolution d'action ou d'hiver selon la saison, avance
le calendrier et renvoie un `TurnReport`. Le rapport contient des sections
typées pour les joueurs, ordres, combats, mouvements, ravitaillement, famine,
nobles et investissements d'hiver. Le moteur ne dépend ni du HTTP ni du rendu
front.

La réception des chaînes est immédiate et atomique. La validation statique
conserve volontairement la non-adjacence jusqu'à l'exécution : un ordre
non-adjacent casse la chaîne à l'endroit où il est rencontré, sans annuler les
ordres précédents.

Plusieurs chaînes ciblant la même armée au même tour constituent une réception
concurrente : elles sont toutes rejetées avant la résolution et aucune nouvelle
chaîne n'est attachée à cette armée. Une chaîne déjà portée reste inchangée.

## 7. Format des ordres

Une chaîne est composée du code du noble émetteur puis d'une ligne par ordre.
Les lignes entre parenthèses sont en boucle ; les autres sont uniques. Les
positions de l'armée sont explicites.

```text
JEA
BRI A ATL
BRI S ATL - NOR
(BRI S ATL)
(ATL A NOR)
H BRI
BRI J ROS
P BRI
BRI O HUG
BRI K HUG
BRI D BRI ATL NOR
```

Les ordres d'hiver v1 sont limités à `R N`, `R T`, `C M`, `C C`, `C D`, `E C`
et `L N`. Les infrastructures absentes du modèle v1 ne possèdent ni symbole
de parser ni coût dans `balance.json`.

## 8. Assets et balance

Les assets sont chargés au démarrage et validés avant de créer la session :

- `communes.csv` fournit les noms, codes et affinités de terrain ;
- `prenoms.csv` fournit les noms et codes de nobles ;
- `balance.json` fournit les coûts, productions, portées, rations, bonus de
  défense et valeurs de départ utilisés par le moteur.

Les paramètres numériques ne doivent pas être recopiés dans les handlers ou
le front. Le moteur reçoit une `assetgen.Balance` déjà chargée.

## 9. Évolutions d'infrastructure

Les fonctionnalités suivantes ne font pas partie de la session v1 en mémoire
et sont suivies par l'issue online :

- identifiant territorial unique fondé sur le trigramme ;
- filtre serveur des vues par joueur ;
- gestion d'une partie active et authentification ;
- persistance JSON avec backend filesystem et backend snapshot pour GCS FUSE ;
- bundle frontend servi par le serveur et déploiement public.

Un brouillard de guerre général pourra éventuellement réintroduire des
infrastructures de vision dédiées. Cela constituera une extension de règles et
un contrat de vue distinct, pas une modification silencieuse du cœur v1.

## 10. Cible online MVP

Le déploiement online cible une seule partie active et deux à cinq joueurs.
L'identifiant de partie est conservé dans les routes `/api/games/{id}` afin de
permettre une évolution multi-parties sans changer les contrats publics.

Le serveur porte l'identité du joueur à partir d'un token Bearer. Le paramètre
`player` peut exister dans un mode de test local, mais il n'est jamais utilisé
par l'API publique authentifiée. Les endpoints hotseat sont désactivés dans le
déploiement public.

La projection serveur conserve les informations dynamiques publiques mais
filtre les chaînes et les combats selon le joueur. La connaissance des chaînes
et les audiences des rapports sont des métadonnées de serveur, indépendantes du
rendu React et persistées avec la partie.

`DATA_DIR` est la frontière de portabilité de la persistance. Un filesystem
local ou un Persistent Disk peut garantir `fsync` et `rename`. Un volume GCS
FUSE doit utiliser une stratégie de snapshots complets validée par un smoke
test de redémarrage ; Cloud Run n'est pas certifié si ce test échoue.
