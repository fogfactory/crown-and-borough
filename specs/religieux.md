# Religieux

**Milestone lié :** [Religieux](https://github.com/fogfactory/crown-and-borough/milestone/5)

**Dépend de :** [Titres & Victoire](titres.md) pour les lieux et les points,
et de [Cartographie](cartographie.md) pour le découpage des évêchés ; le flux
de ressource de la dîme dépend de [Économie et prospérité](economie.md). La
candidature complète à un titre religieux (célibataire, mâle) dépend du
modèle de noble enrichi par [l'issue #18](https://github.com/fogfactory/crown-and-borough/issues/18)
(succession, mariages, Claims) : voir « Candidature » ci-dessous.

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

Tout noble libre d'un joueur peut se porter candidat, sans contrainte de
présence physique dans l'évêché.

> **Dépendance #18 :** *Fief* réserve les titres religieux aux nobles mâles
> célibataires. Cette contrainte est actée pour Crown & Borough mais ne peut
> pas être implémentée avant que [l'issue #18](https://github.com/fogfactory/crown-and-borough/issues/18)
> introduise le sexe et le statut marital du noble. Ce milestone Religieux
> est donc séquencé après #18 pour ce point : jusqu'à l'arrivée de ce modèle,
> tout noble libre est éligible sans distinction.

### Votes

Le total de voix d'un joueur dans une élection (évêque ou pape) est un
cumul global, indépendant de la présence locale :

- 1 voix par lieu-dit contrôlé **ou** occupé dans l'évêché concerné ;
- 1 voix par évêque que le joueur possède, où qu'il se trouve ;
- 2 voix par cardinal que le joueur possède ;
- 3 voix si le joueur est pape.

Ces bonus s'additionnent aux voix territoriales même si le joueur ne
contrôle/occupe aucun lieu-dit de l'évêché concerné : un pape sans aucun
territoire local vote quand même avec ses 3 voix.

Le candidat avec la plus haute majorité relative gagne. En cas d'égalité au
sommet, aucun vainqueur n'est désigné et l'évêché reste vacant (voir
« Déclenchement de l'élection »).

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

Dès que deux cardinaux ou plus sont en jeu, une élection papale est
organisée parmi eux, avec les mêmes règles de vote que l'élection épiscopale
(voir « Votes » ci-dessus), mais à la **majorité absolue** plutôt que
relative. Une élection sans majorité absolue ne désigne aucun vainqueur ; le
trône reste vacant et l'élection est réévaluée chaque tour tant que la
condition (≥ 2 cardinaux) reste vraie, selon le même principe que
l'évêché vacant.

## Fin de titre

Un titre religieux (évêque, cardinal, pape) prend fin par mort ou
excommunication uniquement :

- **Mort** du noble titré : le titre est immédiatement vacant. L'élection
  correspondante (évêché ou conclave) est réévaluée dès le tour suivant si
  sa condition de déclenchement reste vraie.
- **Excommunication** (voir « Pouvoirs » ci-dessous) : le titre est
  immédiatement vacant, comme pour une mort, et le noble excommunié devient
  **définitivement inéligible** à tout titre religieux futur (évêque,
  cardinal ou pape), y compris après une éventuelle libération ou un
  changement de camp.
- **Capture** (`hostage` ou `dungeon`) : le titre est **conservé**, il n'y a
  jamais de vacance ni de nouvelle élection déclenchée par une capture. Un
  noble titré `hostage` conserve l'intégralité de ses voix et pouvoirs
  religieux (cohérent avec le fait qu'il peut déjà émettre des chaînes). Un
  noble titré `dungeon` conserve son titre mais ses voix et pouvoirs
  religieux sont **suspendus** tant qu'il reste au cachot ; ils sont
  restaurés automatiquement dès que son statut repasse à `hostage` ou qu'il
  est libéré.

## Pouvoirs

Les pouvoirs religieux v1 sont au nombre de quatre. Sauf mention contraire,
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

La règle devra préciser le cas où plusieurs cardinaux ciblent le même évêché le
même tour, et si un évêché sans évêque élu reste taxable par un cardinal ou le
pape.

> À trancher : comme la taxe seigneuriale ([titres.md](titres.md)), la dîme
> devrait sans doute autoriser la Révolte (voir
> [ordres-speciaux.md](ordres-speciaux.md)) sur tout territoire de l'évêché
> taxé, la saison où elle est jouée et la saison suivante.

### Excommunication

Le **pape uniquement** peut excommunier n'importe quel noble, titré ou non,
d'un ordre spécial dédié. L'excommunication d'un noble titré met fin à son
titre immédiatement (voir « Fin de titre ») et le rend définitivement
inéligible à tout titre religieux futur. Un noble excommunié qui n'est pas
titré ne subit que cette inéligibilité future ; l'excommunication n'a aucun
effet sur ses titres séculiers (fief, royauté), qui restent une couche
distincte des titres religieux.

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
