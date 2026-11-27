# Chat interne

Le chat permet aux maisons d'une partie online de s'écrire : un salon public
et des conversations privées entre deux maisons. Il n'existe pas en hotseat
(`404 chat_disabled`) et n'a aucun effet sur les ordres ni sur l'état de la
partie. Les routes sont décrites dans [`architecture.md`](architecture.md), le
stockage Firestore et ses règles de lecture dans [`online.md`](online.md).

## Principes

- **L'auteur est une maison.** Un message ne porte ni pseudo, ni email, ni
  uid : seulement `authorHouseId`. Le front affiche le nom de la maison avec
  son blason et sa couleur.
- **Écriture par l'API Go uniquement.** Le client lit Firestore par
  `onSnapshot` et écrit par `POST /api/games/{id}/messages`.
- **Un joueur lit le salon public et ses propres conversations**, quel que
  soit son rôle : le créateur d'une partie qui joue n'a aucun accès de plus.
- **Seul l'observateur** (la partie créée sans maison pour son créateur) lit
  toutes les conversations privées, en lecture seule, et n'écrit nulle part
  (`403 no_house`).
- **Maisons éliminées.** Une maison éliminée n'est destinataire que d'une
  conversation déjà ouverte ; son joueur peut toujours écrire. Une maison est
  « en jeu » tant qu'elle tient un territoire, un fief, une capitale ou une
  armée.

## Messages

| Champ | Contenu |
|---|---|
| `id` | identifiant triable par date de création |
| `channel` | `public`, ou la clé de conversation (identifiants des deux maisons triés et joints par `_`, par exemple `P1_P3`) |
| `authorHouseId` | maison auteur |
| `body` | 500 caractères au plus (runes) |
| `createdAt` | horodatage serveur, microsecondes |
| `turn` | `number`, `season`, `year` du tour courant à l'écriture |
| `batchId`, `part`, `parts` | morceaux d'un texte long (ensemble cohérent, 8 morceaux au plus) |
| `kind`, `previousName` | `kind: "house_renamed"` marque l'événement système écrit dans le salon public quand une maison d'une partie incognito change de nom : `body` est le nouveau nom, `previousName` l'ancien. Absent d'un message ordinaire ; ces messages ne sont soumis ni à la limite de débit ni à la validation du corps, et n'envoient aucun email. |

Les messages sont ordonnés par `createdAt` puis par identifiant. La limite est
d'un message par seconde et par joueur (`429 rate_limited`).

## Notifications par email

Un joueur peut recevoir un email pour un message (voir
[`online.md`](online.md) pour le stockage, l'envoi et la définition d'un joueur
actif). Deux réglages de profil, désactivés par défaut, couvrent les messages :

- **Messages privés** : l'email part vers l'autre maison de la conversation ;
- **Salon public** : l'email part vers toutes les maisons sauf l'auteur.

L'observateur ne reçoit rien. L'option « réduire les notifications » supprime
l'email d'un joueur actif au moment du message.

- **Un email par lecture.** Après un email de message pour une partie, aucun
  autre n'est envoyé au joueur pour cette partie, privé ou public, tant qu'un
  de ses curseurs de lecture (`chat_reads/{uid}`) n'a pas un `lastReadAt`
  postérieur à cet email. La vérification et l'enregistrement de l'heure
  (`messageEmailAt`) sont atomiques, si bien que des messages simultanés
  n'envoient qu'un email, et un texte long découpé en huit morceaux n'en
  envoie qu'un. Un email supprimé par l'option « réduire », un email dont
  l'envoi échoue et un email ignoré faute de serveur SMTP ne consomment rien.
  Le résultat est gardé une minute en mémoire (voir [`online.md`](online.md)) :
  un message reçu dans la minute qui suit une lecture peut ne pas déclencher
  d'email.
- **Aucun contenu.** L'email ne contient ni le texte du message, ni sa maison
  auteure : il indique le nom de la partie, renvoie vers elle et vers le profil.

## Annonce Discord du salon public

Un message du salon public peut aussi être annoncé sur le webhook Discord de la
partie, si l'hôte en a configuré un (voir [`online.md`](online.md) pour la
configuration, les modèles, les mentions et le secret de l'URL). Les messages
privés n'en produisent jamais.

- **Une annonce par fenêtre.** La règle « un email par lecture » est propre à
  chaque joueur ; ici la limite est par partie : au plus une annonce de salon
  public toutes les 5 minutes, quel que soit le nombre de messages ou
  d'auteurs, décidée par une réservation atomique (`lastPublicAt`) qui ne
  consomme rien quand l'envoi échoue. Un texte long découpé en morceaux n'en
  produit qu'une.
- **Aucun contenu.** L'annonce ne contient ni le texte du message ni sa maison
  auteure : elle dit seulement, avec le modèle de l'hôte, qu'un message a été
  écrit dans le salon public de la partie (nom de la partie, tour, saison,
  lien).

## Panneau de chat

Le panneau est un onglet « Messages » du tiroir de la partie online (même
ergonomie que les autres onglets, voir [`charte-graphique.md`](charte-graphique.md)).

### Canaux

- Le salon public est toujours proposé.
- Les conversations privées existantes (au moins un message) sont listées,
  la plus récente en premier, avec le blason de l'interlocuteur. Une
  conversation avec une maison éliminée reste visible et utilisable.
- Les canaux se choisissent dans un sélecteur (liste déroulante accessible au
  clavier) : chaque entrée, ainsi que le déclencheur pour le canal ouvert,
  montre le blason de la maison suivi de son nom et de son nombre de
  non-lus ; le salon public n'a pas de blason, et une paire vue par
  l'observateur en montre deux.
- Une liste « Nouvelle conversation », de même forme (blason et nom), ne
  propose que les maisons encore en jeu avec lesquelles il n'y a pas encore
  de conversation.
- L'observateur voit toutes les paires de maisons, chacune avec un cadenas et
  la mention « Lecture seule » ; le champ de saisie est remplacé par une
  note.

### Lecture et pagination

Chaque canal est écouté par `onSnapshot` sur ses 50 derniers messages. Une
conversation d'un joueur est interrogée avec
`where("participantUids", "array-contains", uid)` ; celles que seul l'observateur
voit sont lues sans filtre, et ne chargent que leur dernier message tant
qu'elles ne sont pas ouvertes. « Charger les messages plus anciens » lit la
page suivante après le plus ancien message reçu, par curseur de document.

### Non-lus, non répondus, lus

Chaque canal (salon public compris) est dans l'un de trois états, du point de
vue de la maison du joueur :

| État | Définition |
|---|---|
| Non lu | au moins un message d'une autre maison postérieur à `lastReadAt` |
| Non répondu | tout est lu, mais le dernier message du canal vient d'une autre maison, postérieur à `acknowledgedAt`, et la maison du joueur n'a rien écrit depuis |
| Lu (et répondu) | le dernier message est de la maison du joueur, ou le canal est vide, ou le marqueur « non répondu » a été acquitté |

Les messages système (`kind: house_renamed`) sont ignorés pour déterminer le
dernier message. L'élimination d'une maison n'entre pas en compte. Le bouton
« Pris en compte » (« Acknowledged »), visible seulement dans l'état non
répondu, retire le marqueur sans répondre par
`POST /api/games/{id}/messages/acknowledge` (qui avance `acknowledgedAt`, jamais
en arrière) ; un nouveau message d'une autre maison rend le canal non lu, puis
non répondu. Répondre retire le marqueur sans autre action. Le non-répondu
porte un indicateur distinct de la pastille de non-lu, dans le sélecteur de
canal et sur son déclencheur (pour les autres canaux) ; il ne compte pas dans
la pastille de l'onglet.

Le nombre de non-lus d'un canal est celui des messages d'autres maisons
postérieurs à `lastReadAt` (tous, si le canal n'a jamais été lu). Sur l'onglet,
une pastille affiche la somme. Dans le sélecteur de canal, chaque entrée avec
du non-lu porte une pastille par maison auteure, à la couleur de cette maison ;
le déclencheur du sélecteur affiche de même le non-lu des autres canaux.
À la première ouverture du panneau, une fois le salon public et les
conversations de la maison chargés, l'ordre de priorité est : d'abord les
canaux non lus, puis les canaux non répondus. Au premier niveau non vide, le
seul canal concerné (salon public compris) est ouvert directement ; s'il y en a
plusieurs, ou aucun à aucun niveau, le salon public l'est. Ce choix n'est fait qu'une fois, et un canal choisi par le joueur
entre-temps est conservé ; aucun curseur n'avance avant ce choix. Un canal
ouvert dans un panneau visible compte pour zéro ; son curseur avance par
`POST /api/games/{id}/messages/read` dès que des messages d'autres maisons
sont à l'écran. L'observateur n'a ni curseur ni pastille.

### Séparateurs de tour

Dans chaque canal, un séparateur « Printemps 1011 » ouvre chaque tour qui
compte au moins un message. Un séparateur final « (tour en cours) » nomme le
tour courant, lu dans la projection de la partie, quand le dernier message
est d'un tour antérieur. Un tour sans message n'a pas de séparateur.

### Messages longs

Un texte peut compter jusqu'à 4 000 caractères.

- Il est découpé en morceaux de 500 caractères au plus : au saut de
  paragraphe, sinon après une phrase, sinon entre deux mots. Un mot n'est
  coupé que s'il dépasse à lui seul 500 caractères. Si les meilleures
  coupures donnent plus de 8 morceaux, les coupures entre mots sont
  utilisées ; si 8 morceaux ne suffisent toujours pas, l'envoi est refusé.
- Les morceaux sont envoyés l'un après l'autre : le suivant part après
  l'accusé du précédent et au moins une seconde plus tard. Chaque morceau
  porte `batchId`, `part` et `parts`.
- La progression (« Envoi 3/8 ») et un bouton Annuler sont affichés. En cas
  d'échec ou d'annulation, le texte non envoyé revient dans le champ de
  saisie.
- À l'affichage, les morceaux consécutifs d'un même lot et d'un même auteur
  forment un seul bloc.

### États et accessibilité

- États vides : salon sans message, conversation sans message, aucune autre
  maison en jeu.
- Erreurs : `429` (le texte non envoyé est rendu), réseau, destinataire
  refusé, lecture impossible ; elles s'annoncent par `role="alert"`.
- La liste des messages est un `role="log"` focalisable ; Entrée envoie,
  Maj+Entrée insère un saut de ligne ; le compteur de caractères et la
  notice de découpage sont lus par les lecteurs d'écran.
- Sur téléphone, le chat est un onglet de la barre du bas ; le panneau
  occupe l'écran.

## Tests

Le découpage, les séparateurs, le calcul des non-lus et l'envoi séquentiel
(échec, annulation, délai) sont des fonctions pures testées dans
`web/src/lib/chat.test.ts` ; le panneau l'est dans
`web/src/components/ChatPanel.test.tsx`. Les règles Firestore et le backend
ont leurs propres tests (`firestore.rules.test.ts`, `internal/api/chat_test.go`).
Les notifications sont testées dans `internal/notify` (décision d'envoi,
notifier avec un mailer factice, anti-spam, webhook Discord avec un faux
serveur `httptest`) et `internal/api/notifications_test.go`.
