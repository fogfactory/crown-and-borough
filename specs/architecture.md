# Architecture : Crown & Borough v1 et MVP online

Cette architecture décrit le cœur v1 livré et les contrats figés du MVP online :
un serveur Go avec un moteur pur, une API HTTP et un front React. La session
hotseat actuelle reste en mémoire et sert au développement. Le MVP hébergé
ajoutera plusieurs parties, Firebase Authentication, les vues privées et la
restauration via Firestore, sans modifier le moteur pour gérer l'hébergement.

## 1. Vue d'ensemble

Le jeu est résolu par saisons. Les joueurs préparent leurs ordres, les
soumettent au serveur, puis le moteur résout un tour complet de manière
déterministe. Le protocole HTTP suffit aux commandes du mode online ; le MVP
hébergé ajoute des listeners Firestore côté front, sans canal permanent entre
le navigateur et le serveur Go.

```text
┌──────────────────────────────────────────────────────────┐
│ Frontend Vite / React / TypeScript                        │
│ Carte SVG, poste de commandement, ordres, rapport         │
└───────────────────────────┬──────────────────────────────┘
                            │ HTTP + JSON · Firebase snapshots
┌───────────────────────────▼──────────────────────────────┐
│ Serveur Go                                                │
│ net/http ServeMux · Auth · store multi-parties             │
│                                                            │
│ ┌─────────────────────┐  ┌─────────────────────────────┐ │
│ │ Projection API       │  │ Moteur pur                   │ │
│ │ map/state/orders     │  │ résolution, carte, rapports  │ │
│ └─────────────────────┘  └─────────────────────────────┘ │
└───────────────────────────┬──────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────┐
│ Firestore Native mode                                     │
│ état canonique · projections · memberships · rapports     │
└──────────────────────────────────────────────────────────┘
```

Les assets `communes.csv`, `prenoms.csv` et `balance.yaml` restent locaux au
conteneur et sont chargés par le serveur ; ils ne constituent pas la
persistance de l'état d'une partie.

La session actuelle contient une seule partie en mémoire. Chaque joueur
soumet séparément ses ordres avec son identifiant. La résolution intervient
quand tous les joueurs ont soumis, ou lorsqu'un client utilise `force`.

## 2. Stack technique

### Backend

- Go 1.26 ;
- `net/http` et `http.ServeMux` avec les patterns de routes du standard ;
- dépendance runtime `gopkg.in/yaml.v3` limitée au chargement strict de la balance ;
- SDK Cloud Firestore et Firebase Admin SDK pour la persistance et la validation
  des ID tokens ;
- moteur sans dépendance web dans `internal/engine` ;
- JSON pour les contrats HTTP et YAML pour l'asset de balance ;
- mutex local par partie pour le cache et la résolution, avec Firestore comme
  autorité de persistance et de coordination inter-instance.

La résolution est organisée autour des fonctions pures du moteur, notamment
`Resolve`, `ResolveWinter`, `ResolveTurn`, `CreateGame` et `GenerateMap`. Les
tests couvrent les invariants du modèle, la génération déterministe, les
scénarios de combat, la logistique, l'hiver, les rapports et les handlers HTTP.

### Frontend

- Vite ;
- React et TypeScript strict ;
- Tailwind CSS et composants shadcn/ui ;
- carte SVG interactive ;
- appels HTTP vers le serveur Go ;
- Firebase Web SDK pour le lien de connexion par email et les listeners
  `onSnapshot` des projections autorisées.

Le front est lancé séparément en développement avec le proxy Vite. Le serveur
Go v1 expose l'API de partie ; l'intégration d'un bundle frontend dans le
conteneur et le déploiement public restent à traiter.

## 3. Organisation du dépôt

```text
.
├── assets/
│   ├── balance.yaml       # paramètres numériques v1
│   ├── communes.csv       # communes et affinités de terrain
│   └── prenoms.csv        # prénoms de nobles
├── cmd/server/
│   └── main.go            # démarrage HTTP et session par défaut
├── internal/
│   ├── api/               # handlers, DTO, auth et CORS de développement
│   ├── db/assetgen/       # chargement et validation des assets
│   ├── engine/            # résolution, rapports, logistique et carte
│   │   ├── demo/           # état de démonstration pour les outils de dev
│   │   ├── mapgen/         # génération géométrique et graphe
│   │   └── orders/         # parsing et validation des ordres
│   ├── models/            # modèles et invariants métier
│   └── store/             # interfaces mémoire et adaptateur Firestore online
├── web/
│   ├── src/                # application React et tests front
│   └── dist/               # sortie Vite locale quand elle est générée
├── Dockerfile
└── Makefile
```

Le contrat online ne conserve qu'une identité territoriale publique : `id`, qui
contient le trigramme de la commune (`ROS`). Le domaine utilise cette même
identité pour les clés, les adjacences, les positions et les références métier.
Aucun contrat, fixture ou format de persistance ne dépend d'un matricule
séquentiel ni n'expose un champ territorial `code` en doublon.

## 4. Contrats JSON

### `map.json`

La carte est statique pour une partie et commune à tous les clients :

```json
{
  "territories": [
    {
      "id": "ROS",
      "name": "Rosemont",
      "terrain": "plain",
      "village": true,
      "points": [[0, 0], [100, 0], [100, 80]],
      "adjacencies": ["BOI"],
      "impassable": ["FOU"]
    }
  ],
  "regions": [
    {
      "id": "ROS",
      "seed": "ROS",
      "territories": ["ROS", "BOI"]
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
      "capitalTerritory": "ROS"
    }
  ],
  "territories": [
    {
      "id": "ROS",
      "owner": "P1",
      "resources": 4,
      "army": {
        "owner": "P1",
        "size": 2,
        "chain": {
          "visibility": "known",
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
      "location": "ROS",
      "status": "free"
    }
  ]
}
```

`army` vaut `null` lorsqu'aucune armée n'occupe la case. Dans une armée, `chain`
vaut `null` lorsqu'aucune chaîne n'est active. Une chaîne existante dont le
détail n'est pas révélé est représentée par `{ "visibility": "hidden" }` ; une
chaîne connue contient `{ "visibility": "known", ... }` avec son détail. Les
identifiants d'armée, de chaîne et d'ordre internes ne sont pas exposés dans la
vue d'état. Les positions et les cibles des ordres utilisent les trigrammes
territoriaux.
`capitalTerritory` désigne le territoire du château actuellement choisi comme
capitale par le joueur ; le champ est absent lorsqu'il n'a pas de capitale.

La session hotseat conserve une projection globale lorsqu'aucun joueur n'est
fourni, mais le front utilise `GET /api/state?player=P1` pour demander la vue
filtrée du joueur sélectionné. La politique de divulgation est la suivante :

- la carte et les valeurs dynamiques chiffrées restent communes ;
- un joueur voit le détail des chaînes qu'il a émises, ainsi que celles émises
  par un noble qu'il détient comme otage, tant que la chaîne reste compatible
  avec la progression de l'armée ;
- un joueur voit les forces et le résultat exact d'un combat s'il intervient
  comme attaquant, défenseur ou soutien ;
- les rapports de combat indiquent séparément le bonus de commandement noble
  inclus dans la force de chaque armée ;
- pour un combat auquel il ne participe pas, il voit le traitement général des
  ordres, mais pas le détail des puissances.

Le filtrage de cette vue est fait côté serveur, avec l'identité du joueur. Le
front ne reçoit jamais les détails masqués et ne fait qu'afficher la variante
retournée par l'API.

### Vues privées des rapports

Un combat possède une visibilité explicite :

- `visibility: "exact"` contient les puissances, les identifiants d'armées et
  les autres détails nécessaires au joueur qui intervient comme attaquant,
  défenseur ou soutien ;
- `visibility: "general"` contient le territoire et le résultat général, mais
  aucun `force`, identifiant d'armée, propriétaire, contender ou détail de
  défense permettant de reconstruire le combat.

Les deux formes sont des variantes distinctes du contrat de rapport, pas deux
valeurs partielles que le front pourrait deviner. Une vue générale peut par
exemple être réduite à :

```json
{
  "visibility": "general",
  "territory": "BOI",
  "outcome": "standoff",
  "summary": "The combat ended without a winner."
}
```

La connaissance privée ne doit pas être déduite uniquement de l'état courant.
Le serveur conserve des métadonnées par partie, joueur, chaîne et combat. Le
schéma de persistance indicatif est :

```json
{
  "chainKnowledge": {
    "P1": {
      "C1": {
        "army": "A1",
        "noble": "N1",
        "currentIndex": 0,
        "orders": [{ "position": "ROS" }]
      }
    }
  },
  "combatParticipation": {
    "P1": ["combat-1"]
  }
}
```

Ce schéma est interne et indicatif. Les snapshots sont persistés avec la
partie : une chaîne émise par le joueur est connue, une chaîne émise par un
noble otage est connue par son détenteur, une progression compatible conserve
la connaissance, et un tiers ne perd pas sa connaissance au simple remplacement
de la chaîne. Elle est purgée lorsque la position publique de l'armée sort de
la trajectoire connue ou lorsque l'armée disparaît.

## 5. API v1 et MVP online

Les réponses de l'API sont JSON, à l'exception du document Markdown de
`/api/rules`. Le serveur de développement est configuré par `ASSETS_DIR`,
`SEED`, `PLAYERS` et `PORT`. Le MVP hébergé utilise Firestore et Firebase
Authentication : le client obtient un ID token par lien email et le serveur le
valide dans le header Bearer. Les tokens ne sont pas stockés par l'application.

| Méthode | Route | Comportement |
|---|---|---|
| `GET` | `/healthz` | Vérifie que le serveur répond. |
| `GET` | `/api/map` | Renvoie la carte de la session courante. |
| `GET` | `/api/state` | Renvoie l'état projeté global ; `?player=P1` active la vue privée hotseat. |
| `GET` | `/api/supply?territory=ROS` | Calcule la ligne ou la zone de ravitaillement sélectionnée. |
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
identité de développement déclarée par le client. Ces routes et le paramètre
`?player=` ne font pas partie de l'API publique authentifiée.

### Contrat MVP hébergé

Le MVP accepte plusieurs parties, chacune avec deux à huit joueurs online.
L'identifiant de partie est conservé dans toutes les routes. La liste des
parties est filtrée par l'appartenance du joueur ; il n'existe plus de conflit
global lorsqu'une autre partie est déjà active.

| Méthode | Route | Contrat |
|---|---|---|
| `GET` | `/api/auth/me` | Valide le JWT Firebase et renvoie le profil Firestore du joueur. |
| `PUT` | `/api/auth/me` | Crée ou met à jour le nom affiché validé du profil courant. |
| `POST` | `/api/games` | Crée une partie avec deux à huit slots ; le créateur devient automatiquement membre. |
| `GET` | `/api/games` | Liste les parties dont le joueur courant est membre. |
| `GET` | `/api/games/{id}` | Renvoie le statut, les slots, le tour et la saison ; le code d'invitation est privé au créateur. |
| `GET` | `/api/games/{id}/invite` | Renvoie le lien d'invitation au créateur uniquement. |
| `POST` | `/api/games/{id}/join` | Rejoint un slot avec le code d'invitation ; l'UID Firebase courant est l'identité du membre. |
| `GET` | `/api/games/{id}/map` | Renvoie le `map.json` commun, dont `territories[].id` est le trigramme. |
| `GET` | `/api/games/{id}/state` | Renvoie la projection privée du joueur connecté ; aucun `?player=` public. |
| `GET` | `/api/games/{id}/supply?territory=ROS` | Calcule la ligne ou la zone de ravitaillement demandée. |
| `POST` | `/api/games/{id}/orders` | Remplace la soumission du joueur courant ; résout automatiquement lorsque tous les joueurs vivants ont soumis. Le corps ne contient aucun identifiant joueur. |
| `POST` | `/api/games/{id}/resolve` | Résolution forcée explicite avec des ordres vides pour les joueurs manquants. |
| `GET` | `/api/games/{id}/reports` | Liste les rapports filtrés pour le joueur connecté. |
| `GET` | `/api/games/{id}/reports/{index}` | Renvoie un rapport filtré pour le joueur connecté. |
| `GET` | `/api/rules?lang=fr` | Renvoie les règles publiques en Markdown. |

Les erreurs utilisent au minimum la forme `{ "error": "code", "message":
"..." }`. Les erreurs de validation peuvent ajouter `details` sans changer les
champs de base. Les statuts structurants sont `400` pour une requête invalide,
`401` pour un token absent ou invalide, `403` pour un joueur non membre,
`404` pour une partie ou une ressource inconnue et `409` pour un conflit de
partie, de slot ou d'état.

Une soumission en attente renvoie `status: "pending"` avec `submitted` et
`remaining`. La dernière soumission renvoie `status: "resolved"` et le rapport
du tour. Il n'existe aucune deadline automatique : `POST /resolve` est l'action
explicite qui permet aux amis de débloquer une partie.

Le front peut écouter directement les projections Firestore suivantes avec le
SDK Firebase Web : `games/{id}` pour le résumé public et
`games/{id}/views/{uid}` pour son état privé. Les règles refusent les documents
canoniques, les soumissions, les rapports non filtrés et les écritures clientes.
Les listeners remplacent le polling régulier ; les routes REST restent la
source des commandes et le fallback d'initialisation ou de reconnexion.

## 6. Moteur et résolution

Les modèles métier sont dans `internal/models`. Ils valident notamment :

- l'unicité des joueurs, territoires, trigrammes, armées, nobles et
  infrastructures ;
- la symétrie du graphe et l'existence des références ;
- une seule armée et une seule infrastructure par territoire ;
- la cohérence entre les index de `GameState` et les entités ;
- la saison calculée à partir du tour absolu.

`ResolveTurn` choisit la résolution d'action ou d'hiver selon la saison, avance
le calendrier et renvoie un `TurnReport`. La soumission `special` est indépendante
des chaînes de nobles et des investissements d'hiver. Les ordres de cartes sont
validés et consommés avant les phases militaires ; leurs effets sont agrégés par
région avant le ravitaillement et l'énumération des intentions. Le rapport
contient des sections typées pour les joueurs, ordres, combats, mouvements,
ravitaillement, famine, nobles, rumeurs publiques et investissements d'hiver. Le
moteur ne dépend ni du HTTP ni du rendu front.

La réception des chaînes est immédiate et atomique. La validation statique
conserve volontairement la non-adjacence jusqu'à l'exécution : un ordre
non-adjacent casse la chaîne à l'endroit où il est rencontré, sans annuler les
ordres précédents.

Plusieurs chaînes ciblant la même armée au même tour constituent une réception
concurrente : elles sont toutes rejetées avant la résolution et aucune nouvelle
chaîne n'est attachée à cette armée. Une chaîne déjà portée reste inchangée.

Les ordres exécutables du moteur sont séparés des DTO parsés et persistés. Les
ordres de cartes sont construits par un registre `CardDefinition` indexé par
`CardKind`. Leur `Apply` consomme la première carte correspondante dans la main,
puis enregistre une intention ; l’agrégation des intentions intervient ensuite
pour préserver la simultanéité.

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
BRI D BRI ATL NOR
```

Les ordres d'hiver v1 sont limités à `A N`, `R N`, `R T`, `C M`, `C C`, `C D`, `E C`,
`O N`, `P N` et `L N`. Une soumission `special` séparée contient les ordres du
deck : `P KIND TER` au printemps, en été et en automne, et `D C KIND` ou `T C` en
hiver. Aucun de ces ordres n'exige de noble et ils ne sont jamais intégrés à la
grammaire des chaînes de nobles. Les infrastructures absentes du modèle v1 ne
possèdent ni symbole de parser ni coût dans `balance.yaml`.

## 8. Assets et balance

Les assets sont chargés au démarrage et validés avant de créer la session :

- `communes.csv` fournit les noms, codes et affinités de terrain ;
- `prenoms.csv` fournit les noms et codes de nobles ;
- `balance.yaml` fournit les coûts, productions, portées, rations, bonus de
  défense, bonus de commandement noble, valeurs de départ et paramètres du
  deck d’ordres spéciaux utilisés par le moteur.

Les paramètres numériques ne doivent pas être recopiés dans les handlers ou
le front. Le moteur reçoit une `assetgen.Balance` déjà chargée.

## 9. Évolutions d'infrastructure

Les fonctionnalités suivantes ne font pas partie de la session v1 en mémoire
et sont suivies par l'issue online :

- identifiant territorial unique fondé sur le trigramme (contrat figé par O1,
  implémentation O2) ;
- filtre serveur des vues par joueur ;
- gestion de plusieurs parties, Firebase Authentication et les invitations ;
- persistance Firestore avec projections publiques et privées ;
- bundle frontend servi par le serveur et déploiement public.

Un brouillard de guerre général pourra éventuellement réintroduire des
infrastructures de vision dédiées. Cela constituera une extension de règles et
un contrat de vue distinct, pas une modification silencieuse du cœur v1.

## 10. Cible online MVP

Le déploiement online cible plusieurs parties de deux à huit joueurs.
L'identifiant de partie est conservé dans les routes `/api/games/{id}` et la
liste est filtrée par membership.

Le serveur porte l'identité du joueur à partir d'un ID token Firebase porté en
Bearer. Le paramètre `player` peut exister dans un mode de test local, mais il
n'est jamais utilisé par l'API publique authentifiée. Les endpoints hotseat sont
désactivés dans le déploiement public.

La projection serveur conserve les informations dynamiques publiques mais
filtre les chaînes et les combats selon le joueur. La connaissance des chaînes
et les audiences des rapports sont des métadonnées de serveur, indépendantes du
rendu React et persistées avec la partie.

Firestore Native mode est la frontière de persistance. Les documents de résumé
public, de vue privée et de rapports filtrés sont séparés des documents
canoniques réservés au backend. Les règles Firestore refusent au navigateur
l'état moteur, les soumissions brutes, les rapports non filtrés et toute
écriture directe.

La mutation d'une partie vérifie la révision dans une transaction. La résolution
utilise une revendication avec lease et un commit conditionnel ; le moteur pur
peut être rejoué sans effet externe. Cette garantie reste valable si Cloud Run
est configuré plus tard avec plusieurs instances.

Firebase Authentication gère l'identité et la session client par lien email.
Le profil `players/{uid}` et les memberships survivent aux redémarrages, mais
les ID tokens ne sont pas copiés dans Firestore. Le frontend utilise
`onSnapshot` uniquement sur `games/{id}` et `games/{id}/views/{uid}` après
authentification ; les commandes passent par l'API Go.

Cloud Run est la cible unique du MVP, avec `min-instances=0` et une limite
initiale d'instances pour maîtriser le coût. Aucun volume GCS FUSE, Persistent
Disk ou workflow Compute Engine de repli n'est requis. Le free-tier GCP reste
un objectif de coût et non une garantie.
