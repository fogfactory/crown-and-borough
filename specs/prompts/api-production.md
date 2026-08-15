# Prompt : API REST online, plusieurs parties et résolution

```
CHOIX O1 ET ARCHITECTURE HÉBERGÉE :
- `territories[].id` est l'unique identité territoriale publique : trigramme,
  sans `code` territorial dupliqué ni matricule `T<number>`.
- Le MVP accepte plusieurs parties, chacune de deux à huit joueurs online, et
  conserve les routes `/api/games/{id}`.
- `chain: null` signifie absence de chaîne ; `visibility: "hidden"` masque une
  chaîne existante ; `visibility: "known"` accompagne un détail connu.
- Les combats utilisent `visibility: "exact"` ou `"general"` ; la vue générale
  n'expose ni forces ni identifiants d'armées.
- L'identité de production vient de Firebase Authentication et est placée dans
  le contexte par le middleware JWT ; aucun champ joueur fourni par le client
  ne fait autorité.
- Firestore sera ajouté par O8. O5 doit déjà exposer des interfaces de store et
  des révisions qui permettront les transactions sans modifier le moteur.

Tu travailles sur "Crown & Borough", un jeu de stratégie par tours. Le moteur
v1 existe et est testé : modèles, carte, parser, résolution, hiver, cycle des
saisons, rapports et endpoints hotseat. Références : `specs/gdd.md`,
`specs/architecture.md`, `specs/online.md` et `specs/online-plan.md`.

PÉRIMÈTRE : construire une API REST `net/http` pour plusieurs parties en
mémoire, avec soumissions d'ordres par joueur, résolution automatique quand
tous les joueurs vivants ont soumis, résolution forcée, rapports, saisons,
élimination et projections privées. O5 ne persiste pas encore dans Firestore,
mais ne doit pas conserver l'ancien singleton `ActiveGame` ni la réponse
`409` pour une deuxième partie. O6 remplacera l'acteur de test par Firebase
Auth ; O8 branchera la persistance transactionnelle.

RÈGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages, enums). Seules les chaînes de contenu de jeu et les labels UI peuvent
être en français.

1. ROUTAGE ET ACTEURS :
   - Utiliser uniquement `net/http` et `http.ServeMux`.
   - Le store expose `Games map[GameID]*GameSession` ou une interface
     équivalente, protégée par un mutex global pour l'index et un mutex par
     partie pour les mutations.
   - Les handlers reçoivent un `Actor` depuis le contexte. En O5, un adaptateur
     de test explicite peut fournir cet acteur ; aucun endpoint public ne doit
     faire confiance à un champ `player` arbitraire.
   - Les endpoints hotseat qui acceptent `?player=` ou un joueur dans le corps
     restent derrière un flag de développement et ne sont jamais montés en
     production.

2. MODÈLE SERVEUR :
   - `GameSession` contient `ID`, `Name`, `Seed`, `State`, `Players`, `Status`,
     `InviteCode` réservé à O6, `TurnSubmissions`, `Reports`, `Winner`,
     `PrivacyMetadata` et une `Revision` monotone.
   - `GameStore` indexe plusieurs parties par ID et fournit `Create`, `List`,
     `Get`, `Join`, `Submit`, `Resolve` et les projections nécessaires.
   - Les IDs de parties sont des UUID v4 ; une collision renvoie une erreur
     interne et ne remplace jamais une partie existante.
   - Chaque partie possède son propre mutex et ses propres soumissions ; une
     mutation d'une partie ne peut pas modifier une autre partie.
   - Le moteur reste pur : les handlers appellent les fonctions de résolution
     et copient les rapports et projections sans I/O dans `internal/engine`.

3. ENDPOINTS :
   - `POST /api/games` crée une partie avec une seed et deux à huit slots dans
     le mode de test O5. La future route authentifiée O6 utilisera le profil et
     ajoutera le créateur comme premier membre.
   - `GET /api/games` liste les parties visibles par l'acteur courant ; en O5,
     le filtre peut utiliser le membership de test, mais il ne doit pas
     retourner des parties d'un autre acteur.
   - `GET /api/games/{id}` renvoie le statut, les slots, le tour, la saison,
     `submitted` et la révision ; un ID inconnu renvoie `404`.
   - `GET /api/games/{id}/map` renvoie la carte immuable.
   - `GET /api/games/{id}/state` renvoie la projection du joueur courant. Le
     paramètre `?player=` n'existe que dans le mode dev explicite.
   - `GET /api/games/{id}/supply?territory=XXX` calcule la ligne ou la zone de
     ravitaillement pour la partie et le joueur autorisés.
   - `POST /api/games/{id}/orders` valide et remplace la soumission du joueur
     courant pour le tour courant. Le joueur courant vient de l'acteur et non
     du corps en production.
   - `POST /api/games/{id}/resolve` utilise les soumissions présentes et des
     ordres vides pour les joueurs manquants.
   - `GET /api/games/{id}/reports` et `/reports/{index}` renvoient les rapports
     filtrés pour le joueur courant.
   - `GET /api/rules?lang=fr` renvoie les règles publiques.

4. SOUMISSION ET RÉSOLUTION :
   - Parser les ordres avant de modifier le store. Une erreur de syntaxe
     renvoie `400` avec la ligne et ne change aucune soumission.
   - Une resoumission valide remplace uniquement celle du joueur et du tour
     courant.
   - Lorsque tous les joueurs vivants ont soumis, appeler la résolution une
     seule fois sous le mutex de la partie. La réponse du dernier submit peut
     contenir `status: "resolved"`, le rapport et la projection.
   - Sinon renvoyer `status: "pending"`, `submitted` et `remaining`.
   - Une résolution forcée est explicite, idempotente et ne crée pas de
     deadline automatique.
   - Un joueur éliminé est considéré comme ayant soumis et ne peut plus
     poster. La partie se termine lorsqu'il ne reste qu'un joueur vivant.
   - Incrémenter `Revision` à chaque mutation persistable pour préparer les
     préconditions Firestore d'O8.

5. PLUSIEURS PARTIES :
   - Créer deux parties avec le même seed doit produire deux IDs et deux mutex
     indépendants ; leurs états et soumissions ne doivent jamais se mélanger.
   - `GET /api/games` ne renvoie que les parties de l'acteur courant. Une
     deuxième création n'est pas un conflit global et ne renvoie pas `409`.
   - Les limites de joueurs s'appliquent par partie. Une partie pleine renvoie
     `409` sans affecter les autres parties.
   - Les endpoints prennent toujours l'ID de partie dans le chemin ; aucune
     route singleton ne doit être nécessaire au contrat online.

6. ERREURS :
   - Utiliser `{ "error": "code", "message": "..." }` et `details` pour les
     validations structurées.
   - `400` requête invalide, `401` acteur absent ou invalide, `403` non-membre,
     `404` ressource inconnue, `409` conflit de slot, de tour ou d'état.
   - Ne pas exposer les ordres bruts, les privacy metadata ou la vue d'un autre
     joueur dans les erreurs.

7. TESTS :
   - CRUD de deux parties, liste filtrée, détail, carte et `404`.
   - Soumission en attente, remplacement, parsing invalide, dernier joueur,
     résolution forcée, quatre saisons, élimination, victoire et rapports.
   - Vues distinctes P1/P2/P3, chaînes masquées et combats exacts/généraux.
   - Deux parties avec même seed : déterminisme par partie et absence de fuite
     d'état.
   - Concurrence : deux soumissions, une resoumission et une résolution sur la
     même partie ; aucune double résolution et aucun data race avec `-race`.
   - Tests de contrat vérifiés par les fixtures O1-O4.

CRITÈRES D'ACCEPTATION :
- `go test -race ./...` et `go vet ./...` passent ;
- deux parties peuvent progresser indépendamment en mémoire ;
- aucun endpoint de production ne fait confiance à une identité de joueur
  fournie dans le JSON ou la query string ;
- le moteur `internal/engine` n'est pas modifié pour la persistance ;
- la `Revision` et les interfaces du store permettent le branchement Firestore
  et ses transactions en O8.

Note : documente dans la réponse finale les choix sur les acteurs de test, le
store multi-parties, les révisions, la résolution synchrone et les routes
dev-only. Ne commit pas sans instruction explicite.
```
