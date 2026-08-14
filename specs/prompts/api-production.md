# Prompt : API REST online, plusieurs parties et resolution

```
CHOIX O1 ET ARCHITECTURE HEBERGEE :
- `territories[].id` est l'unique identite territoriale publique : trigramme,
  sans `code` territorial duplique ni matricule `T<number>`.
- Le MVP accepte plusieurs parties, chacune de deux a huit joueurs online, et
  conserve les routes `/api/games/{id}`.
- `chain: null` signifie absence de chaine ; `visibility: "hidden"` masque une
  chaine existante ; `visibility: "known"` accompagne un detail connu.
- Les combats utilisent `visibility: "exact"` ou `"general"` ; la vue generale
  n'expose ni forces ni identifiants d'armees.
- L'identite de production vient de Firebase Authentication et est placee dans
  le contexte par le middleware JWT ; aucun champ joueur fourni par le client
  ne fait autorite.
- Firestore sera ajoute par O8. O5 doit deja exposer des interfaces de store et
  des revisions qui permettront les transactions sans modifier le moteur.

Tu travailles sur "Crown & Borough", un jeu de strategie par tours. Le moteur
v1 existe et est teste : modeles, carte, parser, resolution, hiver, cycle des
saisons, rapports et endpoints hotseat. References : `specs/gdd.md`,
`specs/architecture.md`, `specs/online.md` et `specs/online-plan.md`.

PERIMETRE : construire une API REST `net/http` pour plusieurs parties en
memoire, avec soumissions d'ordres par joueur, resolution automatique quand
tous les joueurs vivants ont soumis, resolution forcee, rapports, saisons,
elimination et projections privees. O5 ne persiste pas encore dans Firestore,
mais ne doit pas conserver l'ancien singleton `ActiveGame` ni la reponse
`409` pour une deuxieme partie. O6 remplacera l'acteur de test par Firebase
Auth ; O8 branchera la persistence transactionnelle.

REGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages, enums). Seules les chaines de contenu de jeu et les labels UI peuvent
etre en francais.

1. ROUTAGE ET ACTEURS :
   - Utiliser uniquement `net/http` et `http.ServeMux`.
   - Le store expose `Games map[GameID]*GameSession` ou une interface
     equivalente, protegee par un mutex global pour l'index et un mutex par
     partie pour les mutations.
   - Les handlers recoivent un `Actor` depuis le contexte. En O5, un adaptateur
     de test explicite peut fournir cet acteur ; aucun endpoint public ne doit
     faire confiance a un champ `player` arbitraire.
   - Les endpoints hotseat qui acceptent `?player=` ou un joueur dans le corps
     restent derriere un flag de developpement et ne sont jamais montes en
     production.

2. MODELE SERVEUR :
   - `GameSession` contient `ID`, `Name`, `Seed`, `State`, `Players`, `Status`,
     `InviteCode` reserve a O6, `TurnSubmissions`, `Reports`, `Winner`,
     `PrivacyMetadata` et une `Revision` monotone.
   - `GameStore` indexe plusieurs parties par ID et fournit `Create`, `List`,
     `Get`, `Join`, `Submit`, `Resolve` et les projections necessaires.
   - Les IDs de parties sont des UUID v4 ; une collision renvoie une erreur
     interne et ne remplace jamais une partie existante.
   - Chaque partie possede son propre mutex et ses propres soumissions ; une
     mutation d'une partie ne peut pas modifier une autre partie.
   - Le moteur reste pur : les handlers appellent les fonctions de resolution
     et copient les rapports et projections sans I/O dans `internal/engine`.

3. ENDPOINTS :
   - `POST /api/games` cree une partie avec une seed et deux a huit slots dans
     le mode de test O5. La future route authentifiee O6 utilisera le profil et
     ajoutera le createur comme premier membre.
   - `GET /api/games` liste les parties visibles par l'acteur courant ; en O5,
     le filtre peut utiliser le membership de test, mais il ne doit pas
     retourner des parties d'un autre acteur.
   - `GET /api/games/{id}` renvoie le statut, les slots, le tour, la saison,
     `submitted` et la revision ; un ID inconnu renvoie `404`.
   - `GET /api/games/{id}/map` renvoie la carte immuable.
   - `GET /api/games/{id}/state` renvoie la projection du joueur courant. Le
     parametre `?player=` n'existe que dans le mode dev explicite.
   - `GET /api/games/{id}/supply?territory=XXX` calcule la ligne ou la zone de
     ravitaillement pour la partie et le joueur autorises.
   - `POST /api/games/{id}/orders` valide et remplace la soumission du joueur
     courant pour le tour courant. Le joueur courant vient de l'acteur et non
     du corps en production.
   - `POST /api/games/{id}/resolve` utilise les soumissions presentes et des
     ordres vides pour les joueurs manquants.
   - `GET /api/games/{id}/reports` et `/reports/{index}` renvoient les rapports
     filtres pour le joueur courant.
   - `GET /api/rules?lang=fr` renvoie les regles publiques.

4. SOUMISSION ET RESOLUTION :
   - Parser les ordres avant de modifier le store. Une erreur de syntaxe
     renvoie `400` avec la ligne et ne change aucune soumission.
   - Une resoumission valide remplace uniquement celle du joueur et du tour
     courant.
   - Lorsque tous les joueurs vivants ont soumis, appeler la resolution une
     seule fois sous le mutex de la partie. La reponse du dernier submit peut
     contenir `status: "resolved"`, le rapport et la projection.
   - Sinon renvoyer `status: "pending"`, `submitted` et `remaining`.
   - Une resolution forcee est explicite, idempotente et ne cree pas de
     deadline automatique.
   - Un joueur elimine est considere comme ayant soumis et ne peut plus
     poster. La partie se termine lorsqu'il ne reste qu'un joueur vivant.
   - Incrementer `Revision` a chaque mutation persistable pour preparer les
     preconditions Firestore d'O8.

5. PLUSIEURS PARTIES :
   - Creer deux parties avec le meme seed doit produire deux IDs et deux mutex
     independants ; leurs etats et soumissions ne doivent jamais se melanger.
   - `GET /api/games` ne renvoie que les parties de l'acteur courant. Une
     deuxieme creation n'est pas un conflit global et ne renvoie pas `409`.
   - Les limites de joueurs s'appliquent par partie. Une partie pleine renvoie
     `409` sans affecter les autres parties.
   - Les endpoints prennent toujours l'ID de partie dans le chemin ; aucune
     route singleton ne doit etre necessaire au contrat online.

6. ERREURS :
   - Utiliser `{ "error": "code", "message": "..." }` et `details` pour les
     validations structurées.
   - `400` requete invalide, `401` acteur absent ou invalide, `403` non-membre,
     `404` ressource inconnue, `409` conflit de slot, de tour ou d'etat.
   - Ne pas exposer les ordres bruts, les privacy metadata ou la vue d'un autre
     joueur dans les erreurs.

7. TESTS :
   - CRUD de deux parties, liste filtre, detail, carte et `404`.
   - Soumission en attente, remplacement, parsing invalide, dernier joueur,
     resolution forcee, quatre saisons, elimination, victoire et rapports.
   - Vues distinctes P1/P2/P3, chaines masquées et combats exacts/generaux.
   - Deux parties avec meme seed : determinisme par partie et absence de fuite
     d'etat.
   - Concurrence : deux soumissions, une resoumission et une resolution sur la
     meme partie ; aucune double resolution et aucun data race avec `-race`.
   - Tests de contrat verifies par les fixtures O1-O4.

CRITERES D'ACCEPTATION :
- `go test -race ./...` et `go vet ./...` passent ;
- deux parties peuvent progresser independamment en memoire ;
- aucun endpoint de production ne fait confiance a une identite de joueur
  fournie dans le JSON ou la query string ;
- le moteur `internal/engine` n'est pas modifie pour la persistence ;
- la `Revision` et les interfaces du store permettent le branchement Firestore
  et ses transactions en O8.

Note : documente dans la reponse finale les choix sur les acteurs de test, le
store multi-parties, les revisions, la resolution synchrone et les routes
dev-only. Ne commit pas sans instruction explicite.
```
