# Roadmap produit : Crown & Borough v1

Ce document décrit l'état de référence de la v1 et les évolutions qui restent
à suivre dans GitHub. Les plans d'implémentation et les prompts correspondant
aux étapes terminées ont été supprimés : le code et les tests sont désormais la
source de vérité pour les fonctionnalités livrées.

## État v1

La v1 comprend :

- le moteur de jeu en Go pur ;
- la génération déterministe d'une carte de 8 territoires par joueur ;
- les chaînes d'ordres, leur progression simultanée, les combats, les retraites,
  les jonctions, les dispersions et le pillage ;
- le ravitaillement, la famine et la résolution des ordres d'hiver ;
- les rapports de tour et la boucle de jeu locale ;
- le front SVG React ;
- le serveur multijoueur en mémoire, utilisable en ligne entre plusieurs
  joueurs : chaque joueur soumet ses ordres séparément et le tour se résout
  lorsque tous les joueurs ont soumis, ou lorsqu'un joueur force la résolution.

La carte et les données dynamiques chiffrées sont actuellement communes à tous
les clients. La politique de divulgation privée des chaînes et des combats est
définie dans le GDD v1, mais son filtrage serveur reste une fonctionnalité
online à réaliser et à suivre par issue.

## Fonctionnalités livrées

| Domaine | Fonctionnalités v1 | État |
|---|---|---|
| Fondations | Module Go, chargement des assets CSV, balance JSON, front Vite/React/TypeScript | Fait |
| Carte | Génération Voronoï seedée, territoires nommés par trigramme, frontières franchissables ou infranchissables, graphe connexe, villages neutres | Fait |
| Modèle | Joueurs, territoires, armées uniques par territoire, nobles, infrastructures, stocks, chaînes | Fait |
| Ordres | Parser texte, ordres A/S/H/J/P/O/K/D, liaisons `single` et `loop`, validation et remplacement atomique des chaînes | Fait |
| Résolution | Progression simultanée, attaques, soutiens, combats multi-contendants, retraites, jonctions, dispersions, contrôle territorial | Fait |
| Logistique | Rations de terrain, ravitaillement BFS, portée, dépôts de vivres, coûts exponentiels, stocks et famine | Fait |
| Hiver | Recrutement, constructions v1, capitale, libération des nobles, conservation et rapatriement des stocks | Fait |
| Boucle de jeu | Cycle printemps/été/automne/hiver, rapport de tour, partie initiale déterministe | Fait |
| Front | Carte interactive, poste de commandement, sélection de joueur, ordres par noble, rapport et signalisation de l'hiver | Fait |
| Online v1 | Session unique en mémoire, création/réinitialisation de partie, soumission par joueur, résolution synchrone, résolution forcée, endpoint de ravitaillement | Fait |

## Contraintes et décisions v1

- Une partie accepte de 2 à 16 joueurs.
- La carte contient 8 territoires par joueur et `joueurs + 1` villages
  neutres.
- Les villages neutres produisent et stockent leur production. Leur stock est
  inaccessible avant capture et reste sur place lors de la capture.
- Une seule infrastructure occupe une case. Les infrastructures appartiennent
  à leur case, jamais à un joueur ; le contrôleur de la case en bénéficie.
- Une armée est l'unique entité de force d'un territoire et porte un propriétaire
  et une taille. Les ordres s'appliquent à toute l'armée.
- Les identifiants territoriaux internes séquentiels et les trigrammes publics
  coexistent encore. La suppression de ce double identifiant est une évolution
  online distincte.
- Une non-adjacence rencontrée pendant la résolution casse la chaîne à cet
  ordre ; les ordres précédents restent valides et le suffixe n'est pas joué.
- Une chaîne reçue remplace immédiatement la chaîne précédente de l'armée
  concernée. Il n'existe pas de mécanisme de modification partielle.
- Une armée sans chaîne est simplement Sans Ordre : elle ne reçoit aucune
  action automatique.
- Les infrastructures v1 sont le moulin, le dépôt de vivres, le château et le
  village. Les anciennes structures liées à une couche d'information ne font
  pas partie de la v1.

## Fonctionnalités online à suivre

Les prompts `p3.x` restent temporairement dans `specs/prompts/` comme matériau
de travail. Leur réalisation doit être suivie dans une issue GitHub dédiée,
avec une issue séparée pour chaque bug ou fonctionnalité qui le nécessite.

| Sujet | Périmètre | État |
|---|---|---|
| Identité territoriale | Remplacer le couple `TerritoryID`/`Code` par le trigramme territorial dans le domaine et les contrats | À suivre |
| Vue privée par joueur | Filtrer côté serveur les chaînes connues et les détails des combats selon le joueur | À suivre |
| API de production | Parties multiples, ressources d'une partie, contrats REST stabilisés et gestion des erreurs | À suivre |
| Authentification | Identité, sessions, invitation et reprise d'un emplacement de joueur | À suivre |
| Persistance | Sauvegarde et restauration JSON d'une partie sans perte après redémarrage | À suivre |
| Front | Retours de tests, accessibilité, parcours multi-joueur et affichage des vues privées | À suivre |
| Déploiement | Image de production, stockage durable et déploiement public | À suivre |

## Extensions de règles

Les politiques, ordres spéciaux et autres règles que le GDD accueillera plus
tard doivent être ajoutés comme des compléments à ce cœur v1. Ils ne doivent
pas modifier les invariants de base : résolution simultanée, armée unique par
territoire, chaînes d'ordres, ravitaillement, famine, hiver et contrôle
territorial.

Tout ajout ou bug découvert après la v1 est suivi dans GitHub plutôt que par un
nouveau plan d'implémentation local.
