# Plan d'implémentation online

**Issue parente :** [#2 — Online : fonctionnalités online, Firebase et Firestore](https://github.com/fogfactory/crown-and-borough/issues/2)

Ce document décrit le découpage de l'issue #2 en sous-issues livrables et
testables. Il complète [`online.md`](online.md) et les prompts de
[`prompts/`](prompts/).

## Objectif MVP

Crown & Borough doit être jouable entre amis depuis un lien public, avec une
identité Firebase fiable, des vues privées filtrées par le serveur et des
parties restaurables après redémarrage.

Le MVP online adopte les limites suivantes :

- plusieurs parties actives par déploiement ;
- deux à huit joueurs online, tandis que le moteur conserve sa capacité de 2 à
  16 joueurs ;
- un identifiant de partie et des routes `/api/games/{id}` sont conservés pour
  gérer plusieurs parties dès le MVP ;
- un lien d'invitation porte un code opaque de six caractères ;
- l'authentification utilise Firebase Authentication et un lien de connexion
  par email, sans mot de passe géré par l'application ;
- le serveur valide les ID tokens Firebase dans le header Bearer ; les tokens
  ne sont ni générés ni persistés par le backend ;
- la session Firebase est conservée côté navigateur par le SDK Firebase et le
  profil et les appartenances sont persistés dans Firestore ; un redémarrage du
  serveur ne déconnecte donc pas les joueurs ;
- le choix de session persistante ne signifie pas `sessions/{token}` : les
  secrets d'authentification restent gérés par Firebase, tandis que Firestore
  persiste le profil et les memberships ;
- aucune deadline automatique n'est introduite ;
- une action explicite de résolution forcée empêche une partie entre amis de
  rester bloquée indéfiniment ;
- un joueur éliminé est automatiquement considéré comme ayant soumis ;
- les tailles d'armées, ressources, infrastructures et positions des nobles
  restent publiques ; le brouillard de guerre général reste hors périmètre ;
- les mises à jour du front utilisent `onSnapshot` sur les projections
  Firestore publiques et privées autorisées par les règles ; aucun listener ne
  lit l'état canonique, les ordres bruts ou les rapports non filtrés ;
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
authentification, l'identité vient exclusivement du JWT Firebase et ce
paramètre est supprimé de l'API publique.

### Persistance et hébergement

Firestore Native mode est la seule persistance distante du MVP hébergé. Le
serveur utilise le SDK Cloud Firestore avec les credentials ADC du service
account Cloud Run ; le moteur reste pur et ne dépend pas de Firestore.

Le modèle distingue strictement les données backend et les projections lisibles
par le front :

- `players/{uid}` contient le profil et les métadonnées nécessaires à la liste
  des parties ;
- `games/{gameId}` contient les métadonnées publiques d'une partie, sans état
  canonique ni code d'invitation ;
- `games/{gameId}/views/{uid}` contient la projection privée de ce joueur ;
- `games/{gameId}/reports/{uid}/turns/{turn}` contient ses rapports filtrés ;
- les documents d'état moteur, de soumissions brutes, de rapports complets,
  d'invitations hachées et de privacy metadata sont réservés au backend.

Les règles Firestore autorisent à un joueur membre la lecture de la partie
publique et de ses propres projections uniquement. Aucun client ne peut écrire
directement une partie, un ordre, une projection ou une invitation.

Chaque mutation vérifie le tour et la révision dans une transaction. Une
résolution concurrente est d'abord revendiquée atomiquement, puis le moteur
pur calcule le résultat et un commit conditionnel publie l'état, le rapport et
les projections. La revendication possède un lease interne récupérable après
crash ; il ne s'agit pas d'une deadline automatique de jeu.

Cloud Run est la cible unique du MVP : HTTPS géré, scale-to-zero, image OCI
unique et `max-instances=1` comme choix initial de coût, sans en faire une
garantie de cohérence. Le volume GCS FUSE, le backend snapshot, Persistent Disk
et le workflow Compute Engine de repli sont retirés du périmètre.

Le free-tier Firestore, Firebase Authentication et Cloud Run est un objectif et
non une garantie : les lectures des listeners, les écritures de projections,
le stockage, les logs, la région et la facturation peuvent générer des coûts.
Des alertes de budget et des quotas documentés sont requis.

## Découpage GitHub

### [Online Foundations](https://github.com/fogfactory/crown-and-borough/milestone/11)

#### O1 — [Contrats et décisions d'architecture](https://github.com/fogfactory/crown-and-borough/issues/44)

**Titre :** `meta(online): figer les contrats et l'architecture du MVP hébergé`

**Dépendances :** aucune.

**Livrables :** mise à jour des spécifications, exemples JSON dans
[`fixtures/`](fixtures/) et décisions sur les parties multiples, les routes,
la résolution forcée, les erreurs, la privacy metadata, le schéma Firestore,
Firebase Auth et les projections temps réel.

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

#### O5 — [API de plusieurs parties en mémoire](https://github.com/fogfactory/crown-and-borough/issues/39)

**Titre :** `feat(online): ajouter l'API de plusieurs parties et la résolution par joueur`

**Dépendances :** O3, O4.

**Livrables :** store de plusieurs parties, routes `/api/games/{id}`, liste et
détail filtrés par membership, soumissions individuelles, résolution
automatique ou forcée, rapports, saisons, élimination, déterminisme et
endpoints `supply`/`rules`. La limite d'une seule partie active est supprimée ;
les identifiants de partie restent stables et uniques.

**Tests automatiques :** CRUD, soumission en attente, remplacement,
validation atomique, résolution forcée, quatre saisons, élimination, gagnant,
concurrence `-race`, déterminisme et vues distinctes.

**Validation manuelle :** créer deux parties avec `curl`, vérifier que chaque
joueur ne voit que ses parties, soumettre P1 puis P2 dans l'une d'elles et
vérifier `pending` puis `resolved` sans modifier l'autre.

### [Online Friends MVP](https://github.com/fogfactory/crown-and-borough/milestone/12)

#### O6 — [Firebase Auth et invitations](https://github.com/fogfactory/crown-and-borough/issues/43)

**Titre :** `feat(online): intégrer Firebase Auth et les invitations`

**Dépendances :** O5.

**Livrables :** connexion Firebase par lien email, validation des ID tokens par
le backend avec Firebase Admin SDK, `/me`, profil persistant `players/{uid}`,
codes d'invitation hachés, `inviteUrl`, slots et membership par `uid`. Le nom
de joueur est un profil validé séparément ; l'identité `player` fournie par le
client, la reprise par nom et les tokens maison sont supprimés.

**Tests automatiques :** vérification de JWT valide, expiré, mal signé ou d'un
autre projet, validation du profil, création, join, partie pleine,
`401`/`403`, usurpation, contrôle d'appartenance et courses sur join.

**Validation manuelle :** connecter Alice et Bob par lien email dans deux
navigateurs, créer une partie, rejoindre avec le lien, vérifier les accès
interdits, fermer et rouvrir le serveur puis vérifier que les identités et
memberships sont toujours disponibles.

#### O7 — [Parcours front entre amis](https://github.com/fogfactory/crown-and-borough/issues/47)

**Titre :** `feat(online): ajouter le parcours front Firebase et temps réel`

**Dépendances :** O6.

**Livrables :** configuration Firebase Web, connexion par lien email, profil,
liste multi-parties, création/join, lien copiable, slots, abonnements
`onSnapshot` aux projections publiques et privées, vue privée, ordres,
rapports, hiver, victoire, erreurs et accessibilité. Les commandes restent
des appels REST avec un ID token rafraîchi par le SDK. Le parcours est
documenté dans `TESTING.md`.

**Tests automatiques :** tests des écrans, de la connexion email simulée, du
lien, des listeners et de leur désabonnement, des vues masquées et des erreurs
HTTP, plus `npm run test`, `npm run build` et `npm run lint`.

**Validation manuelle :** jouer deux parties depuis plusieurs navigateurs,
tester `F5`, une reconnexion Firebase, un `401`, la mise à jour instantanée
d'une projection, le rapport de combat, l'hiver et une largeur mobile.

### [Online Hosted](https://github.com/fogfactory/crown-and-borough/milestone/10)

#### O8 — [Persistance Firestore](https://github.com/fogfactory/crown-and-borough/issues/40)

**Titre :** `feat(online): persister les parties dans Firestore`

**Dépendances :** O5, O6.

**Livrables :** client Firestore, schéma des collections, documents canoniques
réservés au backend, projections publiques et privées, soumissions en attente,
historique borné, rapports et privacy metadata, transactions de mutation,
revendication de résolution, commit conditionnel, reprise d'une revendication
abandonnée et validation des règles de sécurité. La restauration est vérifiée
par relecture Firestore après redémarrage ; aucun fichier `DATA_DIR` ou volume
GCS FUSE n'est utilisé.

**Tests automatiques :** sauvegarde, restauration, rapport historique,
transactions concurrentes, double résolution, remplacement d'une soumission,
révision obsolète, lease de résolution expiré et validation des invariants.
Les tests utilisent l'émulateur Firestore ; les règles sont testées avec des
identités Firebase simulées.

**Validation manuelle :** arrêter le serveur après une soumission partielle,
redémarrer, poursuivre le tour, vérifier les rapports et les projections dans
deux navigateurs, puis provoquer deux soumissions et deux résolutions
concurrentes.

#### O9 — [Conteneur et CI](https://github.com/fogfactory/crown-and-borough/issues/41)

**Titre :** `build(online): servir le front embarqué et vérifier Firebase`

**Dépendances :** O7, O8.

**Livrables :** build frontend dans Docker, front embarqué dans le binaire Go,
fallback SPA, `/healthz/ready`, cibles Makefile et CI Go/frontend/Docker.

**Tests automatiques :** test d'intégration du conteneur, santé, page HTML,
API, fallback SPA et vérification du workflow.

**Validation manuelle :** construire et lancer l'image sans volume persistant,
ouvrir le front sans Vite, créer une partie, arrêter le conteneur, le relancer
et vérifier la restauration depuis Firestore.

#### O10 — [Déploiement GCP et migration](https://github.com/fogfactory/crown-and-borough/issues/48)

**Titre :** `ops(online): déployer le jeu sur Cloud Run avec Firebase`

**Dépendances :** O8, O9.

**Livrables :** Workload Identity Federation, Artifact Registry, activation et
configuration de Firebase Authentication et Firestore, règles et indexes
Firestore versionnés, déploiement Cloud Run avec `min-instances=0` et une
limite d'instances initiale, sans volume persistant. Les paramètres GCP et la
configuration publique Firebase restent dans les variables du workflow ou les
variables frontend ; aucun secret de service account n'est commité.

**Tests automatiques :** build/push immuable, smoke HTTP, absence de secrets
ou d'identifiants GCP codés en dur.

**Validation manuelle :** ouvrir le lien HTTPS public, connecter deux comptes,
créer deux parties, jouer un tour à deux, redémarrer l'instance et vérifier la
restauration Firestore ainsi que les mises à jour temps réel.

## Dépendances

```text
O1 -> O2 -> O3 -> O4 -> O5 -> O6 -> O7 -> O9 -> O10
                         \-> O8 -----------^
```

O8 peut avancer en parallèle d'O7 après O5 et O6, mais O9 attend les deux.
O5 ne bloque plus la création d'une deuxième partie ; O8 fournit ensuite la
persistance Firestore du store multi-parties. O10 dépend aussi des règles,
indexes et variables Firebase définis par O8/O9.

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
- un lien public fonctionne sur Cloud Run avec Firestore et Firebase Auth ;
- deux parties peuvent être jouées simultanément sans fuite de projection ;
- les listeners temps réel se désabonnent correctement et ne lisent jamais les
  documents canoniques ;
- le free-tier est documenté comme objectif avec ses limites et non comme une
  garantie contractuelle.
