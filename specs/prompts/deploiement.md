# Prompt : deploiement Cloud Run, Firebase et CI

```
CHOIX O1 ET ARCHITECTURE HEBERGEE :
- Le MVP accepte plusieurs parties de deux a huit joueurs.
- Firebase Authentication fournit les liens de connexion email et Firestore
  Native mode fournit la persistence.
- Le serveur Go valide les ID tokens avec Firebase Admin SDK et utilise les
  credentials ADC du service account Cloud Run.
- Le front utilise Firebase Web SDK et des listeners sur des projections
  securisees ; aucun service account ou secret Admin ne va dans l'image frontend.
- Cloud Run est la cible unique du MVP, avec `min-instances=0` et une limite
  initiale d'instances pour controler le cout. La coherence ne depend pas de
  cette limite.
- GCS FUSE, Persistent Disk, `DATA_DIR` et le workflow Compute Engine de
  repli sont hors perimetre.

Tu travailles sur "Crown & Borough", un jeu de strategie par tours. Le moteur,
l'API, les vues privees, Firebase Auth et la persistence Firestore sont les
couches precedentes. References : `specs/architecture.md`, `specs/online.md`,
`specs/online-plan.md` et `specs/roadmap.md`.

PERIMETRE : empaqueter le front et l'API dans une image OCI, deployer sur Cloud
Run, configurer Firebase Authentication et Firestore, et automatiser les tests,
le push et le deploiement. Le but est un lien HTTPS public jouable entre amis,
avec plusieurs parties et restauration apres redemarrage sans volume local.

REGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages techniques, enums). Les labels UI restent en francais.

1. PROJET GCP ET FIREBASE :
   - Utiliser un projet GCP qui heberge Cloud Run, Artifact Registry,
     Firestore et le projet Firebase associe.
   - Activer les APIs necessaires : Cloud Run, Artifact Registry, Firestore,
     Firebase Authentication/Identity Toolkit, Service Usage et IAM.
   - Creer la base Firestore Native mode dans une region documentee et ne pas
     creer une deuxieme base par inadvertance : le free tier est limite a une
     base par projet selon les conditions du service.
   - Configurer le fournisseur email-link Firebase Auth, les domaines autorises
     local et public, et l'URL de continuation Cloud Run.
   - Versionner `firestore.rules` et `firestore.indexes.json`. Deployer les
     regles avant de rendre la configuration frontend publique.
   - Les valeurs Firebase Web (`apiKey`, `authDomain`, `projectId`, `appId`)
     sont des variables publiques de build. Les credentials Admin restent
     fournis par ADC et ne sont jamais ecrits dans GitHub ou le repository.

2. CONTENEUR UNIQUE :
   - Le serveur Go sert l'API `/api/*`, les assets statiques et le fallback SPA.
   - Le Dockerfile multi-stage construit le front avec les variables `VITE_*`
     publiques, puis compile le binaire Go avec `go:embed web/dist`.
   - L'image finale ne contient ni `DATA_DIR`, ni bucket, ni credential JSON.
   - Le serveur ecoute `$PORT` et reste stateless ; toutes les parties et
     projections durables sont dans Firestore.
   - `GET /healthz` verifie seulement que le processus repond.
   - `GET /healthz/ready` verifie que le client Firestore et la configuration
     Firebase Admin sont initialisables, avec un timeout court. Le endpoint ne
     divulgue aucun detail de credential.

3. SERVICE CLOUD RUN :
   - Service public au niveau Cloud Run (`allow-unauthenticated`) : l'acces au
     jeu est protege par Firebase Auth dans l'application, pas par IAM de
     Cloud Run.
   - `min-instances=0`, CPU et memoire minimales mesurees, `max-instances=1`
     pour le premier deploiement et revision unique sans split de trafic.
     Les transactions Firestore doivent permettre d'augmenter cette limite
     plus tard sans changer les invariants.
   - Affecter un service account dedie avec les roles minimaux Firestore et
     Firebase Admin necessaires. Ne pas utiliser le compte par defaut si une
     identite dediee est possible.
   - Configurer `GOOGLE_CLOUD_PROJECT`, le port et les variables de runtime
     non secretes dans le service. Les noms de projet et la region viennent de
     variables GitHub, jamais du code.
   - Desactiver les endpoints hotseat et les modes de test par defaut en
     production.

4. CI/CD GITHUB ACTIONS :
   - Conserver les verifications sur pull request : `go test -race ./...`,
     `go vet ./...`, `npm run test`, `npm run build`, `npm run lint` et
     `docker build`.
   - Sur `main`, utiliser Workload Identity Federation/OIDC via
     `google-github-actions/auth`. Aucun fichier de cle de service ne doit etre
     genere ou televerse.
   - Deployer les regles et indexes Firestore depuis une etape controlee avant
     le service Cloud Run, ou documenter une etape d'administration separee si
     le workflow n'a pas les permissions.
   - Construire une image immuable et la pousser dans Artifact Registry avec
     un tag SHA de commit. Deployer exactement ce digest sur Cloud Run.
   - Injecter les variables publiques `VITE_*` au build et verifier qu'aucune
     variable `FIREBASE_ADMIN_*`, cle privee ou token n'apparait dans les logs.
   - Le workflow doit pouvoir etre relance sans recreer la base ni casser les
     regles existantes.

5. LOCAL ET EMULATEURS :
   - Documenter le lancement de l'emulateur Firestore et de l'emulateur Firebase
     Auth pour les tests et les parcours locaux.
   - Le mode local peut garder la session hotseat et les endpoints dev, mais il
     doit separer explicitement les credentials et URLs de production.
   - Les tests de regles, d'authentification et de transactions ne doivent pas
     consommer le quota d'un projet GCP reel.

6. OBSERVABILITE ET COUT :
   - Ajouter des alertes de budget, un dashboard ou des logs structures pour
     les erreurs Firestore, les transactions rejouees, les leases recuperees et
     les erreurs Auth, sans loguer les tokens, emails complets ou codes
     d'invitation.
   - Documenter les quotas a surveiller : lectures des listeners, lectures et
     ecritures Firestore, stockage, Cloud Run, Artifact Registry et logs.
   - Le free tier est un objectif, pas une garantie. La region, le stockage
     d'image, les logs, les lectures temps reel et une erreur de configuration
     peuvent engendrer des couts.

7. TESTS ET VALIDATION :
   - Test d'integration du conteneur : page HTML, `/healthz`, `/healthz/ready`,
     API REST et fallback SPA.
   - Verifier que le conteneur demarre sans volume et ne cree aucun fichier de
     partie local.
   - Smoke test public : lien HTTPS, lien email, creation de deux parties,
     join, soumission, resolution, listener temps reel et restauration apres
     redemarrage d'instance.
   - Verifier les regles Firestore depuis un compte membre, non-membre et non
     authentifie.
   - Verifier dans l'image et les workflows l'absence de service account JSON,
     de secrets ou d'identifiants GCP en dur.

CRITERES D'ACCEPTATION :
- `go test -race ./...`, `go vet ./...`, les tests frontend et `docker build`
  passent.
- L'image unique sert le front et l'API sans volume persistant.
- Le workflow s'authentifie par WIF, pousse une image immuable et deploie une
  revision Cloud Run reproductible.
- Deux comptes jouent une partie publique avec Firebase Auth et les listeners
  sans fuite de donnees.
- Un redemarrage Cloud Run ne perd ni parties, ni soumissions, ni rapports.
- Les regles, indexes, quotas, budgets et variables publiques sont documentes.
- Aucun fallback Compute Engine, GCS FUSE, Persistent Disk ou `DATA_DIR` n'est
  necessaire.

Note : documente dans la reponse finale le projet/region choisis, les roles du
service account, les variables publiques, les APIs activees, les regles,
indexes, budgets et resultats du smoke test. Ne commit pas sans instruction
explicite.
```
