# Prompt : parcours front Firebase, multi-parties et temps reel

```
CHOIX O1 ET ARCHITECTURE HEBERGEE :
- Le MVP accepte plusieurs parties de deux a huit joueurs.
- Firebase Authentication utilise un lien de connexion par email ; le SDK
  Firebase gere la session et le rafraichissement des ID tokens.
- Le backend Go derive toujours l'identite du JWT Firebase et les commandes
  passent par l'API REST.
- Firestore expose au navigateur uniquement `games/{id}` et la projection
  `games/{id}/views/{uid}` ainsi que les rapports filtres du joueur ; le front
  ne lit jamais l'etat canonique, les soumissions brutes ou les privacy
  metadata.
- Les listeners `onSnapshot` remplacent le polling regulier. Ils sont detaches
  lorsque la partie, le compte ou le composant disparait.
- Les chaines et combats suivent les variantes `null`, `hidden`, `known`,
  `exact` et `general` des fixtures O1-O4.

Tu travailles sur "Crown & Borough", un jeu de strategie par tours. Le moteur,
l'API de partie, les projections privees, Firebase Auth et la persistence
Firestore sont les couches de produit. References : `specs/gdd.md`,
`specs/architecture.md`, `specs/online.md` et `specs/online-plan.md`.

PERIMETRE : construire le parcours utilisateur complet entre amis : connexion
email, profil, liste multi-parties, creation, invitation, join, consultation
temps reel, ordres, resolution, rapports, hiver, victoire et erreurs. Ne pas
mettre de logique de confidentialite dans React : le front rend les projections
deja filtrees et ne doit pas essayer de reconstruire l'etat prive.

REGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages techniques, enums). Les labels et messages de l'interface restent en
francais.

1. CONFIGURATION ET SESSION :
   - Ajouter le Firebase Web SDK et initialiser l'application avec les
     variables `VITE_FIREBASE_API_KEY`, `VITE_FIREBASE_AUTH_DOMAIN`,
     `VITE_FIREBASE_PROJECT_ID` et `VITE_FIREBASE_APP_ID`.
   - Les valeurs Firebase Web sont publiques ; ne jamais mettre de credential
     Admin ou de cle de service dans `web/`.
   - `useAuth` expose l'utilisateur courant, l'etat de chargement et les
     actions `sendSignInLink`, `completeSignIn` et `signOut`.
   - Utiliser `onAuthStateChanged` et la persistence de session Firebase. Ne
     pas copier l'ID token dans `localStorage` ; appeler `getIdToken()` juste
     avant une commande REST.
   - Le lien email doit fonctionner sur le domaine local autorise et sur le
     domaine Cloud Run. L'adresse email necessaire a la finalisation peut etre
     conservee dans le stockage de session du navigateur.

2. PROFIL ET ACCUEIL :
   - Apres connexion, appeler `GET /api/auth/me`. Si le display name manque,
     afficher le formulaire de profil et appeler `PUT /api/auth/me`.
   - Valider le nom cote client pour l'UX, mais laisser le serveur trancher la
     validation et les conflits.
   - L'accueil affiche les parties dont l'utilisateur est membre et une action
     de creation. La liste vient de l'API pour l'initialisation et peut etre
     abonnee directement a la query Firestore securisee
     `games where memberUids array-contains uid`.
   - Les parties d'autres joueurs ne sont jamais affichees comme accessibles.
     Un lien d'invitation preselectionne la partie et le code, puis demande la
     connexion avant d'appeler `/join`.

3. CREATION, INVITATION ET JOIN :
   - Creation : nom de partie et seed optionnelle, puis `POST /api/games` avec
     l'ID token. Afficher le code et le lien au createur uniquement.
   - Join : envoyer uniquement `{ inviteCode }` a
     `POST /api/games/{id}/join`. Ne jamais envoyer un nom pour choisir le slot
     et ne jamais permettre la selection libre d'un `player`.
   - Afficher les slots, les noms de profil autorises et le statut des
     soumissions sans exposer les ordres ou les donnees privees.
   - Le parcours doit supporter plusieurs onglets et plusieurs parties sans
     melanger leurs listeners ou leur etat local.

4. LISTENERS FIRESTORE :
   - `useGameSubscription(gameId, uid)` s'abonne au resume public
     `games/{gameId}` et a la vue `games/{gameId}/views/{uid}`.
   - Les rapports filtres historiques peuvent etre charges par l'API ou par
     une souscription Firestore limitee au chemin du UID courant.
   - Verifier `revision`, `turn` et `updatedAt` pour ignorer un snapshot plus
     ancien qu'une reponse REST deja appliquee.
   - Le listener ne lit jamais `canonical`, `turns`, `submissions`, les
     rapports non filtres ou les invitations.
   - Desabonner chaque `onSnapshot` dans le cleanup React. Desabonner aussi
     lorsqu'un compte se deconnecte, change de partie ou perd son membership.
   - Ne pas implementer un poller de secours permanent. Une reconnexion reseau
     refait une lecture REST ponctuelle puis laisse le listener reprendre.
   - Les erreurs Firestore `permission-denied`, `unauthenticated` et reseau
     sont converties en etats d'interface explicites.

5. ORDRES ET RESOLUTION :
   - Charger la carte statique une fois par partie ; elle ne contient aucun
     secret et ne doit pas etre refetchee a chaque snapshot.
   - Editer les chaines par noble du joueur et les ordres d'hiver dans un etat
     local. La soumission utilise `POST /api/games/{id}/orders` avec l'ID token
     et aucun champ d'identite joueur.
   - Afficher `pending`, les joueurs restants et le tour depuis la projection
     publique. Une resoumission remplace le brouillon distant du meme joueur.
   - Le joueur qui soumet en dernier peut recevoir `resolved` dans la reponse ;
     les autres voient la nouvelle projection via `onSnapshot`.
   - La resolution forcee reste une action explicite avec confirmation ; elle
     n'ajoute aucune deadline automatique.

6. VUES PRIVEES ET RAPPORTS :
   - Rendre `chain: null`, `chain.visibility: hidden` et
     `chain.visibility: known` comme trois etats differents.
   - Rendre un combat `exact` avec ses details uniquement lorsque la projection
     le fournit ; rendre un combat `general` sans tenter de reconstituer les
     forces ou identifiants absents.
   - Ne jamais conserver une copie globale non filtree dans un store React,
     dans les devtools de l'application ou dans un cache persistant.
   - Afficher rapports, mouvements, combats, ravitaillement, famine, nobles,
     investissements d'hiver et victoire selon le contrat de projection.

7. ETATS ET ACCESSIBILITE :
   - Couvrir chargement auth, attente du lien email, profil manquant, liste
     vide, partie introuvable, partie pleine, acces interdit, reseau coupe,
     listener en erreur et deconnexion.
   - Afficher un etat de connexion temps reel sans bloquer l'edition locale des
     ordres ; prevenir clairement avant de perdre une soumission non envoyee.
   - Composants decoupes : Auth, Profile, GameList, GameLobby, Map, Orders,
     Report et GameStatus. Les hooks gerent auth, REST et listeners ; les
     composants ne contiennent pas de logique moteur.
   - Focus visible, labels accessibles, contrastes corrects et largeur mobile.

8. TESTS :
   - Vitest : mocks de Firebase Auth, `onAuthStateChanged`, `onSnapshot`,
     desabonnement, refresh d'ID token et erreurs de permission.
   - Tester deux parties dans deux onglets, un changement instantane de slot,
     une resolution vue par l'autre joueur, un changement de compte et un
     listener detache.
   - Tester le parcours du lien email avec l'emulateur ou un adaptateur de
     test ; ne pas dependre d'un email reel dans la CI.
   - Executer `npm run test`, `npm run build` et `npm run lint`.
   - Documenter dans `TESTING.md` le parcours manuel avec deux comptes, deux
     parties, une reconnexion et une largeur mobile.

CRITERES D'ACCEPTATION :
- Deux comptes se connectent par lien email et jouent dans deux parties sans
  melange d'etat.
- Une soumission apparait en temps reel pour les membres autorises sans
  polling permanent.
- Les rapports et vues privees correspondent au joueur connecte et aucune
  donnee brute n'est conservée ou affichée côté front.
- Un refresh et une reconnexion Firebase retrouvent le profil, les memberships
  et la partie depuis Firestore.
- `npm run test`, `npm run build` et `npm run lint` passent.

Note : documente dans la reponse finale la configuration Firebase Web, la
persistence de session choisie, les chemins Firestore ecoutes, la strategie de
cleanup et les tests de fuite. Ne commit pas sans instruction explicite.
```
