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

Les messages sont ordonnés par `createdAt` puis par identifiant. La limite est
d'un message par seconde et par joueur (`429 rate_limited`).

## Panneau de chat

Le panneau est un onglet « Messages » du tiroir de la partie online (même
ergonomie que les autres onglets, voir [`charte-graphique.md`](charte-graphique.md)).

### Canaux

- Le salon public est toujours proposé.
- Les conversations privées existantes (au moins un message) sont listées,
  la plus récente en premier, avec le blason de l'interlocuteur. Une
  conversation avec une maison éliminée reste visible et utilisable.
- Une liste « Nouvelle conversation » ne propose que les maisons encore en
  jeu avec lesquelles il n'y a pas encore de conversation.
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

### Non-lus

Le nombre de non-lus d'un canal est celui des messages d'autres maisons
postérieurs à `lastReadAt` (tous, si le canal n'a jamais été lu). Une pastille
s'affiche par canal et sur l'onglet (somme). Un canal ouvert dans un panneau
visible compte pour zéro ; son curseur avance par
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
