# Roadmap produit : Crown & Borough

Ce document décrit l'état de référence de la v1 et les évolutions qui restent
à suivre dans GitHub. Les plans d'implémentation et les prompts correspondant
aux étapes terminées ont été supprimés : le code et les tests sont désormais la
source de vérité pour les fonctionnalités livrées.

## État v1

La v1 comprend :

- le moteur de jeu en Go pur ;
- la génération déterministe d'une carte de `8 x N` territoires de jeu et de
  `(N + 1) x 4` territoires dédiés aux villages ;
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
| Carte | Génération Voronoï seedée, territoires nommés par trigramme, frontières franchissables ou infranchissables, graphe connexe, villages neutres et territoires dédiés | Fait |
| Modèle | Joueurs, territoires, armées uniques par territoire, nobles, infrastructures, stocks, chaînes | Fait |
| Ordres | Parser texte, ordres A/S/H/J/P/D, liaisons `single` et `loop`, validation et remplacement atomique des chaînes ; statuts nobles en ordres d'hiver O/P | Fait |
| Résolution | Progression simultanée, attaques, soutiens, combats multi-contendants, bonus de commandement noble, retraites, jonctions, dispersions, contrôle territorial | Fait |
| Logistique | Rations de terrain, ravitaillement BFS, portée, dépôts de vivres, coûts exponentiels, stocks et famine | Fait |
| Hiver | Recrutement, constructions v1, capitale, libération des nobles, conservation et rapatriement des stocks | Fait |
| Boucle de jeu | Cycle printemps/été/automne/hiver, rapport de tour, partie initiale déterministe | Fait |
| Front | Carte interactive, poste de commandement, sélection de joueur, ordres par noble, rapport et signalisation de l'hiver | Fait |
| Online v1 | Session unique en mémoire, création/réinitialisation de partie, soumission par joueur, résolution synchrone, résolution forcée, endpoint de ravitaillement | Fait |

## Contraintes et décisions v1

- Une partie accepte de 2 à 16 joueurs.
- La carte contient `8 x N` territoires de jeu et `(N + 1) x 4` territoires
  supplémentaires dédiés aux `N + 1` villages neutres.
- Les `N` châteaux de départ sont placés sur des territoires qui ne portent pas
  ces villages neutres, avec au moins 4 étapes franchissables entre deux
  départs ; ils ne consomment donc aucun des `N + 1` villages.
- Les villages neutres produisent et stockent leur production. Leur stock est
  inaccessible avant capture et reste sur place lors de la capture.
- Une seule infrastructure occupe une case. Les infrastructures appartiennent
  à leur case, jamais à un joueur ; le contrôleur de la case en bénéficie.
- Une armée est l'unique entité de force d'un territoire et porte un propriétaire
  et une taille. Les ordres s'appliquent à toute l'armée.
- Le trigramme territorial est l'identifiant unique d'une case dans le domaine,
  les contrats et les références de résolution ; aucun matricule séquentiel
  n'est généré ou exposé.
- Une non-adjacence rencontrée pendant la résolution casse la chaîne à cet
  ordre ; les ordres précédents restent valides et le suffixe n'est pas joué.
- Une chaîne reçue remplace immédiatement la chaîne précédente de l'armée
  concernée. Plusieurs chaînes ciblant la même armée au même tour sont toutes
  rejetées ; il n'existe pas de mécanisme de modification partielle.
- Une armée sans chaîne est simplement Sans Ordre : elle ne reçoit aucune
  action automatique.
- Les infrastructures v1 sont le moulin, le dépôt de vivres, le château et le
  village. Les anciennes structures liées à une couche d'information ne font
  pas partie de la v1.
- Lever une troupe exige un noble libre du joueur sur la cible ou un territoire
  adjacent à celle-ci par une frontière franchissable ; recruter un noble
  conserve les exigences du château ou village et de l'armée sur la cible.

## Organisation des spécifications

L'index thématique est disponible dans [`specs/README.md`](README.md). Les
documents ne portent pas de numéro de version dans leur nom : les versions
d'atterrissage seront décidées dans GitHub lorsque les dépendances seront mieux
stabilisées.

## Fonctionnalités online à suivre

Les prompts liés au mode online restent temporairement dans
[`specs/prompts/`](prompts/) comme matériau de travail. Leur réalisation est
suivie dans [l'issue #2](https://github.com/fogfactory/crown-and-borough/issues/2),
avec une sous-issue testable pour chaque étape du plan
[`specs/online-plan.md`](online-plan.md). Les étapes sont regroupées dans les
milestones GitHub `Online Foundations`, `Online Friends MVP` et `Online Hosted`.

| Sujet | Périmètre | État |
|---|---|---|
| Contrats et architecture online | Figer les contrats JSON, les routes, la confidentialité, la résolution forcée et la frontière Firestore ; fixtures dans [`specs/fixtures/`](fixtures/) | Fait : O1 |
| Identité territoriale | Utiliser le trigramme territorial comme identifiant unique dans le domaine et les contrats | Fait : O2-O3 |
| Vue privée par joueur | Filtrer côté serveur les chaînes connues et les détails des combats selon le joueur | Planifié : O4 |
| API de production | Plusieurs parties, ressources d'une partie, contrats REST stabilisés et gestion des erreurs | Planifié : O5 |
| Authentification | Firebase Auth par lien email, profils Firestore, invitations et membership par UID | Planifié : O6 |
| Persistance | Transactions, projections privées et restauration Firestore sans perte après redémarrage | Planifié : O8 |
| Front | Retours de tests, accessibilité, parcours multi-joueur, Firebase Web et listeners temps réel | Planifié : O7 |
| Déploiement | Image de production, Firestore/Firebase et déploiement public Cloud Run | Planifié : O9-O10 |

## Extensions de règles

Les politiques, ordres spéciaux et autres règles que le GDD accueillera plus
tard doivent être ajoutés comme des compléments à ce cœur v1. Ils ne doivent
pas modifier les invariants de base : résolution simultanée, armée unique par
territoire, chaînes d'ordres, ravitaillement, famine, hiver et contrôle
territorial.

Tout ajout ou bug découvert après le socle actuel est suivi dans GitHub plutôt
que par un nouveau plan d'implémentation local. Les spécifications thématiques
servent à conserver les règles et les décisions ; les issues servent à planifier
et livrer le travail.
