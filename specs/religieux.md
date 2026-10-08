# Religieux

**Milestone lié :** [Religieux](https://github.com/fogfactory/crown-and-borough/milestone/5)

**Dépend de :** [Titres & Victoire](titres.md) pour les lieux et les points,
et de [Cartographie](cartographie.md) pour le découpage des évêchés ; le flux
de ressource de la dîme dépend de [Économie et prospérité](economie.md). Les conditions d'éligibilité reposent sur le sexe
([#245](https://github.com/fogfactory/crown-and-borough/issues/245)) et le mariage
([#288](https://github.com/fogfactory/crown-and-borough/issues/288)) du noble.

## Évêchés et évêques

La carte est divisée en `N + 1` évêchés de taille approximativement égale.
Le découpage réutilise directement les régions déjà générées pour les
calamités et les cartes bonus (`internal/engine/mapgen/regions.go`, une
région par chef-lieu seed, couverture totale et connexité garanties) :
même cardinalité `N + 1`, mêmes garanties de connexité. Ce découpage est déjà
implémenté et n'a pas besoin d'un nouveau travail de cartographie ; seule son
exposition en tant qu'évêché (nommage, contrat) relève de ce milestone. Un
territoire appartient à exactement une région/évêché, indépendamment de son
appartenance ou non à un fief séculier — les deux découpages sont disjoints
dans leur origine (fief : dynamique, acheté ; évêché : statique, fixé à la
génération de la carte). Le village dédié de chaque territoire de départ est
un lieu-dit de son évêché comme n'importe quel autre village ; seul le
chef-lieu sert de seed à la région.

### Déclenchement de l'élection

Une élection d'évêque est organisée dès que chaque lieu-dit de l'évêché est
soit **contrôlé** (fief constitué), soit **occupé** (statut positionnel) par
un joueur — la condition la plus permissive des deux statuts définis dans
[titres.md](titres.md). Un lieu-dit neutre (village non capturé) bloque le
déclenchement.

Tant que cette condition reste vraie, l'élection est **réévaluée à chaque
tour** : un évêché sans évêque (jamais élu, ou vacant après une égalité, une
mort ou une excommunication) retente automatiquement l'élection à chaque
tour tant que tous ses lieux-dits restent occupés ou contrôlés, sans ordre
dédié pour la relancer.

### Candidature

Tous les ordres d'une élection (candidature, vote) se saisissent dans la
feuille d'ordres d'hiver ; les joueurs s'accordent hors jeu. Un noble est
**éligible** s'il est un homme célibataire, ni capturé, ni excommunié ; pour
l'élection épiscopale, il n'est pas non plus déjà évêque. Aucune présence
physique dans l'évêché n'est requise.

Syntaxe (une ligne par ordre, sans coût) :

- `K E NNN BBB` : candidature du noble `NNN` du joueur à l'évêché dont le
  village seed est `BBB` ;
- `K P NNN` : candidature du noble `NNN`, cardinal, au conclave ;
- `V E NNN BBB` : vote du joueur pour le candidat `NNN` à l'évêché `BBB` ;
- `V P NNN` : vote du joueur pour le candidat `NNN` au conclave.

Un joueur ne vote que pour un candidat déclaré à la même élection, qu'il soit
à lui ou à un autre joueur. Le moteur regroupe les candidatures par élection,
puis les votes par élection et par candidat. Pour une même élection, un joueur
peut saisir plusieurs candidatures ou plusieurs votes : le **premier ordre
valide** de la feuille est retenu, les suivants sont ignorés. Les élections se
résolvent dans un ordre déterministe : évêchés par ordre stable (identifiant de
région), puis conclave.

### Votes

Le total de voix d'un joueur dans une élection épiscopale est un cumul global,
indépendant de la présence locale :

- 1 voix par lieu-dit contrôlé **ou** occupé dans l'évêché concerné ;
- 1 voix par évêque que le joueur possède, où qu'il se trouve ;
- 2 voix par cardinal que le joueur possède ;
- 3 voix si un des nobles du joueur est pape.

Les titres se cumulent sur un même noble, façon *Fief* : un cardinal est
forcément évêque et garde son évêché ; le pape est évêque ou cardinal et garde
ses titres. Pour les voix, seul le **titre le plus haut** de chaque noble
compte (pape 3, cardinal 2, évêque 1). Un noble excommunié ou au cachot ne
vote pas (voir « Fin de titre »).

Comme pour les autres titres, le pape est un noble, pas le joueur lui-même :
ce bonus s'attache au joueur propriétaire de ce noble. Ces bonus s'additionnent
aux voix territoriales même si le joueur ne contrôle ni n'occupe aucun
lieu-dit de l'évêché concerné.

Le candidat avec la plus haute majorité relative gagne. En cas d'égalité au
sommet, ou sans candidat éligible, aucun vainqueur n'est désigné et l'évêché
reste vacant (voir « Déclenchement de l'élection »).

## Cardinaux et pape

### Nomination et achat d'un cardinal

Un cardinal est obtenu par la promotion d'un évêque déjà en place — jamais
directement depuis un noble libre. Deux voies, cumulables :

- une **carte de nomination**, piochée dans le deck spécial et jouable par
  n'importe quel joueur (pas seulement le pape), ciblant obligatoirement un
  de ses propres évêques ;
- un **achat direct en hiver**, contre R, ciblant également un évêque du
  joueur qui paie. Le coût est à fixer dans `assets/balance.yaml` au même
  titre que les autres investissements hivernaux.

Dans les deux cas, le nombre total de cardinaux en jeu est plafonné à
`N - 1` (`N` = nombre de joueurs) ; une nomination ou un achat qui
dépasserait ce plafond est rejeté sans effet (carte perdue ou R non prélevé,
selon la règle générale de rejet des ordres du GDD §2). Il n'existe pas de
mécanisme de bootstrap dédié : la nomination n'étant pas réservée au pape,
les deux premiers cardinaux nécessaires à la toute première élection papale
peuvent être obtenus normalement par n'importe quel joueur, avant même
qu'un pape existe.

### Élection papale

Dès que deux cardinaux ou plus sont en jeu, un conclave est organisé. Seuls
les cardinaux votent, à raison d'**une voix chacun**, quel que soit leur
joueur ; les candidats sont des cardinaux éligibles (`K P`). Le vote se fait
par `V P` : l'ordre est porté par un cardinal du joueur, qui exprime la voix
de chacun de ses cardinaux pour le candidat nommé (le premier ordre valide
d'un joueur est retenu). L'élection exige la **majorité absolue** des
cardinaux en jeu. Sans majorité absolue, le trône reste vacant et le conclave
est réévalué chaque tour tant que la condition (≥ 2 cardinaux) reste vraie,
selon le même principe que l'évêché vacant.

## Fin de titre

Un titre religieux (évêque, cardinal, pape) prend fin par mort ou
excommunication uniquement :

- **Mort** du noble titré : le titre est immédiatement vacant. L'élection
  correspondante (évêché ou conclave) est réévaluée dès le tour suivant si
  sa condition de déclenchement reste vraie.
- **Excommunication** (voir « Pouvoirs » ci-dessous) : un évêque ou cardinal
  excommunié **perd son titre définitivement**, même après la levée de
  l'excommunication ; le titre est immédiatement vacant, comme pour une
  mort. Tant qu'elle dure, l'excommunié ne vote pas et n'est pas candidat ;
  à la levée, il redevient éligible à un titre religieux. Le pape ne peut
  pas s'excommunier lui-même.
- **Capture** (`hostage` ou `dungeon`) : le titre est **conservé**, il n'y a
  jamais de vacance ni de nouvelle élection déclenchée par une capture. Un
  noble titré `hostage` conserve l'intégralité de ses voix et pouvoirs
  religieux (cohérent avec le fait qu'il peut déjà émettre des chaînes). Un
  noble titré `dungeon` conserve son titre mais ses voix et pouvoirs
  religieux sont **suspendus** tant qu'il reste au cachot ; ils sont
  restaurés automatiquement dès que son statut repasse à `hostage` ou qu'il
  est libéré.

## Pouvoirs

Les pouvoirs religieux v1 sont l'excommunication, le procès à deux cardinaux, l'enquête, la dîme, l'apaisement de révolte et la dissolution de mariage. Sauf mention contraire,
ils sont des ordres spéciaux résolus indépendamment des chaînes d'ordres, au
même titre que les autres cartes du deck spécial (voir
[ordres-speciaux.md](ordres-speciaux.md)).

### Dîme

Une carte de dîme (jouée depuis le deck d'ordres spéciaux, voir
[ordres-speciaux.md](ordres-speciaux.md)) détourne la production des moulins
d'un évêché vers la capitale du joueur qui la joue, au lieu du village ou du
château normalement bénéficiaire (voir le flux de ressource dans
[economie.md](economie.md)). Elle ne touche jamais le revenu de territoire des
fiefs, qui relève exclusivement de la taxe seigneuriale.

- l'**évêque** peut poser une dîme sur son propre évêché ;
- le **cardinal** peut la poser sur n'importe quel évêché, mais seulement celui
  qui n'est pas déjà tenu par son évêque ce tour-là (priorité au titulaire
  local, symétrique à la règle roi/seigneur de [titres.md](titres.md)) ;
- le **pape** peut la poser sur n'importe quel évêché, avec la même priorité
  au titulaire local (évêque, ou cardinal s'il a déjà posé une dîme ce tour).

Syntaxe : `P DI XXX` dans la soumission `special`, `XXX` étant le village
seed de l'évêché visé.

- **Plusieurs cardinaux sur un même évêché** : la production des moulins de
  l'évêché est répartie **équitablement** entre les joueurs dont un cardinal a
  posé une dîme dessus (un joueur compte une fois, même avec plusieurs
  cardinaux ou ordres). Le partage se fait en unités entières de ressource,
  moulin par moulin ; le **surnuméraire** (reste de la division) est laissé sur
  place, sur le moulin, selon le flux normal de [economie.md](economie.md).
- **Évêché sans évêque** : il reste taxable par un cardinal ou le pape ; la
  priorité du titulaire local ne s'applique qu'à un évêque en place.
- **Révolte** : comme la taxe seigneuriale ([titres.md](titres.md)), la dîme
  autorise la Révolte (voir [ordres-speciaux.md](ordres-speciaux.md)) sur tout
  territoire de l'évêché taxé, la saison où elle est jouée et la saison
  suivante.

### Excommunication

Le **pape uniquement** peut excommunier un noble, titré ou non, par un ordre
d'hiver gratuit :

- `X E NNN` : excommunie le noble `NNN` ;
- `X L NNN` : lève l'excommunication de `NNN`.

Limites : **1 excommunication par hiver**, et **1 excommunié à la fois par
joueur adverse** (excommunier un second noble du même joueur exige d'abord
d'en lever un). Le pape peut viser ses propres nobles, mais pas lui-même.

Effets : l'excommunié ne vote pas et n'est pas candidat ; un évêque ou
cardinal excommunié perd son titre définitivement (voir « Fin de titre »). À
la levée, le noble redevient éligible. L'excommunication n'a aucun effet sur
les titres séculiers (fief, royauté). La mort du pape met fin à toutes ses
excommunications.

Les excommunications d'office (Éon et Sorcière démasqués, voir
[dames.md](dames.md)) ne comptent pas dans les limites ci-dessus et ne sont pas
levables.

### Procès à deux cardinaux

Deux cardinaux **distincts** envoient chacun `J NNN` (procès sans carte, noble
ou dame `NNN`) le même hiver ; le pape seul ne suffit jamais, mais un pape qui
est aussi cardinal compte comme cardinal. Les deux cardinaux peuvent
appartenir au même joueur. La cible suit le périmètre de la carte de procès
([dames.md § Carte de procès](dames.md#carte-de-procès)) : procès direct pour
une dame éligible, excommunication préalable pour tout autre personnage. Le
jugement a lieu en toute fin de tour. Deux ordres sur des cibles différentes
ne s'additionnent pas : ils sont sans effet.

### Enquête

Un cardinal ou le pape peut jouer `Q NNN` en hiver : coût fixe
`religious.inquiry_cost` R (`assets/balance.yaml`), une enquête par cardinal ou
pape et par hiver. Elle cible un noble ou une dame de n'importe quel joueur et
révèle sa dignité cachée (Éon, Correspondante, Espionne, Sorcière) avec les
conséquences de [dames.md](dames.md). Elle est résolue après les
excommunications et avant le jugement des procès. Une cible sans dignité cachée
consomme le coût sans effet et sans information.

### Apaisement de révolte

Le **pape, les cardinaux et les évêques** peuvent jouer un ordre spécial
d'apaisement, ciblant un territoire. S'il porte une armée `NEUTRAL` créée par
la calamité de révolte (voir GDD §2, « Cartes bonus et calamités »), elle est
retirée immédiatement, rendant le territoire directement reprenable. Un
évêque ne peut cibler que son propre évêché ; le cardinal et le pape peuvent
cibler n'importe quel territoire, avec la même priorité au titulaire local
que pour la dîme.

### Dissolution de mariage

> **Dépendance #18 :** ce pouvoir est spécifié ici mais son implémentation
> est différée après [l'issue #18](https://github.com/fogfactory/crown-and-borough/issues/18),
> faute de modèle de mariage/alliance à ce jour.

Le **pape uniquement** peut jouer un ordre spécial dissolvant un mariage
existant entre deux nobles, quel que soit leur propriétaire. Les effets
exacts sur les alliances et la succession seront définis avec le modèle de
mariage de #18 ; cette section sera complétée à ce moment-là sans revenir sur
la restriction « pape uniquement » actée ici.
