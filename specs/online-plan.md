# Plan d'implémentation online

**Issue parente :** [#2 — Online : finaliser les fonctionnalités P3.x et la vue privée par joueur](https://github.com/fogfactory/crown-and-borough/issues/2)

Ce document décrit le découpage de l'issue #2 en sous-issues livrables et
testables. Il complète [`online.md`](online.md) et les prompts de
[`prompts/`](prompts/).

## Objectif MVP

Crown & Borough doit être jouable entre amis depuis un lien public, avec une
identité serveur fiable, des vues privées filtrées par le serveur et une
partie restaurable après redémarrage.

Le MVP online adopte les limites suivantes :

- une seule partie active par déploiement ;
- deux à huit joueurs online, tandis que le moteur conserve sa capacité de 2 à
  16 joueurs ;
- un identifiant de partie et des routes `/api/games/{id}` sont conservés pour
  permettre une évolution multi-parties ultérieure ;
- un lien d'invitation porte un code opaque de six caractères ;
- l'inscription ne demande pas de mot de passe ;
- les tokens Bearer sont conservés en mémoire et côté front dans `localStorage` ;
- une session est perdue au redémarrage, mais un joueur peut reprendre son slot
  avec son nom et le code d'invitation ;
- aucune deadline automatique n'est introduite ;
- une action explicite de résolution forcée empêche une partie entre amis de
  rester bloquée indéfiniment ;
- un joueur éliminé est automatiquement considéré comme ayant soumis ;
- les tailles d'armées, ressources, infrastructures et positions des nobles
  restent publiques ; le brouillard de guerre général reste hors périmètre ;
- les endpoints hotseat de développement ne sont jamais exposés en production.

## Décisions techniques

### Identité territoriale

Le trigramme territorial est l'identité unique du domaine et des contrats.
Dans `map.json` et `state.json`, le champ `id` contient le trigramme et aucun
champ `code` territorial dupliqué n'est exposé. Aucun `T<number>` ne doit être
généré, sérialisé ou affiché.

### Vues privées

La projection serveur distingue explicitement l'absence de chaîne d'une chaîne
masquée :

- `chain: null` signifie qu'aucune chaîne n'est active ;
- `chain: { "visibility": "hidden" }` signifie qu'une chaîne existe mais que
  son détail n'est pas révélé ;
- `chain: { "visibility": "known", ... }` contient le détail connu par le
  joueur.

Les combats portent également une visibilité `exact` ou `general`. Une vue
`general` ne contient pas les puissances, les identifiants d'armées ni les
autres détails permettant de reconstruire le combat.

La connaissance d'une chaîne ne doit pas être déduite uniquement de l'état
courant. Le serveur conserve des métadonnées privées par partie, joueur et
chaîne, avec les règles d'invalidation suivantes :

- une chaîne émise par le joueur est connue ;
- une chaîne émise par un noble otage détenu par le joueur reste connue selon le
  GDD ;
- la progression compatible conserve la connaissance ;
- le joueur propriétaire de l'armée connaît le remplacement ; un tiers conserve
  sa connaissance précédente jusqu'à une contradiction observable ;
- les métadonnées nécessaires à la projection des rapports historiques sont
  persistées avec la partie.

Le comportement des nobles prisonniers doit rester cohérent avec l'issue #29.

### API et développement local

Le serveur de production utilise exclusivement `net/http` et
`http.ServeMux`. Les anciens endpoints hotseat peuvent rester disponibles sous
un mode de développement explicite, mais ils ne doivent pas fournir de chemin
d'accès non authentifié au déploiement public.

Les routes de jeu couvrent au minimum :

- création, liste et détail de la partie ;
- carte, état et ravitaillement ;
- soumission des ordres ;
- résolution forcée ;
- rapports ;
- règles du jeu.

Avant authentification, `?player=` est un mécanisme de test local. Après
authentification, l'identité vient exclusivement du token et ce paramètre est
supprimé de l'API publique.

### Persistance et hébergement

`DATA_DIR` est la seule interface de stockage visible par l'application.

- Sur filesystem local ou Persistent Disk, le backend utilise un fichier
  temporaire, `fsync` et `rename` atomique.
- Sur Cloud Run avec volume GCS FUSE, un spike de durabilité est obligatoire.
  Le backend peut utiliser des snapshots complets versionnés et sélectionner
  au redémarrage la dernière génération JSON valide, sans dépendre de
  l'atomicité POSIX de `rename`.

Cloud Run + GCS est la première cible pour le HTTPS géré, le scale-to-zero et
le free-tier potentiel. Compute Engine e2-micro + disque persistant est la
cible de repli si le smoke test GCS ne permet pas de garantir la restauration.
Le code applicatif et l'image OCI restent identiques entre les deux cibles ;
seul le workflow et le montage de `DATA_DIR` changent.

Le free-tier est un objectif et non une garantie : les opérations GCS, le
stockage d'images, les logs et la région choisie peuvent générer des coûts.

## Découpage GitHub

### [Online Foundations](https://github.com/fogfactory/crown-and-borough/milestone/11)

#### O1 — [Contrats et décisions d'architecture](https://github.com/fogfactory/crown-and-borough/issues/44)

**Titre :** `meta(online): figer les contrats et l'architecture du MVP hébergé`

**Dépendances :** aucune.

**Livrables :** mise à jour des spécifications, exemples JSON dans
[`fixtures/`](fixtures/) et décisions sur la partie unique active, les routes,
la résolution forcée, les erreurs, la privacy metadata, le format de
persistance et les deux cibles de stockage.

**Tests automatiques :** round-trip des fixtures JSON et tests Go/Vitest de
contrat sur les champs territoriaux, les chaînes masquées et les combats
généraux.

**Validation manuelle :** relire les exemples JSON et vérifier qu'ils
permettent de distinguer une armée sans chaîne d'une chaîne masquée et un
combat exact d'un combat général.

#### O2 — [Identité territoriale du domaine](https://github.com/fogfactory/crown-and-borough/issues/45)

**Titre :** `refactor(online): utiliser le trigramme comme identité territoriale canonique`

**Dépendances :** O1.

**Livrables :** refactor de `models`, `engine`, `mapgen`, `orders`, `demo` et
des tests ; suppression du double concept `TerritoryID`/`Code` et de toute
génération de `T<number>`.

**Tests automatiques :** déterminisme et unicité des trigrammes, round-trip de
l'état, résolution complète, ravitaillement, hiver, dispersion, `go test
-race ./...` et `go vet ./...`.

**Validation manuelle :** inspecter `map.json`, `state.json`, les erreurs et
les rapports après une partie locale ; aucune référence `T<number>` ne doit
être visible.

#### O3 — [Contrats publics et fixtures](https://github.com/fogfactory/crown-and-borough/issues/46)

**Titre :** `feat(online): stabiliser les contrats map/state/report sur les trigrammes`

**Dépendances :** O2.

**Livrables :** migration des DTO Go, types TypeScript, clés React, tooltips,
rapports, règles, erreurs, fixtures et exemples vers les contrats canoniques.

**Tests automatiques :** tests API map/state/supply/rules, tests Vitest,
`npm run test`, `npm run build` et `npm run lint`.

**Validation manuelle :** sélectionner un territoire, saisir une chaîne avec
les trigrammes de la carte et vérifier que les mêmes références apparaissent
dans le rapport et les erreurs.

#### O4 — [Vues privées serveur](https://github.com/fogfactory/crown-and-borough/issues/42)

**Titre :** `feat(online): filtrer les chaînes et combats par joueur côté serveur`

**Dépendances :** O1, O3 ; coordination avec l'issue #29 pour le cas des
nobles prisonniers.

**Livrables :** projection serveur de l'état et des rapports, métadonnées de
connaissance par joueur, redaction exacte/générale des combats et absence de
fuite dans le JSON brut.

**Tests automatiques :** projections P1/P2/P3, chaîne connue/masquée/remplacée,
cas otage, combat exact pour un participant, combat général pour un spectateur,
golden files JSON et absence de mutation de l'état source.

**Validation manuelle :** comparer les réponses brutes de trois joueurs et
vérifier qu'un spectateur ne peut pas retrouver les forces d'un combat non
impliqué.

#### O5 — [API d'une partie active en mémoire](https://github.com/fogfactory/crown-and-borough/issues/39)

**Titre :** `feat(online): ajouter l'API d'une partie et la résolution par joueur`

**Dépendances :** O3, O4.

**Livrables :** store de partie, routes `/api/games/{id}`, soumissions
individuelles, résolution automatique ou forcée, rapports, saisons,
élimination, déterminisme et endpoints `supply`/`rules`. Une deuxième partie
active renvoie `409`.

**Tests automatiques :** CRUD, soumission en attente, remplacement,
validation atomique, résolution forcée, quatre saisons, élimination, gagnant,
concurrence `-race`, déterminisme et vues distinctes.

**Validation manuelle :** créer une partie avec `curl`, soumettre P1 puis P2,
vérifier `pending` puis `resolved`, puis vérifier qu'une seconde création
renvoie `409`.

### [Online Friends MVP](https://github.com/fogfactory/crown-and-borough/milestone/12)

#### O6 — [Authentification et invitations](https://github.com/fogfactory/crown-and-borough/issues/43)

**Titre :** `feat(online): ajouter les tokens, invitations et reprises de slot`

**Dépendances :** O5.

**Livrables :** register, `/me`, tokens Bearer, codes d'invitation, `inviteUrl`,
slots, membership, reprise par nom et code, maximum de cinq joueurs et
suppression de l'identité `player` fournie par le client.

**Tests automatiques :** validation des noms, tokens invalides, création,
join, partie pleine, reprise, `401`/`403`, usurpation, courses sur register et
join.

**Validation manuelle :** inscrire Alice et Bob dans deux navigateurs, créer
une partie, rejoindre avec le lien, vérifier les accès interdits et reprendre
un slot après redémarrage du serveur.

#### O7 — [Parcours front entre amis](https://github.com/fogfactory/crown-and-borough/issues/47)

**Titre :** `feat(online): ajouter le parcours front inscription-invitation-partie`

**Dépendances :** O6.

**Livrables :** écrans inscription/création/join, token localStorage, lien
copiable, slots, vue privée, polling suspendu dans un onglet caché, ordres,
rapports, hiver, victoire, erreurs et accessibilité. Le parcours est documenté
dans `TESTING.md`.

**Tests automatiques :** tests des écrans, du token, du lien, du polling, des
vues masquées et des erreurs HTTP, plus `npm run test`, `npm run build` et
`npm run lint`.

**Validation manuelle :** jouer une partie complète depuis deux navigateurs,
tester `F5`, un `401`, le rapport de combat, l'hiver et une largeur mobile.

### [Online Hosted](https://github.com/fogfactory/crown-and-borough/milestone/10)

#### O8 — [Persistance et spike GCS FUSE](https://github.com/fogfactory/crown-and-borough/issues/40)

**Titre :** `feat(online): sauvegarder et restaurer les parties`

**Dépendances :** O5, O6.

**Livrables :** `DATA_DIR`, format JSON versionné, soumissions en attente,
historique borné, rapports et privacy metadata, restauration, quarantine,
nettoyage des temporaires et deux stratégies filesystem/snapshot. Le spike
GCS FUSE doit être réalisé avant de certifier Cloud Run.

**Tests automatiques :** sauvegarde, restauration, rapport historique,
snapshot incomplet, fichier corrompu, version inconnue, concurrence et
validation des invariants.

**Validation manuelle :** arrêter le serveur après une soumission partielle,
redémarrer, poursuivre le tour, vérifier les rapports, puis tester un fichier
corrompu et un snapshot incomplet.

#### O9 — [Conteneur et CI](https://github.com/fogfactory/crown-and-borough/issues/41)

**Titre :** `build(online): servir le front embarqué et automatiser les vérifications`

**Dépendances :** O7, O8.

**Livrables :** build frontend dans Docker, front embarqué dans le binaire Go,
fallback SPA, `/healthz/ready`, cibles Makefile et CI Go/frontend/Docker.

**Tests automatiques :** test d'intégration du conteneur, santé, page HTML,
API, fallback SPA et vérification du workflow.

**Validation manuelle :** construire et lancer l'image avec un volume
`DATA_DIR`, ouvrir le front sans Vite, créer une partie, redémarrer et vérifier
la restauration.

#### O10 — [Déploiement GCP et migration](https://github.com/fogfactory/crown-and-borough/issues/48)

**Titre :** `ops(online): déployer le jeu sur GCP avec un workflow portable`

**Dépendances :** O8, O9.

**Livrables :** Workload Identity Federation, Artifact Registry, déploiement
Cloud Run avec une instance maximum et `min-instances=0`, volume GCS et
workflow Compute Engine de repli avec Persistent Disk. Les paramètres GCP
restent dans les variables du workflow.

**Tests automatiques :** build/push immuable, smoke HTTP, absence de secrets
ou d'identifiants GCP codés en dur.

**Validation manuelle :** ouvrir le lien HTTPS public, jouer un tour à deux,
redémarrer l'instance et vérifier la restauration. Si le test GCS échoue,
déployer la même image avec le workflow Compute Engine.

## Dépendances

```text
O1 -> O2 -> O3 -> O4 -> O5 -> O6 -> O7 -> O9 -> O10
                         \-> O8 -----------^
```

O8 peut avancer en parallèle d'O7 après O5 et O6, mais O9 attend les deux.

Les issues #26, #27 et #28 restent hors périmètre. L'issue #29 est liée à O4
pour garantir la sémantique des nobles prisonniers sans absorber tout son
périmètre.

## Fermeture de l'issue #2

- `go test -race ./...` passe ;
- `go vet ./...` passe ;
- `npm run test`, `npm run build` et `npm run lint` passent ;
- `docker build` passe ;
- deux navigateurs jouent une partie complète via invitation ;
- les vues serveur diffèrent selon le joueur ;
- aucun joueur ne peut lire ou soumettre pour un autre ;
- un redémarrage ne perd ni l'état, ni les soumissions, ni les rapports ;
- un lien public fonctionne sur Cloud Run ou sur le workflow Compute Engine de
  repli ;
- le free-tier est documenté comme objectif avec ses limites et non comme une
  garantie contractuelle.
