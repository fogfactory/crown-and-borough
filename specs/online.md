# Online

**Milestone lié :** [Online Hosted](https://github.com/fogfactory/crown-and-borough/milestone/10)

**Issue principale :** [#2 — fonctionnalités online, Firebase et Firestore](https://github.com/fogfactory/crown-and-borough/issues/2)

**Plan d'implémentation :** [`online-plan.md`](online-plan.md)

**Contrats O1 :** [`fixtures/`](fixtures/) contient les documents JSON complets
utilisés pour figer les contrats de carte, d'état et de rapport. Le trigramme
est l'unique identité territoriale publique (`territories[].id`) ; `code` n'est
pas un champ territorial requis.

Le mode online actuel fournit une session en mémoire, la soumission par joueur,
une résolution synchrone et la projection privée hotseat via
`GET /api/state?player=P1`. Le plan cible un MVP jouable entre amis avec
plusieurs parties simultanées, des invitations privées, une projection serveur
filtrée et une restauration après redémarrage. Les documents de travail sont
regroupés dans [`prompts/`](prompts/) avec un nom thématique :

- [`identifiants-territoires.md`](prompts/identifiants-territoires.md)
- [`api-production.md`](prompts/api-production.md)
- [`authentification-sessions.md`](prompts/authentification-sessions.md)
- [`persistance.md`](prompts/persistance.md)
- [`front.md`](prompts/front.md)
- [`deploiement.md`](prompts/deploiement.md)

## Périmètre suivi

- identité territoriale et contrats publics ;
- API de plusieurs parties actives et soumission par joueur ;
- authentification Firebase, invitations et reprise de session/membership ;
- persistance et restauration ;
- vues privées côté serveur ;
- parcours front multi-joueur ;
- déploiement et stockage durable.

## Contrats figés du MVP hébergé

Les décisions structurantes sont communes à toutes les sous-issues online :

- plusieurs parties actives par déploiement, chacune pour deux à huit joueurs
  online ;
- les routes conservent `/api/games/{id}` pour l'évolution multi-parties ;
- `chain: null` signifie qu'aucune chaîne n'est active et
  `chain.visibility: "hidden"` qu'une chaîne existe sans détail révélé ;
- les combats portent `visibility: "exact"` ou `"general"`, avec suppression
  des forces et identifiants dans la vue générale ;
- la résolution forcée est explicite et aucune deadline automatique n'est
  ajoutée ;
- l'identité est fournie par Firebase Authentication avec un lien de connexion
  par email ; le serveur valide le JWT Firebase porté en Bearer ;
- les profils, appartenances et parties sont persistés dans Firestore ; aucun
  token Firebase ni secret de session n'est écrit dans Firestore ;
- la session persistante est fournie par Firebase côté navigateur et par les
  profils/memberships Firestore côté serveur ; aucune collection
  `sessions/{token}` n'est nécessaire ni autorisée ;
- les codes d'invitation sont opaques, de six caractères, stockés sous forme
  hachée et vérifiés uniquement par le backend ;
- le front utilise le SDK Firebase Web pour conserver la session côté client et
  `onSnapshot` uniquement sur les projections Firestore lisibles par le joueur ;
  l'état canonique, les ordres bruts et les métadonnées privées restent
  inaccessibles au client ;
- les transitions de partie et la résolution utilisent des transactions et des
  préconditions Firestore ; la cohérence ne dépend pas de `max-instances` ;
- `net/http` et `http.ServeMux` sont utilisés, tandis que les endpoints hotseat
  restent dev-only ;
- la route `GET /api/games/{id}/my-submission` expose au joueur membre sa
  dernière soumission du tour courant (chaînes et hiver) pour réhydrater les
  zones de saisie après refresh ou reconnexion, sans exposer les ordres des tiers.

## Persistance Firestore

Firestore Native mode est la frontière de persistance du MVP hébergé. Le
serveur Go utilise le SDK Admin/Cloud Firestore avec les credentials ADC du
service account Cloud Run. Le moteur reste pur et ne connaît ni Firestore ni
Firebase Authentication.

Les documents lisibles par le navigateur sont des projections dédiées :

- `games/{gameId}` contient uniquement les métadonnées publiques, les slots et
  les statuts de soumission, jamais l'état canonique ni le code d'invitation ;
- `games/{gameId}/views/{uid}` contient l'état projeté pour le joueur dont le
  `uid` est dans le chemin ;
- `games/{gameId}/reports/{uid}/turns/{turn}` contient les rapports filtrés de
  ce joueur ;
- `players/{uid}` contient le profil du joueur et les métadonnées nécessaires à
  la liste de ses parties, ainsi que ses réglages de notifications par email
  (objet `notifications` : `newTurn`, `privateMessage`, `publicMessage`,
  `reduceWhenActive`, `language`). Un profil sans cet objet lit tout à
  `false` : aucune migration ni changement de `schemaVersion`.

Les documents réservés au backend comprennent l'état moteur, les soumissions
brutes, les rapports non filtrés, les métadonnées de confidentialité et les
informations hachées d'invitation. Les règles Firestore refusent toute lecture
ou écriture cliente de ces documents. Les écritures clientes passent par l'API
Go, qui vérifie le JWT et l'appartenance à la partie.

Une soumission est enregistrée avec le numéro de tour attendu dans une
transaction. La résolution est revendiquée une seule fois avec une précondition
sur la révision de la partie ; le moteur déterministe calcule ensuite le nouvel
état et un commit conditionnel publie l'état canonique, le rapport et toutes
les projections. Une tentative concurrente est rejetée ou rejouée après
relecture, sans double résolution. Une revendication abandonnée possède un
lease interne récupérable ; ce lease n'est pas une deadline de jeu.

Le free tier Firestore et Cloud Run est un objectif de coût, pas une garantie :
les lectures des listeners, les écritures de projections, le stockage, les
logs, la région et l'activation d'un compte de facturation peuvent générer des
coûts. Des alertes de budget et une documentation des quotas sont obligatoires
avant le déploiement public.

## Notifications par email

Un joueur peut demander un email quand un nouveau tour commence dans l'une de
ses parties, quand sa maison reçoit un message privé et quand une autre maison
écrit dans le salon public. Ces trois réglages, l'option « réduire les
notifications » et la langue des emails se règlent sur la page de profil ; les
notifications sont désactivées par défaut
(`PUT /api/auth/me/notifications`, voir [`architecture.md`](architecture.md)).
Le chat est décrit dans [`chat.md`](chat.md).

- **Destinataires d'un nouveau tour.** Chaque membre qui tient une maison non
  éliminée, dans une partie encore en cours après la résolution : rien quand la
  partie vient de se terminer, et rien pour l'observateur. L'envoi suit une
  soumission ou une résolution forcée qui a effectivement résolu le tour.
- **Condition d'envoi.** Le réglage correspondant est activé, le profil porte un
  email et, si `reduceWhenActive` est activé, le joueur n'était pas actif au
  moment de l'événement. Les messages ajoutent la règle anti-spam de
  [`chat.md`](chat.md).
- **Joueur actif.** Le serveur enregistre l'heure de la dernière requête API
  authentifiée de chaque joueur (aucun signal de présence côté client) ; un
  joueur est actif si cette heure se situe dans les cinq minutes qui précèdent
  l'événement. Cette heure est écrite au plus une fois toutes les deux minutes
  par joueur et par instance du serveur.
- **Langue.** Le joueur choisit le français ou l'anglais dans son profil. Tant
  qu'il n'a rien choisi, le profil propose la langue de l'application et
  l'enregistre avec le premier réglage sauvegardé ; un profil sans langue
  reçoit ses emails en français.
- **Cache.** Le serveur garde une minute en mémoire le profil lu pour un
  destinataire et le résultat de la règle anti-spam d'un joueur dans une
  partie : un réglage modifié, ou un chat relu, est pris en compte au plus
  une minute plus tard.
- **Contenu.** Texte brut dans la langue du joueur : le nom de la partie, un lien vers
  la partie et un lien vers le profil pour modifier ou désactiver les
  notifications. Aucun message, auteur ni ordre.
- **Envoi.** Le paquet `internal/notify` envoie par SMTP (bibliothèque standard
  Go, STARTTLS ou TLS implicite, jamais en clair) derrière une interface
  `Mailer` ; il ne dépend ni de Firestore ni de Firebase. Sans `SMTP_HOST` ou
  `SMTP_FROM`, un mailer neutre journalise une ligne par email ignoré
  (partie et uid, ni adresse ni contenu). Les variables d'environnement sont
  décrites dans [`architecture.md`](architecture.md).
- **Moment de l'envoi.** Le handler écrit et vide sa réponse, puis envoie les
  emails dans la même requête, avec un délai de dix secondes, car Cloud Run
  limite le CPU une fois la réponse partie. Un échec d'envoi est journalisé et
  ne change jamais le résultat HTTP.
- **Hotseat et développement.** Aucune notification en hotseat ni avec
  `ONLINE_DEV_MODE` : le notifier n'est monté que là où le chat l'est.

Documents réservés au backend, refusés aux clients par `firestore.rules` :

- `player_activity/{uid}` : `lastActiveAt`, heure de la dernière requête API ;
- `games/{gameId}/notify_state/{uid}` : `messageEmailAt`, heure du dernier
  email de message envoyé au joueur pour cette partie.

## Chat

Le chat interne (salon public et conversations privées entre deux maisons) est
disponible online uniquement ; les routes sont décrites dans
[`architecture.md`](architecture.md). Les messages sont écrits par l'API Go,
jamais par le client, et lus par `onSnapshot` sous `games/{gameId}/` :

- `chat_public/{messageId}` : salon public, lisible par tous les membres ;
- `chat_private/{clé}/messages/{messageId}` : conversation entre deux maisons,
  dont la clé est leurs identifiants triés et joints par `_` (`P1_P3`). Chaque
  message porte `participantUids` ; les règles l'autorisent aux deux
  participants et à l'observateur (`spectatorUid`), qui lit tout ; le créateur
  d'une partie qui joue n'a pas d'autre accès que les autres joueurs. Le client interroge une
  conversation avec `where("participantUids", "array-contains", uid)` et
  `orderBy("createdAt", "desc")` (index composite dans
  `firestore.indexes.json`) ;
- `chat_reads/{uid}` : curseurs de lecture `channels.{canal}` (`lastReadAt`),
  lisibles par leur propriétaire ;
- `chat_state/{uid}` : état de la limite d'un message par seconde, réservé au
  backend (vérifié dans la même transaction que l'écriture du message).

Les messages sont ordonnés par `createdAt` puis par identifiant ; la
pagination par curseur reprend après le plus ancien message déjà reçu.

Règles de fonctionnement, détaillées dans [`chat.md`](chat.md) :

- **Canaux.** Le salon public et une conversation privée par paire de
  maisons ; pas de groupe libre. Les messages sont signés de la maison, jamais
  du compte.
- **Cloisonnement.** Un joueur lit le salon public et ses propres
  conversations. Seul l'observateur, l'hôte d'une partie créée sans maison,
  lit toutes les conversations, sans pouvoir écrire. Un hôte qui joue une
  maison n'a aucun accès de plus que les autres joueurs.
- **Historique.** Conservé sans purge ; le client lit les 50 derniers messages
  de chaque canal et pagine vers le passé.
- **Non-lus.** Comptés par canal à partir de `chat_reads/{uid}`, mis à jour
  par `POST /api/games/{id}/messages/read`.
- **Hotseat.** Aucune route de chat n'est montée : `404 chat_disabled`.
- **Ordres.** Écrire un message ne soumet rien et ne modifie pas la partie.

Les fixtures distinguent volontairement les variantes de confidentialité afin
que les sous-issues O2 à O8 puissent les réutiliser sans redéfinir le contrat.

Le découpage en sous-issues, les critères de validation manuelle et les
garanties de stockage sont décrits dans [`online-plan.md`](online-plan.md).

Les extensions de règles décrites dans les autres fichiers ne doivent pas être
ajoutées à l'issue #2 lorsqu'elles ne sont pas nécessaires à la finalisation du
mode online.
