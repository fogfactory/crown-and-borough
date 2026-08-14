# Prompt : Firebase Authentication, profils et invitations

```
CHOIX O1 ET ARCHITECTURE HEBERGEE :
- Le MVP accepte plusieurs parties de deux a huit joueurs et conserve les
  routes `/api/games/{id}`.
- Firebase Authentication utilise le lien de connexion par email ; aucun mot
  de passe n'est gere par Crown & Borough.
- Le serveur valide les ID tokens Firebase avec Firebase Admin SDK. Il ne genere
  ni ne stocke de token Bearer maison.
- Firestore persiste `players/{uid}`, les memberships, les parties et les
  projections. Les secrets d'authentification ne sont jamais copies dans
  Firestore.
- Le front utilise la persistence de session du SDK Firebase et `getIdToken()`
  pour les commandes REST. Il n'ecrit pas manuellement un ID token dans
  `localStorage`.
- Un code d'invitation opaque de six caracteres reste une capability de join,
  mais ne remplace ni le compte Firebase ni le controle d'appartenance.

Tu travailles sur "Crown & Borough", un jeu de strategie par tours. L'API REST
de jeu et les projections privees sont les couches precedentes. References :
`specs/gdd.md`, `specs/architecture.md`, `specs/online.md` et
`specs/online-plan.md`.

PERIMETRE : remplacer l'inscription maison et les sessions en memoire par
Firebase Authentication, un profil joueur persistant et des memberships
Firestore. Le compte Firebase est l'identite stable ; le nom affiché est une
propriete de profil modifiable sous validation. Les invitations ne doivent pas
permettre de rejoindre une partie sans authentification ni de lire une partie
dont le joueur n'est pas membre.

REGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages, enums). Seules les chaines de contenu de jeu et les labels UI peuvent
etre en francais.

1. PARCOURS FIREBASE AUTH :
   - Le front configure `initializeApp` avec les variables publiques Firebase
     `apiKey`, `authDomain`, `projectId` et `appId`. Ces valeurs ne sont pas des
     secrets ; aucun service account JSON ne doit etre embarque.
   - L'utilisateur saisit son email. Le front appelle
     `sendSignInLinkToEmail` avec une URL de continuation HTTPS autorisee et
     conserve l'email necessaire a la finalisation dans le stockage de session
     du navigateur.
   - Lorsque le lien est ouvert, le front appelle `signInWithEmailLink`, puis
     laisse Firebase gerer le rafraichissement de l'ID token et la persistence
     de la session (`browserLocalPersistence` ou persistence equivalente).
   - Pour chaque commande REST, le front obtient `getIdToken()` et envoie
     `Authorization: Bearer <firebase-id-token>`. Il ne fabrique jamais le
     header a partir d'un nom de joueur.
   - Un changement d'authentification remet a jour le profil et detache les
     listeners Firestore de la session precedente.
   - Les domaines autorises, le template email et les limites de Firebase Auth
     sont documentes dans la procedure de deploiement.

2. VALIDATION SERVEUR :
   - Initialiser Firebase Admin SDK avec les credentials ADC et le project ID
     attendu par l'environnement ; ne jamais accepter un project ID fourni par
     le client.
   - Le middleware protege toutes les routes de partie, de profil et de
     commande. Il refuse un header absent, un JWT invalide, expire, mal signe,
     revoque ou emis par un autre projet avec `401`.
   - Le middleware place dans le contexte le Firebase UID et les claims
     minimaux verifies. Les handlers ne lisent jamais un champ `uid`, `player`
     ou `globalPlayerId` pour choisir l'identite appelante.
   - La verification doit utiliser le cache officiel des cles et un timeout ;
     une panne de verification ne doit pas devenir un acces anonyme.

3. PROFIL PERSISTANT :
   - `GET /api/auth/me` verifie le token, lit `players/{uid}` et renvoie
     `{ player: { uid, email, displayName } }`. Le premier appel peut creer un
     profil minimal cote serveur avec l'email issu du token.
   - `PUT /api/auth/me` accepte `{ displayName }`, applique la validation de
     longueur et de normalisation, puis ecrit uniquement le profil du UID du
     token. Un nom vide ou duplique selon la politique choisie renvoie `400` ou
     `409` sans mutation partielle.
   - L'email et le UID sont en lecture seule pour l'application ; ils viennent
     du token Firebase. Le display name ne devient jamais une cle de securite.
   - Les profiles sont idempotents : deux premiers appels concurrents ne
     doivent pas creer deux documents ni perdre le nom valide.

4. INVITATIONS ET MEMBERSHIP :
   - `POST /api/games` exige un profil valide, genere un UUID de partie et un
     code d'invitation de six caracteres dans un alphabet sans ambiguite.
   - Stocker uniquement le hash du code dans `invitations/{codeHash}` avec
     `gameId`, `createdBy`, `active` et les dates utiles. Le code en clair est
     renvoye une seule fois au createur dans la reponse et n'est pas place dans
     `games/{gameId}` lisible par les membres.
   - Le createur devient le premier slot et son UID est ajoute atomiquement a
     `memberUids`. Une partie accepte deux a huit slots.
   - `POST /api/games/{id}/join` exige le JWT Firebase et `{ inviteCode }`.
     Le backend hache le code, verifie qu'il correspond a la partie, puis
     ajoute le UID courant au premier slot libre dans une transaction.
   - Un membre deja present obtient une reponse idempotente. Un non-membre ne
     peut pas reprendre le slot d'un autre par son nom. Une partie pleine,
     terminee ou un code invalide renvoie `409` ou `403` selon le conflit
     precise documente par l'API.
   - Le lien contient l'ID de partie et le code opaque, mais sa possession ne
     donne pas acces aux documents Firestore ni aux commandes sans JWT valide.
   - La liste `/api/games` et les listeners de collection utilisent
     `memberUids array-contains uid` ; aucune partie hors membership n'est
     renvoyee.

5. CONTROLE D'ACCES :
   - Un membre peut lire la carte, le resume, sa projection d'etat, son rapport
     et son ravitaillement.
   - Le code d'invitation et le lien complet sont visibles par le createur via
     l'API de detail ou une route dediee, jamais par la projection Firestore
     commune.
   - `POST /api/games/{id}/orders` derive le joueur du contexte Firebase ; le
     corps ne contient plus de champ `player`.
   - `GET /api/games/{id}/state` et les rapports utilisent le UID du contexte
     pour projeter les donnees. `?player=` reste uniquement dans le serveur de
     test local, jamais dans le deploiement public.
   - Un utilisateur authentifie mais non membre obtient `403`, y compris s'il
     connait un ID de partie valide. Un utilisateur non authentifie obtient
     `401` sur toute route protegee.

6. TESTS :
   - Utiliser l'emulateur Firebase Auth pour les liens email et les ID tokens,
     et l'emulateur Firestore pour les profils, memberships et regles.
   - Tester lien valide, lien deja utilise, email absent, token expire, token
     d'un autre projet, token invalide, requete sans bearer et changement de
     compte dans le meme navigateur.
   - Tester creation, code unique, join correct, code incorrect, partie pleine,
     membre deja present, compte exterieur et deux joins concurrents sur le
     dernier slot.
   - Tester qu'un UID ne peut pas modifier le profil, soumettre des ordres,
     lire une vue ou lire un rapport pour un autre UID.
   - Tester qu'aucun ID token, refresh token ou code d'invitation en clair n'est
     ecrit dans Firestore ou dans les logs.

CRITERES D'ACCEPTATION :
- Deux comptes Firebase jouent depuis deux navigateurs via un lien email et un
  lien d'invitation.
- `GET /api/auth/me` retrouve le profil apres redemarrage du serveur.
- Les ID tokens sont verifies cote serveur et l'UID n'est jamais fourni par le
  client.
- Un joueur ne peut ni lire ni soumettre pour un autre et un lien seul ne
  contourne pas l'authentification.
- La session du navigateur est restauree par Firebase sans stockage maison de
  tokens ; les profils et memberships persistent dans Firestore.

Note : documente dans la reponse finale le domaine Firebase, la persistence
client choisie, les routes de profil, le hash d'invitation, les claims utilises
et les regles d'acces. Ne commit pas sans instruction explicite.
```
