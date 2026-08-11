# Prompt : déploiement et CI automatique

```
Tu travailles sur "Crown & Borough", un jeu de stratégie par tours.
Tout est en place et testé : moteur v1, API REST
(P3.1), auth + codes d'invitation (P3.2), persistance JSON (P3.3, DATA_DIR),
front polié (P3.4). Le Dockerfile multi-stage existe depuis P0.1 (étape
front prévue, commentaire en place), la CI GitHub Actions existe depuis P0.1
(go vet, test, build, docker build).
Références : specs/roadmap.md (P3.5) et specs/architecture.md (stack).

PÉRIMÈTRE : empaqueter l'application en UN conteneur (front + API), la
déployer d'abord sur Cloud Run, automatiser build+push+deploy dans la CI,
persister sur un bucket GCS si le spike de durabilité le valide, et fournir un
workflow Compute Engine de repli avec Persistent Disk. Le but : un LIEN PUBLIC
jouable entre amis, avec une seule partie active au MVP.

RÈGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages, enums). Seuls les labels UI restent en français.

1. UN SEUL CONTENEUR (front embarqué) :
   - Le serveur Go sert l'API (/api/*) ET les fichiers statiques du front :
     ajoute go:embed web/dist sur le serveur (dist absent → le serveur
     démarre quand même, 404 sur le front — pratique pour le dev)
   - SPA : toute route non-/api/* non-statique renvoie index.html (fallback
     pour le routing React)
   - Dockerfile multi-stage : build front (node → web/dist) PUIS build Go
     (embed dist) → image finale alpine/scratch ; front et API en same-origin
     → AUCUN CORS à configurer
   - Healthcheck : GET /healthz (existant) + GET /healthz/ready (vérifie
     que DATA_DIR est écrivable — le volume est monté et prêt)

2. PERSISTANCE SUR CLOUD RUN :
   - Le stockage de P3.3 (DATA_DIR) pointe sur /data
   - Cloud Run : montage de volume Cloud Storage (gcsfuse) d'un bucket
     dédié (ex. crown-and-borough-data-<projet>) sur /data, accessible en
     écriture ; documente le mode (gcsfuse, cache: stat-cache-max-ttl court
     — les fichiers changent au fil des tours)
   - DATA_DIR=/data via variable d'environnement du service ; P3.3
     fonctionne avec le backend choisi par le workflow
   - Le smoke test doit provoquer un redémarrage et vérifier que la dernière
     génération JSON complète est restaurée. Cloud Run n'est pas certifié si
     cette vérification échoue.

3. INFRASTRUCTURE (gcloud, projet GCP — nom à mettre en secret/variable) :
   - Artifact Registry : repository docker nommé crown-and-borough, région
     (ex. europe-west1)
   - Cloud Run : service crown-and-borough (region, cpu 1, mem 512 Mi,
     min-instances 0, max-instances 1, une seule révision active sans split
     de trafic) ; l'accès public Cloud Run est autorisé car
     l'authentification et les invitations sont gérées par l'application
   - Tout se fait via la CLI gcloud (pas de Terraform au MVP — choix
     documenté), les commandes sont reproductibles (fichier deploy.md ou
     script deploy.sh versionné)
   - IDs de projet et région en variables d'environnement GitHub
     (secrets/vars) — jamais en dur dans le repo

4. CI/CD (GitHub Actions) :
   - Workflow sur push/PR vers main : make test (avec -race) + make vet →
     build front → docker build → (push uniquement sur main) push vers
     Artifact Registry → gcloud run deploy
   - Authentification GCP sans fichier clé : workload identity federation
     (OpenID Connect, google-github-actions/auth) — le workflow de la CI
     existante (P0.1) est étendu, pas remplacé
   - Déploiement uniquement sur push main (les PR font juste les tests et
     le build docker)
   - Tag d'image : sha du commit (immutable, rollback facile)

5. DOCUMENTATION (README section "Déploiement") :
   - Commandes de création du projet GCP (une fois) : gcloud projects
     create, artefact registry repo, bucket, enable services (run,
     artifactregistry, storage)
   - Commandes de mise en place de la fédération d'identité (une fois)
   - Le workflow déploie ensuite automatiquement

6. TESTS / VALIDATION (local, sans GCP pour l'essentiel) :
   - docker build réussit et l'image sert le front + l'API (curl /healthz,
     /api/games, et la page HTML)
   - go test ./... + -race passe dans la CI (déjà en place, inchangé)
   - Si tu as accès à un projet GCP : déploie et vérifie que le lien
     public répond (healthz + inscription + création de partie). Sinon :
     documente précisément les étapes pour le faire (le propriétaire du
     projet le fera)
   - Si le test GCS échoue, déploie la même image sur une VM Compute Engine
     e2-micro avec un Persistent Disk monté sur DATA_DIR et rejoue le test.

Critères d'acceptation :
- make test passe (avec -race) ; make vet passe ; docker build OK
- Le conteneur sert front + API en same-origin (aucun CORS)
- Le workflow CI étendu est vert (au moins la partie tests/build) ;
  le déploiement automatique est prêt (workflow + documentation)
- La persistance survit aux redémarrages Cloud Run grâce au backend GCS validé,
  ou au workflow Compute Engine si le smoke test GCS échoue

Note : documente dans la réponse finale les choix tranchés (région,
allow-unauthenticated, Terraform différé, une partie active, max-instances 1,
gcsfuse et workflow Compute Engine de repli). Ne commite pas sans instruction
explicite.
```
