# Titres et victoire

**Milestone lié :** [Titres & Victoire](https://github.com/fogfactory/crown-and-borough/milestone/2)

**Dépend de :** la carte, le contrôle territorial et le ravitaillement du
socle actuel ; les flux de ressources dépendent de
[Économie et prospérité](economie.md) et les titres religieux de
[Religieux](religieux.md).

## Occupé contre contrôlé

Le socle actuel ne connaît qu'un seul statut territorial, positionnel : une
case reste au dernier joueur dont l'armée s'y est arrêtée. Les titres
introduisent un second statut, plus stable, et distinguent :

- **occupé** : le statut positionnel actuel, inchangé — dernier joueur dont
  l'armée s'est arrêtée sur la case. Un territoire occupé rapporte des
  ressources à la **capitale du joueur** (voir [economie.md](economie.md)).
- **contrôlé** : le territoire fait partie d'un fief constitué. Il rapporte des
  ressources à la **capitale du fief**, sans dépendre de la présence d'une
  armée — une armée ennemie qui s'y arrête n'interrompt pas la production tant
  que la capitale du fief elle-même n'est pas prise.

**Exception :** la capitale d'un joueur est toujours considérée comme
contrôlée, même si elle n'appartient à aucun fief constitué — elle agit comme
un fief gratuit et implicite de taille 1. Ce statut suit la capitale à chaque
redésignation (`E C XXX`) : tout château nouvellement désigné capitale devient
contrôlé s'il ne l'était pas déjà.

La règle devra préciser ce qu'il advient d'un fief dont la capitale est prise
par un autre joueur (le fief tombe-t-il entièrement, ou seule la capitale
change-t-elle de contrôleur ?) et si une armée ennemie stationnée sur un
territoire contrôlé (hors capitale) peut malgré tout bloquer son propre
ravitaillement au passage.

## Constitution d'un fief

Un fief se constitue en rachetant un groupe de territoires **adjacents,
occupés par le même joueur et contenant un château**. Le rachat transforme ces
territoires d'occupés à contrôlés ; le château devient la **capitale du
fief**. Le coût et la dénomination dépendent de la taille du groupe racheté :

| Titre | Territoires inclus | Coût indicatif |
|---|---:|---:|
| Baronnie | 3 | 6 R |
| Comté | 4 | 8 R |
| Duché | 6 | 12 R |

> À trancher : la table historique incluait un palier Marquisat (5
> territoires, 10 R) entre comté et duché ; la dernière discussion de design
> n'a mentionné que trois paliers (baronnie/comté/duché). À confirmer avant
> implémentation.

La règle devra encore préciser :

- les conditions de contiguïté exactes et le traitement des recouvrements
  entre deux fiefs candidats ;
- si un fief peut s'agrandir a posteriori en rachetant des territoires
  occupés adjacents à un fief existant, et à quel coût ;
- le sort d'un moulin ou d'un village occupé par un autre joueur à l'intérieur
  d'un fief nouvellement constitué.

Lorsqu'un fief est créé, son château capitale devient une cité et apporte
`+2` en défense.

Un joueur peut détenir plusieurs fiefs simultanément ; chacun rapporte ses
propres revenus (voir [economie.md](economie.md)) et son propre point de
victoire.

## Taxe seigneuriale

Une carte de taxe (jouée depuis le deck d'ordres spéciaux, voir
[ordres-speciaux.md](ordres-speciaux.md)) permet à un seigneur ou à un roi de
prélever un revenu supplémentaire sur un fief constitué :

- le **seigneur** (titulaire du fief) double le revenu de territoire de son
  propre fief pour ce tour ;
- le **roi** peut taxer n'importe quel fief constitué, mais seulement celui
  qui n'est pas déjà taxé par son seigneur ce tour-là (priorité au titulaire
  local) ; le supplément est alors détourné vers la capitale du roi au lieu de
  la capitale du fief.

Le détail du calcul (montant par territoire, avec et sans village) est défini
dans [economie.md](economie.md). Les cas de conflit entre plusieurs
prétendants au titre royal restent à trancher dans l'issue du milestone
[Politique royale](politique.md).

## Points et victoire

**Dépend aussi de :** [Succession](succession.md), pour l'application des
mariages et alliances au score.

Ce système remplace intégralement le score et la condition de victoire du
cœur v1 (GDD §9 : territoire, village, moulin, château, noble, troupe,
ressource). Le remplacement prend effet lorsque les milestones **Titres &
Victoire** et **Succession & Couronnement** sont tous deux livrés ; `gdd.md`
§9 est alors réécrit en conséquence, comme le prévoit la section « Évolution
du document » du GDD.

### Score de titres

| Titre | Points |
|---|---:|
| Baronnie | 1 |
| Comté | 2 |
| Duché | 3 |
| Cardinal | 2 |
| Pape | 5 |
| Roi | 5 |

> À trancher : valeurs indicatives, à valider en table de jeu avec les
> milestones Religieux et Politique royale pour équilibrer titres
> religieux/royaux face aux fiefs séculiers.

Le score d'un joueur est la somme des titres qu'il détient, ajustée par ses
mariages : voir [succession.md § Mariages et alliances](succession.md#mariages-et-alliances)
pour le calcul du poids d'alliance et des catégories tête/mixte/secondaire.

### Seuil de victoire et fin de partie

Une partie se termine immédiatement dès qu'un joueur franchit un seuil de
suprématie sur son score de titres. Deux seuils distincts existent, fixés
dans `assets/balance.yaml` en fonction du nombre de joueurs :

- **seuil solo** : score individuel requis pour une victoire majeure sans
  alliance ;
- **seuil d'alliance** : score combiné (les deux époux d'une tête active,
  voir succession.md) requis pour une victoire majeure commune. Le seuil
  d'alliance est strictement supérieur au seuil solo — une victoire à deux
  doit coûter plus cher que réussir seul, pas seulement cumuler deux scores
  plus faciles à atteindre séparément.

**Un joueur qui a une tête active ne peut jamais gagner seul**, même si son
score individuel atteint ou dépasse le seuil solo : tant qu'une tête est
active, seul le score combiné contre le seuil d'alliance est évalué pour lui.
Un joueur sans tête active (aucun mariage tête, ou toutes ses têtes sont
retombées faute d'être la mieux classée — ce qui ne devrait pas arriver
puisqu'une tête existante est toujours active pour son porteur, sauf
décès) reste évalué contre le seuil solo.

Si aucun seuil n'est atteint à la durée maximale de la partie (1 à 50 années,
GDD §2), la partie se termine sur le score de titres le plus élevé à cet
instant, selon les mêmes règles de seuil et d'alliance.

> À trancher : formule exacte des deux seuils (fixes vs proportionnels au
> nombre de joueurs, écart minimal entre seuil solo et seuil d'alliance), à
> arrêter dans l'issue de milestone dédiée au calibrage.

### Victoire majeure, victoire mineure, échec

- **Victoire majeure** : un joueur sans tête active qui franchit le seuil
  solo est déclaré vainqueur majeur seul. Un joueur avec tête active ne peut
  être vainqueur majeur qu'avec son conjoint, et seulement si leur score
  combiné franchit le seuil d'alliance ; les deux époux sont alors vainqueurs
  à égalité, sans hiérarchie entre eux.
- **Victoire mineure** : parmi tous les joueurs reliés au vainqueur majeur
  par une chaîne de mariages (tête inactive, mixte ou secondaire, y compris
  transitive à travers plusieurs maisons), seul celui dont le lien a le
  poids d'alliance le plus élevé obtient une victoire mineure. Les autres
  membres de la chaîne n'obtiennent rien de cette victoire.
- **Échec** : tout joueur restant, éliminé ou non, qui n'obtient ni victoire
  majeure ni victoire mineure.

Une égalité stricte de score entre deux joueurs ou alliances non mariés ne
désigne aucun vainqueur officiel, comme au GDD §9.

### Titres de courtoisie

Le conjoint d'un titulaire de fief (baron, comte, duc) ou de couronne (roi)
porte un titre de courtoisie assorti — baronne, comtesse, duchesse, reine —
quel que soit son sexe et quelle que soit la catégorie du mariage (tête,
mixte ou secondaire). Ce titre est **strictement d'affichage** : il n'entre
dans aucun calcul de score, de poids d'alliance, de vote ou de rang de
succession, et ne confère aucun accès aux titres réservés aux hommes
(évêque, cardinal, pape, roi lui-même). Il suit le titulaire réel du fief ou
de la couronne et change ou disparaît avec lui (remariage, mort, perte du
titre).

### Lisibilité et simulateur

Le calcul combine poids d'alliance, tête active, catégorie de mariage et deux
seuils distincts : il n'est pas raisonnable de demander aux joueurs de le
recalculer de tête. Le front doit exposer un simulateur — à la manière de
`POST /api/games/{id}/orders/preview` pour les ordres — qui projette, à la
demande, le score de titres et le statut de victoire (majeure/mineure/échec)
d'un joueur pour un état hypothétique (avant de conclure un mariage, après un
Claim, etc.), sans engager l'action. Voir l'issue dédiée dans le milestone.
