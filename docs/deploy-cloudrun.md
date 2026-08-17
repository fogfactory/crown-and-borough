# Cloud Run Deployment

This runbook covers the one-time Google Cloud and Firebase setup for the
production workflow in [`.github/workflows/deploy-cloudrun.yml`](../.github/workflows/deploy-cloudrun.yml).
The workflow uses GitHub Actions OIDC and Workload Identity Federation (WIF).
It never needs a service-account key file.

Cloud Run is public at the infrastructure layer so that the browser can load
the application. The Go API still requires a Firebase ID token for profiles,
games, invitations, and orders. Firestore is the only hosted game persistence;
the container has no volume or local game database.

## Deployment Model

The workflow has two promotion paths:

- A push to `main` builds and pushes an image tagged with the commit SHA. It
  does not change Cloud Run traffic.
- A `v*` tag or a confirmed `workflow_dispatch` builds the image, deploys the
  versioned Firestore rules and indexes, creates a Cloud Run candidate with no
  service traffic, runs an HTTPS smoke test against its `smoke` tag URL, and
  routes 100% of traffic to the exact image digest only after the smoke test
  passes.

The first Cloud Run deployment is the exception required by the Cloud Run CLI:
`--no-traffic` is not accepted while creating a new service. The first revision
is therefore created with normal traffic and smoke-tested at the public service
URL. Every later deployment uses the no-traffic `smoke` tag flow described above.

The initial service settings are `min-instances=0`, `max-instances=1`, one CPU,
`512Mi` memory, and concurrency `80`. These settings control the initial cost;
the Firestore transactions and conditional commits must remain correct if the
instance limit is increased later.

## Prerequisites

Install and authenticate the following tools locally:

- Google Cloud CLI with permission to manage project services, IAM, Artifact
  Registry, Cloud Run, Firestore, and Firebase configuration.
- Firebase CLI, authenticated with `firebase login` if the project is not
  already associated with Firebase.
- GitHub CLI with repository administration permission, or equivalent access to
  repository Actions variables.

Choose a GCP project, a Cloud Run region, and a Firestore location. The example
below uses `europe-west1`; replace it when the project requires another region.
Cloud Run and Firestore may use different supported locations, but the selected
Firestore location cannot be changed after the database is created.

```bash
export PROJECT_ID="replace-with-gcp-project-id"
export REGION="europe-west1"
export FIRESTORE_LOCATION="europe-west1"
export AR_REPOSITORY="crown-and-borough"
export CLOUD_RUN_SERVICE="crown-and-borough"
export RUNTIME_SA_NAME="crown-and-borough-runtime"
export DEPLOYER_SA_NAME="crown-and-borough-deployer"
export GITHUB_REPOSITORY="fogfactory/crown-and-borough"

gcloud config set project "$PROJECT_ID"
```

Do not put these values in Go code or in the Dockerfile. They are supplied to
the workflow through GitHub variables.

## Project And APIs

If the Firebase project does not exist yet, create or select the GCP project
first and associate it with Firebase in the Firebase console. For an existing
GCP project, this command can be used when the Firebase project association is
not present:

```bash
firebase projects:addfirebase "$PROJECT_ID"
```

The command may report that the project is already a Firebase project. That is
safe; do not create a second project for the same deployment.

Enable the APIs used by the workflow, Cloud Run, Firebase Admin SDK, and
Firestore:

```bash
gcloud services enable \
  artifactregistry.googleapis.com \
  cloudresourcemanager.googleapis.com \
  firestore.googleapis.com \
  firebase.googleapis.com \
  iam.googleapis.com \
  iamcredentials.googleapis.com \
  identitytoolkit.googleapis.com \
  run.googleapis.com \
  serviceusage.googleapis.com \
  --project "$PROJECT_ID"
```

Confirm the result before continuing:

```bash
gcloud services list --enabled --project "$PROJECT_ID" \
  --filter='name:(artifactregistry.googleapis.com firestore.googleapis.com run.googleapis.com identitytoolkit.googleapis.com iam.googleapis.com serviceusage.googleapis.com)'
```

## Firestore

Check the databases before creating anything. The MVP expects the default
Firestore Native database and must not create a second database by accident:

```bash
gcloud firestore databases list --project "$PROJECT_ID"
```

If `(default)` is not listed, create it once:

```bash
gcloud firestore databases create \
  --database="(default)" \
  --location="$FIRESTORE_LOCATION" \
  --type=firestore-native \
  --project="$PROJECT_ID"
```

The repository already versions [`firestore.rules`](../firestore.rules),
[`firestore.indexes.json`](../firestore.indexes.json), and their Firebase CLI
mapping in [`firebase.json`](../firebase.json). The deployment workflow applies
the rules and indexes before it creates the Cloud Run candidate.

## Artifact Registry

Create one Docker repository in the selected region. The `describe` guard makes
this command safe to rerun:

```bash
gcloud artifacts repositories describe "$AR_REPOSITORY" \
  --location="$REGION" \
  --project="$PROJECT_ID" >/dev/null 2>&1 || \
gcloud artifacts repositories create "$AR_REPOSITORY" \
  --repository-format=docker \
  --location="$REGION" \
  --description="Crown and Borough production images" \
  --project="$PROJECT_ID"
```

The workflow deploys the pushed image by digest, not by a mutable tag. The
commit SHA tag is used for discovery and auditability only. If immutable tags
are enabled for the repository, do not rerun a release that would try to push
an already-existing SHA tag; promote the existing digest or use a new commit.

## Service Accounts

Use separate service accounts for GitHub Actions and the running Cloud Run
container. The deployer can publish and deploy, while the runtime account can
read and write the application data.

Create the accounts if they do not already exist:

```bash
gcloud iam service-accounts create "$RUNTIME_SA_NAME" \
  --display-name="Crown and Borough Cloud Run runtime" \
  --project="$PROJECT_ID"

gcloud iam service-accounts create "$DEPLOYER_SA_NAME" \
  --display-name="Crown and Borough GitHub deployer" \
  --project="$PROJECT_ID"

export RUNTIME_SA="$RUNTIME_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com"
export DEPLOYER_SA="$DEPLOYER_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com"
```

Grant the runtime account only the application roles. `roles/datastore.user`
is the Firestore Native client role; `roles/firebaseauth.admin` is needed by
the Admin SDK for Firebase ID-token verification, including the revoked-token
check used by the server. `roles/serviceusage.serviceUsageConsumer` allows the
client libraries to consume the enabled APIs:

```bash
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$RUNTIME_SA" \
  --role="roles/datastore.user"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$RUNTIME_SA" \
  --role="roles/firebaseauth.admin"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$RUNTIME_SA" \
  --role="roles/serviceusage.serviceUsageConsumer"
```

Grant the deployer the roles needed by the workflow. It must be able to deploy
Cloud Run, push Artifact Registry images, deploy Firebase rules and indexes,
and impersonate the runtime account:

```bash
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$DEPLOYER_SA" \
  --role="roles/run.admin"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$DEPLOYER_SA" \
  --role="roles/artifactregistry.writer"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$DEPLOYER_SA" \
  --role="roles/firebase.admin"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$DEPLOYER_SA" \
  --role="roles/serviceusage.serviceUsageConsumer"
gcloud iam service-accounts add-iam-policy-binding "$RUNTIME_SA" \
  --member="serviceAccount:$DEPLOYER_SA" \
  --role="roles/iam.serviceAccountUser" \
  --project="$PROJECT_ID"
```

If the organization does not allow the broad `roles/firebase.admin` role, use
the organization's approved equivalent for Firebase Rules and Firestore index
deployment and verify it with a dry run before adding the workflow variables.
Do not grant the runtime account deployer permissions.

## Workload Identity Federation

Create a global WIF pool and an OIDC provider restricted to this repository and
to the `main` branch or `v*` tags. The condition prevents an unrelated GitHub
repository from impersonating the deployer account:

```bash
export WIF_POOL_ID="github-actions"
export WIF_PROVIDER_ID="github"
export PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"

gcloud iam workload-identity-pools create "$WIF_POOL_ID" \
  --location=global \
  --display-name="GitHub Actions" \
  --project="$PROJECT_ID"

gcloud iam workload-identity-pools providers create-oidc "$WIF_PROVIDER_ID" \
  --location=global \
  --workload-identity-pool="$WIF_POOL_ID" \
  --display-name="GitHub Actions OIDC" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref" \
  --attribute-condition="assertion.repository == '$GITHUB_REPOSITORY' && (assertion.ref == 'refs/heads/main' || assertion.ref.startsWith('refs/tags/v'))" \
  --project="$PROJECT_ID"

export WIF_PROVIDER="projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$WIF_POOL_ID/providers/$WIF_PROVIDER_ID"
export WIF_MEMBER="principalSet://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$WIF_POOL_ID/attribute.repository/$GITHUB_REPOSITORY"

gcloud iam service-accounts add-iam-policy-binding "$DEPLOYER_SA" \
  --member="$WIF_MEMBER" \
  --role="roles/iam.workloadIdentityUser" \
  --project="$PROJECT_ID"
```

The workflow's `permissions` block grants `id-token: write` and does not create
or upload a JSON key. The WIF provider resource name and deployer service
account email are identifiers, not secrets.

If a pool or provider already exists, inspect it instead of creating a second
one:

```bash
gcloud iam workload-identity-pools describe "$WIF_POOL_ID" \
  --location=global --project="$PROJECT_ID"
gcloud iam workload-identity-pools providers describe "$WIF_PROVIDER_ID" \
  --workload-identity-pool="$WIF_POOL_ID" \
  --location=global --project="$PROJECT_ID"
```

## Firebase Authentication

In the Firebase console for this project:

1. Create a Web app if one does not exist.
2. Copy the Web configuration values `apiKey`, `authDomain`, `projectId`, and
   `appId`. These values are public and are injected into the Vite build.
3. Open Authentication, enable the Email/Password provider, and enable the
   Email link (passwordless sign-in) option.
4. Add `localhost` and the deployed Cloud Run hostname to Authorized domains.
   Add a custom application domain too if one will be used.

The first Cloud Run deployment can derive its `run.app` URL automatically. Get
the resulting URL with:

```bash
export PUBLIC_APP_URL="$(gcloud run services describe "$CLOUD_RUN_SERVICE" \
  --region="$REGION" --project="$PROJECT_ID" --format='value(status.url)')"
printf '%s\n' "$PUBLIC_APP_URL"
```

Add the hostname without the scheme to Firebase Authorized domains. For a
custom domain, set `PUBLIC_APP_URL` to the complete HTTPS origin in GitHub so
invitation URLs use that domain. With no variable, the workflow uses the
Cloud Run service URL discovered from GCP.

## GitHub Actions Variables

Create repository **variables**, not secrets, for the values below. The
workflow has no service-account key secret and must not be given one.

| Variable                       | Value                                                     |
| ------------------------------ | --------------------------------------------------------- |
| `GCP_PROJECT_ID`               | `$PROJECT_ID`                                             |
| `GCP_REGION`                   | `$REGION`                                                 |
| `GCP_ARTIFACT_REGISTRY`        | `$REGION-docker.pkg.dev`                                  |
| `GCP_AR_REPOSITORY`            | `$AR_REPOSITORY`                                          |
| `GCP_CLOUD_RUN_SERVICE`        | `$CLOUD_RUN_SERVICE`                                      |
| `GCP_DEPLOYER_SERVICE_ACCOUNT` | `$DEPLOYER_SA`                                            |
| `GCP_RUNTIME_SERVICE_ACCOUNT`  | `$RUNTIME_SA`                                             |
| `GCP_WIF_PROVIDER`             | `$WIF_PROVIDER`                                           |
| `VITE_FIREBASE_API_KEY`        | Firebase Web `apiKey`                                     |
| `VITE_FIREBASE_AUTH_DOMAIN`    | Firebase Web `authDomain`                                 |
| `VITE_FIREBASE_PROJECT_ID`     | Firebase Web `projectId`                                  |
| `VITE_FIREBASE_APP_ID`         | Firebase Web `appId`                                      |
| `PUBLIC_APP_URL`               | Optional complete HTTPS origin, usually the Cloud Run URL |

With GitHub CLI, the variables can be set without writing any credential file:

```bash
gh variable set GCP_PROJECT_ID --repo "$GITHUB_REPOSITORY" --body "$PROJECT_ID"
gh variable set GCP_REGION --repo "$GITHUB_REPOSITORY" --body "$REGION"
gh variable set GCP_ARTIFACT_REGISTRY --repo "$GITHUB_REPOSITORY" --body "$REGION-docker.pkg.dev"
gh variable set GCP_AR_REPOSITORY --repo "$GITHUB_REPOSITORY" --body "$AR_REPOSITORY"
gh variable set GCP_CLOUD_RUN_SERVICE --repo "$GITHUB_REPOSITORY" --body "$CLOUD_RUN_SERVICE"
gh variable set GCP_DEPLOYER_SERVICE_ACCOUNT --repo "$GITHUB_REPOSITORY" --body "$DEPLOYER_SA"
gh variable set GCP_RUNTIME_SERVICE_ACCOUNT --repo "$GITHUB_REPOSITORY" --body "$RUNTIME_SA"
gh variable set GCP_WIF_PROVIDER --repo "$GITHUB_REPOSITORY" --body "$WIF_PROVIDER"

# Set these four from the Firebase Web app configuration.
gh variable set VITE_FIREBASE_API_KEY --repo "$GITHUB_REPOSITORY" --body "replace-me"
gh variable set VITE_FIREBASE_AUTH_DOMAIN --repo "$GITHUB_REPOSITORY" --body "replace-me"
gh variable set VITE_FIREBASE_PROJECT_ID --repo "$GITHUB_REPOSITORY" --body "$PROJECT_ID"
gh variable set VITE_FIREBASE_APP_ID --repo "$GITHUB_REPOSITORY" --body "replace-me"

# Optional when invitation links should use a custom origin.
gh variable set PUBLIC_APP_URL --repo "$GITHUB_REPOSITORY" --body "$PUBLIC_APP_URL"
```

Replace every `replace-me` value before starting a deployment. Check the final
names, without printing their values, with:

```bash
gh variable list --repo "$GITHUB_REPOSITORY"
```

## First Promotion

After the workflow and variables are available on `main`, create a release tag:

```bash
git tag -a v0.1.0 -m "Deploy v0.1.0"
git push origin v0.1.0
gh run list --repo "$GITHUB_REPOSITORY" --workflow deploy-cloudrun.yml
gh run watch --repo "$GITHUB_REPOSITORY"
```

The workflow builds and pushes the SHA-tagged image, deploys rules and indexes,
smoke-tests the candidate URL, and promotes the exact digest. On the first
deployment, if `PUBLIC_APP_URL` is empty and the service has no URL yet, the
workflow briefly uses an internal placeholder to create the service, reads the
real `run.app` URL, and creates a second first-service revision with the real
origin before running the smoke test. That first-service revision necessarily
serves normal traffic; subsequent promotions use the isolated `smoke` tag.

For an explicit manual promotion instead of a tag:

```bash
gh workflow run deploy-cloudrun.yml \
  --repo "$GITHUB_REPOSITORY" \
  --ref main \
  -f confirm_deploy=true
```

Do not treat a successful image build on `main` as a production deployment.
Only a successful candidate smoke test followed by the promotion job changes
service traffic.

## Manual Acceptance Checklist

Complete this checklist against the public HTTPS URL after the first
promotion. Use two separate browser profiles or two browsers so Firebase
sessions cannot be confused.

1. Open the Cloud Run URL and confirm the embedded application loads without
   Vite or a second frontend service.
2. Sign in as Alice and Bob with separate email links, opening each link in the
   browser that requested it.
3. Complete both profiles and confirm that an unauthenticated request receives
   `401` while a signed-in request reaches `/api/auth/me`.
4. Create two games, join one through its invitation URL, and confirm each
   account sees only its own memberships and permitted projections.
5. Submit a turn from both accounts and verify public status and private state
   changes arrive through the Firestore `onSnapshot` listeners.
6. Check the report, winter flow, forced resolution, and a mobile-width view.
7. Reload both browsers and confirm Firebase sessions and memberships persist.
8. Allow the Cloud Run instance to scale to zero, reload the game, and verify
   that Firestore restores the games, pending submissions, reports, and private
   projections. A later candidate deployment can also be used to exercise a
   fresh instance.
9. Confirm that two games can progress in parallel without a projection or
   invitation from one game appearing in the other.
10. Inspect Cloud Run and Firebase logs for failures without recording ID
    tokens, complete email addresses, invitation codes, or private game data.

The emulator tests remain the safe place to validate rules, transactions, Auth
identities, leases, and concurrent mutations before changing a real project:

```bash
FIRESTORE_EMULATOR_HOST=127.0.0.1:8081 make test-firestore
```

## Smoke And Security Contract

The post-deployment smoke job verifies all of the following on the candidate
tag URL:

- `/healthz/ready` reports `{"status":"ready"}`;
- `/` and a client-side route return the embedded SPA root;
- `/api/rules` is reachable;
- `/api/games` and `/api/auth/me` return `401` without a Firebase token.

The local container smoke test still checks `/healthz`. On the public Cloud Run
hostname, the exact `/healthz` path is intercepted by the Cloud Run edge and
returns its own 404 before reaching the container, so the hosted smoke test
uses `/healthz/ready`, which verifies both the process and Firebase/Firestore
readiness.

The image verification step pulls the pushed digest and rejects credential-like
files and Admin/private-key environment variables. The final scratch image is
expected to contain only the Go binary, public game assets, and CA certificates.
Firebase Web configuration is public build configuration; Admin credentials are
provided to the Go SDK through Cloud Run Application Default Credentials.

## Costs, Quotas, And Logs

The free tier is a cost objective, not a guarantee. Billing can be affected by
region, Firestore listener reads, transaction retries, projection writes,
Firestore storage, Cloud Run requests and CPU, Artifact Registry storage, and
Cloud Logging retention.

Create a budget in **Billing > Budgets & alerts** for the selected billing
account and project. A useful first setup has alerts at 50%, 80%, and 100% of a
small monthly budget. Review these metrics at least weekly:

- Firestore document reads, writes, deletes, storage, and listener activity;
- Firebase Authentication email-link usage and sign-in quotas;
- Cloud Run request count, instance count, CPU, memory, and egress;
- Artifact Registry storage and image retention;
- Cloud Logging ingestion and retention.

Use structured error searches and short retention for development logs. Never
log Authorization headers, Firebase ID tokens, complete invitation codes,
complete email addresses, or unfiltered game state. Keep `max-instances=1` as a
cost limit only; it is not a consistency mechanism.

## Rollback

List recent revisions and identify the last known-good revision:

```bash
gcloud run revisions list \
  --service="$CLOUD_RUN_SERVICE" \
  --region="$REGION" \
  --project="$PROJECT_ID" \
  --sort-by='~metadata.creationTimestamp' \
  --limit=5
```

Route all traffic to the known-good revision after confirming it is healthy:

```bash
export PREVIOUS_REVISION="replace-with-known-good-revision"
gcloud run services update-traffic "$CLOUD_RUN_SERVICE" \
  --to-revisions="$PREVIOUS_REVISION=100" \
  --region="$REGION" \
  --project="$PROJECT_ID"
```

Rules and indexes are versioned in Git. To roll those back, restore the desired
repository revision and run a new controlled promotion; do not edit the live
rules only in the Firebase console.
