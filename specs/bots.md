# Ligue de bots

Une ligue est une partie online jouée **uniquement par des agents** : chaque
maison est confiée à un modèle de langage choisi par l'observateur humain, qui
suit la partie sans y jouer. L'outil est `cmd/botleague` ; son mode d'emploi
est la skill `.opencode/skills/crown-and-borough-league`.

## Principes

- **Un orchestrateur déterministe, des modèles qui décident.** `botleague`
  gère l'authentification, la création et la jonction de partie, la boucle de
  tours, le chat, la mémoire et les reprises. Les modèles ne produisent que
  des messages, des ordres et leur mémoire.
- **Aucun navigateur.** L'authentification passe par l'API REST de Firebase :
  l'émulateur expose le code de connexion, un serveur déployé l'envoie par
  mail (le lien est fourni à l'outil).
- **L'observateur crée la partie** (`spectate`) : il lit l'état complet et
  toutes les conversations privées, sans jouer ni écrire. En local, l'outil
  crée la partie avec le compte `host.email` ; sur un serveur déployé,
  l'observateur la crée dans l'application et donne l'URL d'invitation.
- **Un modèle par maison**, via toute API compatible OpenAI (OpenRouter,
  OpenCode Zen, Ollama, LM Studio). Les clés restent dans l'environnement.
- **Les personas** (`play_style`, `trust`, `tone`, règles de décision) sont
  des fichiers YAML embarqués dans le binaire (`cmd/botleague/personas/`).
- **Diplomatie par le chat interne** : salon public et conversations privées,
  500 caractères par message, un message par seconde et par joueur.
- **Mémoire privée par maison** : texte réécrit par le modèle à chaque appel
  (ancre d'identité restaurée, taille bornée) et historique des issues écrit
  par l'orchestrateur ; le contexte d'un appel reste de taille constante.

## Déroulement d'un tour

1. `negotiation_rounds` tours de messages seuls : chaque bot lit le chat
   (nouveaux messages signalés), répond et propose.
2. Un tour d'ordres : chaque bot voit les réponses, écrit ses ordres, les
   envoie ; en cas de rejet, l'erreur du serveur lui est renvoyée (jusqu'à
   `max_repairs` corrections), puis la maison passe son tour.
3. Le serveur résout quand toutes les maisons ont soumis ; au-delà de
   `turn_timeout_minutes`, l'observateur local force la résolution.

La partie reprend là où elle s'est arrêtée (`session.json`, mémoires, curseurs
de lecture) quand la commande est relancée.

## Limites connues

- Les modèles gratuits suivent inégalement la syntaxe des ordres : le journal
  de ligue compte les tours passés par maison.
- Le mode déployé suppose que l'observateur transmet les liens de connexion
  reçus par mail.
