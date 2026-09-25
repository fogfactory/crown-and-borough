# Architecture : Crown & Borough v1 et MVP online

Cette architecture décrit le cœur v1 et le MVP online livrés : un serveur Go
avec un moteur pur, une API HTTP et un front React servi par le même binaire.
Le MVP hébergé gère plusieurs parties, Firebase Authentication, les vues
privées et la restauration via Firestore, sans que le moteur dépende de
l'hébergement. La session hotseat en mémoire reste disponible pour le
développement local.

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

Le hotseat local utilise la même API de parties que le mode en ligne, servie
par le store mémoire en mode de développement : chaque joueur soumet ses
ordres sous son identité de développement et la résolution intervient quand
tous les joueurs attendus ont soumis, ou lorsque l'hôte la force (section 5).

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

Le bundle Vite est embarqué dans le binaire Go (`web/embed.go`) et servi sur
la même origine que l'API, en hotseat comme en ligne. `make web-dev` reste
disponible pour itérer sur le front seul, avec le proxy Vite vers un serveur Go
lancé séparément. Le déploiement public passe par Cloud Run et Firebase
Hosting, déclenché par un tag `v*` (voir `docs/deploy-cloudrun.md`).

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
│   ├── store/             # interfaces mémoire et adaptateur Firestore online
│   └── turn/              # cycle d'un tour partagé par le hotseat et les stores
├── web/
│   ├── embed.go            # embarque dist/ dans le binaire Go
│   ├── handler.go          # sert la SPA sur la même origine que l'API
│   ├── src/                # application React et tests front
│   └── dist/               # sortie Vite, générée par `make web-build`
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
territoires supplémentaires dédiés aux `joueurs + 1` villages. Chaque partie
sert sa carte via `GET /api/games/{id}/map`.

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
      "capitalTerritory": "ROS",
      "projectedIncome": 6
    }
  ],
  "territories": [
    {
      "id": "ROS",
      "owner": "P1",
      "resources": 4,
      "projectedIncome": 6,
      "incomeDestination": "ROS",
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

`projectedIncome` est le revenu territorial (`territory_income` +
`village_income` éventuel) que rapporterait ce joueur ou ce territoire au
prochain tour d'action, en ignorant les cartes calamité déjà tirées ce tour
pour ne pas en révéler l'effet à l'avance ; il vaut `0` en hiver.
`incomeDestination` est le territoire qui recevra ce revenu (la capitale du
joueur, à défaut le château contrôlé le plus proche, à défaut le village
contrôlé le plus proche) ; il est absent si le revenu est perdu faute de
destination.

`army` vaut `null` lorsqu'aucune armée n'occupe la case. Dans une armée, `chain`
vaut `null` lorsqu'aucune chaîne n'est active. Une chaîne existante dont le
détail n'est pas révélé est représentée par `{ "visibility": "hidden" }` ; une
chaîne connue contient `{ "visibility": "known", ... }` avec son détail. Les
identifiants d'armée, de chaîne et d'ordre internes ne sont pas exposés dans la
vue d'état. Les positions et les cibles des ordres utilisent les trigrammes
territoriaux.
`capitalTerritory` désigne le territoire du château actuellement choisi comme
capitale par le joueur ; le champ est absent lorsqu'il n'a pas de capitale.

`GET /api/games/{id}/state` renvoie la vue filtrée du joueur connecté ; le
hotseat demande celle du joueur sélectionné avec `?player=P1` en mode de
développement. La politique de divulgation est la suivante :

- la carte et les valeurs dynamiques chiffrées restent communes ;
- `specialHand` contient uniquement les kinds bonus de la main du joueur courant ; la pioche, la défausse et les IDs internes restent absents ;
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
| `GET` | `/api/version` | Renvoie la version de l'application. |
| `GET` | `/api/rules?lang=fr` | Renvoie les règles publiques en Markdown. |

En mode de développement (`ONLINE_DEV_MODE=true`), l'identité du joueur n'est
pas authentifiée : l'API de parties accepte `?player=P1` ou l'en-tête
`X-Dev-Player`, et `P1` par défaut. Le store mémoire crée au démarrage une
partie hotseat décrite par `SEED` et `PLAYERS` (2 à 16 joueurs, 4 par défaut),
dont `P1` est l'hôte : il peut forcer la résolution. Le hotseat du navigateur
utilise uniquement cette API ; il n'existe plus de routes hotseat dédiées.

### Contrat MVP hébergé

Le MVP accepte plusieurs parties, chacune avec deux à huit joueurs online.
L'identifiant de partie est conservé dans toutes les routes. La liste des
parties est filtrée par l'appartenance du joueur ; il n'existe plus de conflit
global lorsqu'une autre partie est déjà active.

| Méthode | Route | Contrat |
|---|---|---|
| `GET` | `/api/auth/me` | Valide le JWT Firebase et renvoie le profil Firestore du joueur. |
| `PUT` | `/api/auth/me` | Crée ou met à jour le nom affiché validé du profil courant. |
| `POST` | `/api/games` | Crée une partie avec deux à huit slots ; le créateur devient automatiquement membre, sauf avec `spectate: true`, où il devient un hôte observateur sans occuper de slot. |
| `GET` | `/api/games` | Liste les parties dont le joueur courant est membre. |
| `GET` | `/api/games/{id}` | Renvoie le statut, les slots, le tour et la saison ; le code d'invitation est privé au créateur. |
| `GET` | `/api/games/{id}/invite` | Renvoie le lien d'invitation au créateur uniquement. |
| `POST` | `/api/games/{id}/join` | Rejoint un slot avec le code d'invitation ; l'UID Firebase courant est l'identité du membre. |
| `GET` | `/api/games/{id}/map` | Renvoie le `map.json` commun, dont `territories[].id` est le trigramme. |
| `GET` | `/api/games/{id}/state` | Renvoie la projection privée du joueur connecté ; un hôte observateur reçoit la projection complète ; aucun `?player=` public. |
| `GET` | `/api/games/{id}/supply?territory=ROS&special=…` | Calcule la ligne ou la zone de ravitaillement demandée, après les calamités de la saison (peste, famine, mauvais temps) et les cartes du brouillon `special` du joueur connecté (Beau temps, Bonne récolte ; la Révolte est ignorée pour ne pas révéler son tirage). La ligne détaille `terrainProduction`, `famineRations` et `bonusRations`. |
| `GET` | `/api/games/{id}/supply?territory=ROS&target=BOI` | Estime la route d'un transfert d'action vers `BOI`. |
| `POST` | `/api/games/{id}/orders` | Remplace la soumission du joueur courant (`chains`, `winter`, `special`) ; résout automatiquement lorsque tous les joueurs attendus ont soumis : un joueur éliminé n'est jamais attendu et, en saison d'action, un joueur sans noble libre ou otage non plus. Un hôte observateur ne peut pas soumettre. Le corps ne contient aucun identifiant joueur. |
| `POST` | `/api/games/{id}/orders/preview` | Résolution à blanc du brouillon du joueur courant, sans rien enregistrer : erreurs (syntaxe, adjacence, réception), ordres parsés de chaque chaîne, issue simulée de chaque ligne d'hiver et coût d'hiver. Le client l'appelle pendant la saisie et ne réimplémente aucune règle d'ordre. |
| `GET` | `/api/games/{id}/my-submission` | Renvoie la dernière soumission du tour courant pour le joueur connecté (chaînes et hiver) pour réhydrater les formulaires après refresh. |
| `GET` | `/api/games/{id}/submitted-orders` | Renvoie les ordres déjà soumis du tour courant à l'hôte observateur uniquement, avec leur résolution à blanc (`preview`), afin d'afficher la couche d'intentions ; les joueurs ne peuvent pas lire les ordres des autres. |
| `POST` | `/api/games/{id}/resolve` | Résolution forcée explicite avec des ordres vides pour les joueurs manquants. |
| `GET` | `/api/games/{id}/reports` | Liste les rapports filtrés pour le joueur connecté. |
| `GET` | `/api/games/{id}/reports/{index}` | Renvoie un rapport filtré pour le joueur connecté, ou complet pour l'hôte observateur. |
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
SDK Firebase Web : `games/{id}` pour le résumé public, `games/{id}/views/{uid}`
pour l'état privé d'un joueur et `games/{id}/observer/{uid}` pour l'état complet
d'un hôte qui ne joue pas. Le document observateur ne contient qu'un pointeur
vers le dernier rapport ; le rapport complet reste servi par REST. Les règles
refusent les documents canoniques, les soumissions, les rapports non filtrés et
les écritures clientes.
Les listeners remplacent le polling régulier ; les routes REST restent la
source des commandes, des rapports et le fallback d'initialisation ou de
reconnexion. Un hôte observateur est aussi ajouté à `memberUids` pour pouvoir
écouter le résumé public et le document observateur, mais il n'est jamais
compté parmi les deux à huit joueurs.

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

La réception des chaînes est immédiate et atomique. La validation est en une
seule couche : `orders.ValidateChain` porte toutes les règles statiques
(forme, références, adjacence, jonction en dernier, affectations de nobles…)
et `ResolveTurn` refuse la soumission sur la moindre de ces erreurs, avec sa
ligne source. `AssignChain` ne vérifie plus que les conditions de réception
(noble apte, armée présente et possédée), et l'exécution que les conditions
du monde (position, infrastructure, contrôle, route, nobles présents).
`Resolve` revérifie en préalable que les chaînes stockées passent la
validation statique, et refuse un état qui ne la passe pas.

Plusieurs chaînes ciblant la même armée au même tour constituent une réception
concurrente : elles sont toutes rejetées avant la résolution et aucune nouvelle
chaîne n'est attachée à cette armée. Une chaîne déjà portée reste inchangée.

L'adjudication des attaques, jonctions et dispersions (`adjudicator.go`) est
un graphe de décisions booléennes : une attaque atteint sa destination, une
jonction ou une dispersion vide son origine, une armée restée en place est
délogée. Chaque décision est une fonction pure des autres ; l'équation de
mouvement et les forces (attaque, maintien, défense, prévention) suivent
« The Math of Adjudication » de Lucas Kruijswijk, référence des Diplomacy
Adjudicator Test Cases, avec les forces de Crown & Borough (§5 du GDD). Avant
que ce graphe ne soit construit, toute jonction ou dispersion dont l'origine
est la cible d'une attaque, quel qu'en soit l'auteur et quelle qu'en soit
l'issue, est annulée (`cancelAttackedOriginPeaceful`) : son armée reste sur
place comme si elle tenait, et `applyContestOutcomes` en rapporte la raison
(`attacked_origin`) une fois les délogements connus, pour ne pas écraser un
délogement par cette raison. Cette règle générale élimine toute dépendance
d'un combat envers un départ pacifique, ce qui ne laisse plus que les cycles
d'attaques (rotation, jonction croisant une attaque) que les jonctions et
dispersions restantes évaluent par les règles pacifiques existantes, sur le
groupe d'ordres pacifiques qui partagent leurs territoires, en lisant les
combats à travers ces décisions. Le graphe statique des dépendances est
découpé en composantes fortement connexes (Tarjan), résolues dans l'ordre
topologique. Un cycle est résolu par recherche en profondeur : chaque ordre,
dans un ordre fixe, réussit dès qu'une résolution cohérente le permet, ce qui
généralise le mouvement circulaire de Diplomacy. Un cycle sans résolution
cohérente ne devrait donc plus survenir, la règle ci-dessus ayant retiré
toute dépendance d'origine ; le statu quo des attaques (`statusQuo`) reste
néanmoins en place comme filet de sécurité bon marché, au cas où un cas
échapperait à la règle. La résolution termine toujours, sans plafond
d'itérations.

Le corpus `internal/engine/testdata/adjudication_corpus.golden` fige le
résultat complet de deux tours consécutifs sur 10 000 plateaux aléatoires ;
les graines où l'adjudicateur actuel diffère volontairement de l'ancien point
fixe itératif sont listées avec leur motif dans
`adjudication_corpus.divergences`.

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

Les ordres d'hiver v1 comprennent `A N`, `R N`, `R T`, `C M`, `C C`, `C D`, `E C`,
`O N`, `P N`, `L N` et `G XXX YYY N`, avec `D C KIND` pour les défausses de cartes bonus.
Une soumission `special` séparée contient les ordres jouables du deck : `P KIND TER`
au printemps, en été et en automne. En hiver, la main est reconstituée
automatiquement après les défausses selon la balance ; il n'existe pas d'ordre de
pioche. Aucun de ces ordres n'exige de noble et ils ne sont jamais intégrés à la
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

Les fonctionnalités suivies par l'issue online sont livrées : identifiant
territorial fondé sur le trigramme, filtre serveur des vues par joueur,
plusieurs parties avec Firebase Authentication et invitations, persistance
Firestore avec projections publiques et privées, bundle frontend servi par le
serveur et déploiement public.

Un brouillard de guerre général pourra éventuellement réintroduire des
infrastructures de vision dédiées. Cela constituera une extension de règles et
un contrat de vue distinct, pas une modification silencieuse du cœur v1.

## 10. MVP online

Le déploiement online gère plusieurs parties de deux à huit joueurs.
L'identifiant de partie est conservé dans les routes `/api/games/{id}` et la
liste est filtrée par membership.

Le serveur porte l'identité du joueur à partir d'un ID token Firebase porté en
Bearer. Le paramètre `player` peut exister dans un mode de test local, mais il
n'est jamais utilisé par l'API publique authentifiée.

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
`onSnapshot` uniquement sur `games/{id}`, `games/{id}/views/{uid}` et, pour un
hôte observateur, `games/{id}/observer/{uid}` après authentification ; les
commandes passent par l'API Go.

Cloud Run est la cible unique du MVP, avec `min-instances=0` et une limite
initiale d'instances pour maîtriser le coût. Aucun volume GCS FUSE, Persistent
Disk ou workflow Compute Engine de repli n'est requis. Le free-tier GCP reste
un objectif de coût et non une garantie.
