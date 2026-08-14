# Prompt : persistance Firestore, transactions et projections

```
CHOIX O1 ET ARCHITECTURE HEBERGEE :
- `territories[].id` est l'unique identite territoriale publique : trigramme,
  sans `code` territorial duplique ni matricule `T<number>`.
- Le MVP accepte plusieurs parties, chacune pour deux a huit joueurs online,
  et conserve les routes `/api/games/{id}`.
- `chain: null` signifie absence de chaine ; `visibility: "hidden"` masque une
  chaine existante ; `visibility: "known"` accompagne un detail connu.
- Les combats utilisent `visibility: "exact"` ou `"general"` ; la vue generale
  n'expose ni forces ni identifiants d'armees.
- Firebase Authentication fournit l'identite par lien email. Le backend valide
  l'ID token Firebase porte en `Authorization: Bearer` avec Firebase Admin SDK.
- Les parties et profils sont persistés dans Firestore Native mode. Aucun ID
  token, refresh token ou secret de session n'est copie dans Firestore.
- Le front utilise `onSnapshot` uniquement sur des projections lisibles selon
  les regles Firestore ; les ordres passent par l'API Go.
- La resolution forcee est explicite, sans deadline de jeu. Les leases
  internes de reprise apres crash ne sont pas des deadlines de jeu.
- Le serveur utilise `net/http` et `http.ServeMux`, et les endpoints hotseat
  restent dev-only.

Tu travailles sur "Crown & Borough", un jeu de strategie asynchrone par tours.
L'API REST de plusieurs parties, les projections privees et Firebase Auth sont
les couches online precedentes. Le moteur est pur, deterministe et
serialisable. References : `specs/architecture.md`, `specs/online.md`,
`specs/online-plan.md` et `specs/roadmap.md`.

PERIMETRE : remplacer la persistance locale ou GCS FUSE prevue par une
persistance Firestore Native mode. Toute mutation de partie, soumission en
attente, resolution, rapport, historique borne et privacy metadata doit survivre
au redemarrage d'une instance Cloud Run. Le store doit permettre plusieurs
parties et ne doit pas supposer qu'une seule instance Cloud Run est toujours
active.

REGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages, enums). Seules les chaines de contenu de jeu et les labels UI peuvent
etre en francais.

1. CLIENT FIRESTORE ET RESPONSABILITES :
   - Utiliser le SDK Cloud Firestore cote serveur avec les credentials ADC du
     service account Cloud Run ; ne jamais charger un fichier de cle dans le
     conteneur.
   - Isoler l'adaptateur dans `internal/store` ou un package equivalent ;
     `internal/engine` ne depend ni de Firestore ni de Firebase Auth.
   - Le store garde des types de persistence explicites et une `schemaVersion`
     pour les documents ; une migration future doit etre detectee avant de
     charger un document inconnu.
   - Les lectures et ecritures du backend sont soumises a un contexte avec
     timeout ; une erreur Firestore ne doit jamais etre transformee en succes
     HTTP.

2. MODELE DE DOCUMENTS :
   - `players/{uid}` : `{ uid, email, displayName, createdAt, updatedAt }`.
     L'email est une donnee privee ; le client ne lit que son propre profil.
   - `games/{gameId}` : resume lisible par les membres :
     `{ schemaVersion, id, name, ownerUid, memberUids, players, status,
        turn, season, winnerUid, submittedUids, revision, createdAt,
        updatedAt }`. Ne jamais y mettre l'etat moteur, les ordres, le code
     d'invitation ou les donnees privees d'un autre joueur.
   - `games/{gameId}/canonical/current` : document reserve au backend avec le
     `GameState`, la carte necessaire au calcul, les rapports non filtres,
     `privacyMetadata`, les soumissions attendues et l'etat de resolution.
   - `games/{gameId}/turns/{turn}/submissions/{uid}` : ordres bruts du joueur
     pour le tour et `submittedAt`. Ce chemin est reserve au backend.
   - `games/{gameId}/reports/{turn}` : rapport non filtre reserve au backend.
   - `games/{gameId}/views/{uid}` : projection `state` privee pour ce joueur,
     avec `revision`, `turn`, `season` et `updatedAt`.
   - `games/{gameId}/reports/{uid}/turns/{turn}` : rapport filtre lisible par
     le joueur du chemin uniquement.
   - `invitations/{codeHash}` : `{ gameId, createdBy, expiresAt, active }`.
     Le code opaque de six caracteres n'est jamais persiste en clair et n'est
     jamais publie dans un document lisible par un membre non proprietaire.

3. REGLES FIRESTORE :
   - Les clients authentifies peuvent lire `games/{gameId}` si leur UID est
     dans `memberUids`.
   - Un client peut lire `games/{gameId}/views/{uid}` et
     `games/{gameId}/reports/{uid}/turns/{turn}` uniquement si `uid` est son
     propre UID et qu'il est membre de la partie.
   - Un client peut lire son propre `players/{uid}` seulement. Les mises a jour
     de profil passent par `PUT /api/auth/me` afin que l'email, l'UID et les
     validations viennent du token et du backend, pas d'une ecriture Firestore
     directe.
   - Les clients ne peuvent jamais lire ou ecrire `canonical`, `turns`, les
     soumissions, les rapports non filtres, les invitations ou les privacy
     metadata.
   - Les clients ne peuvent jamais ecrire directement `games`, `views` ou les
     rapports. Toutes les commandes de jeu passent par l'API Go.
   - Tester les regles avec l'emulateur Firebase/Firestore pour un membre, un
     non-membre, un autre UID et une requete non authentifiee.

4. SOUMISSION ATOMIQUE :
   - Parser et valider les ordres hors transaction ; une erreur de parsing ne
     doit rien ecrire.
   - Dans une transaction Firestore, lire le resume, le document canonique et
     la soumission du joueur pour le tour courant.
   - Refuser une partie inconnue, un joueur non membre, un tour obsolete ou un
     joueur elimine.
   - Remplacer la soumission du joueur dans
     `turns/{turn}/submissions/{uid}` et mettre a jour `submittedUids` dans le
     resume uniquement si la revision lue est toujours la bonne.
   - La transaction doit etre idempotente : une resoumission valide remplace
     l'ancienne et une repetition de la meme requete ne cree pas deux ordres.
   - Apres le commit, publier ou rafraichir la projection publique de statut ;
     le contenu des ordres reste reserve au backend.

5. RESOLUTION ATOMIQUE ET REPRISE :
   - Une transaction lit les joueurs vivants, les soumissions du tour et la
     revision du document canonique.
   - Si toutes les soumissions sont presentes, ou si `/resolve` est appele,
     elle pose une revendication `{ status: "claimed", operationId,
     claimedAt, leaseUntil, baseRevision }`. Une seule revendication peut
     reussir pour une revision.
   - Le moteur pur calcule le resultat a partir d'un snapshot immuable. Il peut
     etre rejoue si Firestore relance une transaction ; aucun effet externe ne
     doit etre execute pendant ce calcul.
   - Un commit conditionnel verifie `operationId`, `baseRevision` et le tour,
     puis ecrit le nouvel etat canonique, le rapport, la revision suivante,
     les projections de tous les joueurs et le resume public, et supprime ou
     archive les soumissions du tour dans une operation coherente.
   - Une revendication dont `leaseUntil` est depasse peut etre reprise par une
     nouvelle transaction. La reprise ne force pas un tour et ne constitue pas
     une deadline pour les joueurs.
   - Une tentative concurrente qui perd la revendication relit le document et
     renvoie l'etat deja resolu ou `pending`, sans appliquer le moteur une
     seconde fois.
   - Respecter les limites Firestore de taille de document, de transaction et
     de nombre d'ecritures ; les projections doivent etre bornees et
     l'historique ne doit pas croitre sans limite.

6. RESTAURATION ET LISTE DES PARTIES :
   - Au demarrage, le serveur ne charge pas un repertoire local : il initialise
     le client Firestore et peut rechercher les parties avec une revendication
     expiree.
   - Une partie valide est relue depuis `canonical/current` et son resume ;
     toute incoherence est signalee et la partie reste indisponible plutot que
     d'etre servie avec un etat partiellement charge.
   - `GET /api/games` utilise une requete `array-contains` sur `memberUids` ou
     une projection d'index equivalente ; il ne renvoie aucune partie dont le
     joueur n'est pas membre.
   - Les rapports historiques sont lus depuis la projection du joueur et ne
     sont jamais reconstruits depuis un rapport non filtre cote client.

7. TESTS :
   - Utiliser l'emulateur Firestore pour les tests Go et l'emulateur Firebase
     Auth pour les ID tokens de test ; aucun projet GCP reel ne doit etre
     necessaire a la suite locale.
   - Sauvegarde et restauration : creation, join, soumission partielle,
     resolution, rapport et reprise apres redemarrage du serveur.
   - Transactions : deux soumissions simultanees, deux resoumissions, deux
     resolutions, une revision obsolete et une revendication expiree.
   - Confidentialite : aucun client ne lit le canonique, une vue d'un autre
     UID, un rapport non filtre, une soumission ou une invitation.
   - Invariants : GameState, joueurs, territoires, rapports, privacy metadata,
     revision monotone et projections sans mutation de la source.
   - Quotas : mesurer les lectures/ecritures pour une partie de deux a huit
     joueurs et plusieurs parties avant le deploiement.

CRITERES D'ACCEPTATION :
- `go test -race ./...` et `go vet ./...` passent avec l'emulateur ;
- le moteur `internal/engine` reste pur et n'importe pas Firestore ;
- une soumission ou une resolution concurrente ne perd pas d'ordre et ne
  double jamais un tour ;
- un redemarrage Cloud Run retrouve l'etat, les soumissions et les rapports
  depuis Firestore ;
- un navigateur membre ne peut observer que le resume public et ses propres
  projections ;
- aucune dependance a `DATA_DIR`, GCS FUSE, Persistent Disk ou `rename` ne
  subsiste dans le backend online.

Note : documente dans la reponse finale le schema Firestore, les regles de
lecture, le mecanisme de revendication, les leases, les limites de taille, les
quotas mesures et les choix de migration. Ne commit pas sans instruction
explicite.
```
