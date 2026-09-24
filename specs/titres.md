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

- chaque fief rapporte 1 point, quelle que soit sa taille ;
- chaque portion de 5 territoires contrôlés rapporte 1 point supplémentaire ;
- une partie peut se terminer à la durée prévue en tours ou lorsqu'un joueur
  atteint un seuil de suprématie.

Le choix entre durée, seuil fixe et seuil dépendant du nombre de joueurs reste à
arrêter dans l'issue du milestone.
