# Titres et victoire

**Milestone lié :** [Titres & Victoire](https://github.com/fogfactory/crown-and-borough/milestone/2)

**Dépend de :** la carte, le contrôle territorial et le ravitaillement du
socle actuel ; les flux de ressources dépendent de
[Économie et prospérité](economie.md) et les titres religieux de
[Religieux](religieux.md).


Les règles de contrôle, de fief et de taxe ci-dessous sont des décisions de
conception actées pour le milestone
[Économie & Fiefs](https://github.com/fogfactory/crown-and-borough/milestone/19),
pas encore implémentées. Chaque section renvoie à l'issue qui la livre ; les
points « à trancher » sont tranchés au début de cette issue. Aucune
compatibilité avec les parties existantes n'est requise (version majeure).

## Contrôle et occupation

Issue : [#196](https://github.com/fogfactory/crown-and-borough/issues/196).

Le socle actuel ne connaît qu'un statut territorial, positionnel : une case
reste au dernier joueur dont l'armée s'y est arrêtée. Les fiefs le rendent
transitif, sans introduire de second statut stocké :

- **contrôlé** : le statut porté par la case (`OwnerID`). Hors fief, il reste
  positionnel. Dans un fief, la case est contrôlée par le joueur qui détient
  le fief, même lorsqu'une armée adverse s'y arrête. Seule la prise de la
  capitale du fief transfère le fief et donc le contrôle de tous ses
  territoires (voir ci-dessous).
- **occupé** : une armée est présente sur la case. C'est une information
  dérivée, jamais stockée.

Un territoire contrôlé rapporte son revenu à la capitale du joueur, ou à la
capitale du fief s'il appartient à un fief, y compris lorsqu'il est occupé par
une armée adverse (voir [economie.md](economie.md#revenu-territorial)).

La capitale d'un joueur n'est pas un fief implicite. Hors fief, une capitale
sans armée reste contrôlée tant qu'aucune armée adverse ne s'y arrête.

> À trancher dans #196 : le traitement d'une case d'un fief occupée par une
> armée adverse (stock, pillage, investissements d'hiver, bonus de portée d'un
> dépôt). Voir [economie.md](economie.md#portée-de-ravitaillement).

## Constitution d'un fief

Issue : [#194](https://github.com/fogfactory/crown-and-borough/issues/194).

Un fief se constitue par un **ordre d'hiver** qui désigne un noble titulaire
et un groupe de territoires, dont le premier est la **capitale du fief**
(syntaxe proposée : `T F NNN XXX YYY ZZZ …`). Le groupe doit :

- compter au moins 3 territoires, contigus par des frontières franchissables ;
- être entièrement contrôlé par le joueur ;
- avoir un château sur sa capitale ;
- ne contenir aucun territoire appartenant déjà à un fief.

Tout rejet est explicite et ne prélève rien. Le coût vaut
`fief_cost_per_territory` (2 R) par territoire et se paie comme les autres
investissements d'hiver. Le titre dépend de la taille :

| Titre | Territoires | Coût par défaut |
|---|---:|---:|
| Baronnie | 3 | 6 R |
| Comté | 4 | 8 R |
| Marquisat | 5 | 10 R |
| Duché | 6 et plus | 2 R par territoire |

Le titre appartient au **noble** titulaire, qui doit être un noble libre du
joueur. Un joueur peut détenir plusieurs fiefs simultanément. Lorsqu'un fief
est créé, son château capitale devient une **cité** et apporte `+2` en
défense.

L'agrandissement d'un fief existant est différé : il n'est pas prévu dans ce
milestone.

> À trancher dans #194 : la syntaxe définitive des ordres de constitution et
> d'attribution ; le bonus de cité (+2 au total, recommandé, ou +2 en plus du
> +1 du château) ; un même noble peut-il porter plusieurs titres (recommandé :
> oui) ; le groupe peut-il contenir d'autres châteaux (recommandé : oui) ou
> une armée adverse (recommandé : non).

## Perte et vacance d'un fief

Issue : [#194](https://github.com/fogfactory/crown-and-borough/issues/194).

- **Capitale du fief prise** : lorsqu'un autre joueur prend le contrôle de la
  case de la capitale, le fief entier passe à ce joueur, **vacant**.
- **Mort du titulaire** (par exemple de la peste) : le fief reste au joueur
  qui le détient mais devient vacant.
- **Capture du titulaire** (otage ou donjon) : aucun effet sur le fief.
- **Fief vacant** : il continue d'exister et de produire. Un ordre d'hiver
  l'attribue à un noble libre du joueur qui le détient ; sans attribution à la
  fin de l'hiver, le fief est dissous et ses territoires redeviennent
  contrôlés hors fief.

> À trancher dans #194 : un fief vacant compte-t-il son point de victoire
> (recommandé : oui, jusqu'à sa dissolution).

## Taxe seigneuriale

Issue : [#189](https://github.com/fogfactory/crown-and-borough/issues/189).

Une carte de taxe (jouée depuis le deck d'ordres spéciaux, voir
[ordres-speciaux.md](ordres-speciaux.md)) permet au joueur qui détient un fief
de doubler, pour le tour, le revenu territorial de ce fief, village inclus.
L'ordre `P <KIND> XXX` cible la capitale du fief, par exception à la règle
« `TER` est le village seed d'une région », au printemps, en été ou en
automne. Il est rejeté si le joueur ne détient pas le fief. La production des
moulins n'est jamais touchée.

Le **roi** pourra taxer n'importe quel fief constitué, mais seulement celui
qui n'est pas déjà taxé par son seigneur ce tour-là (priorité au titulaire
local) ; le supplément est alors détourné vers la capitale du roi au lieu de
la capitale du fief. Cette taxe royale est suivie dans le milestone
[Politique royale](politique.md).

Le détail du calcul (montant par territoire, avec et sans village) est défini
dans [economie.md](economie.md).

> À trancher dans #189 : le code du kind ; la taxe d'un fief vacant
> (recommandé : autorisée pour le joueur qui le détient) ; deux cartes sur le
> même fief le même tour (recommandé : pas de cumul, seconde carte consommée
> sans effet).

## Points et victoire

- chaque fief rapporte 1 point, quelle que soit sa taille, en plus du barème
  de `gdd.md` §9 (livré par
  [#194](https://github.com/fogfactory/crown-and-borough/issues/194)) ;
- une partie peut se terminer à la durée prévue en tours ou lorsqu'un joueur
  atteint un seuil de suprématie.

Les autres évolutions du score (points par tranche de territoires contrôlés)
et le choix entre durée, seuil fixe et seuil dépendant du nombre de joueurs
restent à arrêter dans le milestone Titres & Victoire.
