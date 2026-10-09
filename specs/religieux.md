# Religieux

**Milestone lié :** [Religieux](https://github.com/fogfactory/crown-and-borough/milestone/5)

**Ordre de résolution :** l'ordre exact des ordres d'hiver religieux entre eux et
avec les autres ordres (mariage, enquête, procès, gestion) est fixé par
[hiver.md](hiver.md), qui prime en cas de divergence.

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
hiver** : un évêché sans évêque (jamais élu, ou vacant après une égalité, une
mort ou une excommunication) retente automatiquement l'élection à chaque
hiver tant que tous ses lieux-dits restent occupés ou contrôlés, sans ordre
dédié pour la relancer. Le registre des élections ouvertes est figé au début
de l'hiver : un siège rendu vacant pendant l'hiver n'est électable qu'à
l'hiver suivant ([hiver.md](hiver.md#principes)).

### Candidature

Tous les ordres d'une élection (candidature, vote) se saisissent dans la
feuille d'ordres d'hiver ; les joueurs s'accordent hors jeu. Un noble est
**éligible** s'il est un homme célibataire, ni capturé, ni excommunié ; pour
l'élection épiscopale, il n'est pas non plus déjà évêque. Aucune présence
physique dans l'évêché n'est requise.

Syntaxe (une ligne par ordre, sans coût) :

- `K E NNN BBB` : candidature du noble `NNN` du joueur à l'évêché dont le
  village seed est `BBB` ;
- `K P NNN` : candidature du noble `NNN`, évêque ou cardinal, au conclave ;
- `V E NNN BBB` : vote du joueur pour le candidat `NNN` à l'évêché `BBB` ;
- `V P NNN` : vote du joueur pour le candidat `NNN` au conclave.

Un joueur ne vote que pour un candidat déclaré à la même élection, qu'il soit
à lui ou à un autre joueur. Le moteur regroupe les candidatures par élection,
puis les votes par élection et par candidat. Pour une même élection, un joueur
peut saisir plusieurs candidatures ou plusieurs votes : le **premier ordre
valide** de la feuille est retenu, les suivants sont ignorés. Un même noble peut
être candidat à plusieurs élections, mais pas à un évêché après avoir été élu
à un autre le même hiver. Les élections se résolvent l'une après l'autre
(évêchés par identifiant de région, puis conclave), avec des voix lues sur un
instantané des titres. Un titre gagné pendant l'hiver par élection ou par
achat de cardinal n'est conféré qu'à l'investiture, après tous les décomptes
([hiver.md](hiver.md#ordre-de-résolution)) ; la carte de cardinal, comme toute
dignité, prend effet aussitôt.

### Votes

Le total de voix d'un joueur dans une élection épiscopale est un cumul global,
indépendant de la présence locale :

- 1 voix par lieu-dit contrôlé **ou** occupé dans l'évêché concerné, sauf le
  chef-lieu de l'évêché (le seed de la région), qui vaut 2 voix dans l'élection
  de son évêque ;
- 1 voix par évêque que le joueur possède, où qu'il se trouve ;
- 2 voix par cardinal que le joueur possède ;
- 3 voix si un des nobles du joueur est pape ;
- 1 voix par Abbesse du joueur (au cachot ou excommuniée, elle ne vote pas)
  dont l'abbaye est dans l'évêché concerné, dans l'élection de son seul
  évêque. Posée à l'étape 2 de l'hiver, elle vote dès l'étape 6.

Les titres se cumulent sur un même noble, façon *Fief* : un cardinal est
forcément évêque et garde son évêché ; le pape est évêque ou cardinal et garde
ses titres. Pour les voix, seul le **titre le plus haut** de chaque noble
compte (pape 3, cardinal 2, évêque 1). Un noble excommunié ou au cachot ne
vote pas (voir « Fin de titre »).

Comme pour les autres titres, le pape est un noble, pas le joueur lui-même :
ce bonus s'attache au joueur propriétaire de ce noble. Ces bonus s'additionnent
aux voix territoriales même si le joueur ne contrôle ni n'occupe aucun
lieu-dit de l'évêché concerné.

Le candidat avec la plus haute majorité relative gagne. Les bulletins
individuels restent privés ; le rapport publie le total par candidat. En cas d'égalité au
sommet, ou sans candidat éligible, aucun vainqueur n'est désigné et l'évêché
reste vacant (voir « Déclenchement de l'élection »).

## Cardinaux et pape

### Nomination et achat d'un cardinal

Un cardinal est obtenu par la promotion d'un évêque déjà en place — jamais
directement depuis un noble libre. Deux voies, cumulables, chacune avec son
propre plafond :

- une **carte de cardinal** (code `CAR`), une dignité du
  [deck de nobles](succession.md#deck-de-nobles), gratuite. Elle se joue comme
  toute carte de dignité (`D N NNN CAR`) mais exclusivement sur un de ses
  propres évêques. Le deck contient `1 + ⌊N / 3⌋` cartes de cardinal (`N` =
  nombre de joueurs : 1 en dessous de 3 joueurs, 2 en dessous de 6, etc. ;
  `religion.cardinal_card_base` et `religion.cardinal_card_players_per_extra`
  de `assets/balance.yaml`) : ce nombre est le plafond de cette voie, il n'y a
  pas d'autre limite que celle des cartes disponibles ;
- un **achat direct en hiver**, contre R, ciblant également un évêque du
  joueur qui paie (`N C NNN`). Le coût est `religion.cardinal_cost` dans
  `assets/balance.yaml` (8 R), prélevé sur les réserves de paiement du joueur
  depuis sa capitale. Les cardinaux achetés en jeu sont plafonnés à
  `1 + ⌊N / 6⌋` (1 en dessous de 6 joueurs, 2 en dessous de 12, etc. ;
  `religion.cardinal_purchase_base` et
  `religion.cardinal_purchase_players_per_extra`) ; l'ordre qui dépasserait ce
  plafond est rejeté sans prélèvement. Les achats déjà acceptés dans l'hiver
  comptent dans le plafond.

Dans les deux cas l'ordre se résout parmi les ordres de gestion. La **carte**,
comme toute carte de dignité, se joue à tout moment : le noble est cardinal
aussitôt, ses voix comptent dans les élections du même hiver et il compte dans
la majorité absolue du conclave. L'**achat** n'est conféré qu'à l'investiture :
le nouveau cardinal ne vote ni ne se présente au conclave du même hiver. Une
excommunication du même hiver, résolue avant, rend l'ordre caduc : l'achat
n'est pas prélevé, la carte reste en main. Un ordre
rejeté (noble qui n'est pas un évêque du joueur, déjà cardinal ou déjà promu
cet hiver) ne coûte rien et ne consomme pas la carte.

Les deux plafonds sont indépendants : à 3 joueurs, jusqu'à 2 cardinaux par
carte et 1 par achat, soit 3 au total, de quoi tenir un conclave. Il n'existe
pas de mécanisme de bootstrap dédié : n'importe quel joueur peut obtenir les
premiers cardinaux, avant même qu'un pape existe.

**Origine du titre.** Le cardinal obtenu par carte porte la dignité
« cardinal » (la carte repose sur lui, comme toute dignité) ; le cardinal acheté
n'en porte pas. Quand le titre prend fin (mort ou excommunication, voir « Fin de
titre ») :

- un cardinal **obtenu par carte** rend sa carte à la défausse du deck de nobles
  (comme toute dignité) : elle peut être repiochée et rejouée ;
- un cardinal **acheté** libère sa place sous le plafond d'achat : elle est de
  nouveau disponible à l'achat.

### Élection papale

Dès que deux cardinaux ou plus sont en jeu, un conclave est organisé. Seuls
les cardinaux votent, à raison d'**une voix chacun**, quel que soit leur
joueur ; les candidats sont des évêques ou cardinaux éligibles (`K P`), titrés avant
le début de l'hiver (cumul à la Fief : l'élu garde ses autres titres). Le vote se fait
par `V P` : l'ordre est porté par un cardinal du joueur, qui exprime la voix
de chacun de ses cardinaux pour le candidat nommé (le premier ordre valide
d'un joueur est retenu). L'élection exige la **majorité absolue** : strictement plus de la moitié de
tous les cardinaux en jeu, y compris ceux dont la voix est suspendue (cachot).
Le conclave n'est ouvert que si le trône est vacant. Sans majorité absolue, le trône reste vacant et le conclave
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

La dîme est l'une des deux issues de la carte **Impôts** du deck d'ordres
spéciaux (voir [ordres-speciaux.md](ordres-speciaux.md) et, pour la taxe,
[titres.md § Taxe seigneuriale](titres.md#taxe-seigneuriale)). Le joueur précise
le **noble émetteur** et la **cible** : lorsque l'émetteur est un évêque, un
cardinal ou le pape et que la cible est un évêché, l'impôt est une dîme. Elle
détourne la production des moulins de l'évêché vers la capitale du joueur qui
la joue, au lieu du village ou du château normalement bénéficiaire (voir le
flux de ressource dans [economie.md](economie.md)). Elle ne touche jamais le
revenu de territoire des fiefs, qui relève exclusivement de la taxe.

- l'**évêque** ne cible que son propre évêché ;
- le **cardinal** et le **pape** peuvent cibler n'importe quel évêché.

Syntaxe : `P DI HHH XXX` dans la soumission `special`, `HHH` étant le noble
émetteur et `XXX` le village seed de l'évêché visé. `P TX HHH XXX` sur un village
seed qui n'est la capitale d'aucun fief est aussi une dîme ; le code `DI` lève
l'ambiguïté lorsque `XXX` est à la fois la capitale d'un fief et le seed d'un
évêché : un même noble peut être seigneur et évêque et choisir, pour la même
carte, la taxe ou la dîme.

- **Priorité** : lorsque plusieurs dîmes visent le même évêché le même tour,
  l'**évêque** de cet évêché l'emporte sur les cardinaux et sur le pape ; les
  **cardinaux** l'emportent sur le pape. Le pape ne récupère donc que la dîme
  des évêchés sur lesquels aucune autre dîme n'a été posée. Les dîmes
  battues par une priorité sont consommées sans effet.
- **Plusieurs cardinaux sur un même évêché** (sans dîme de l'évêque) : la
  production des moulins de l'évêché est répartie **équitablement** entre les
  joueurs dont un cardinal a posé une dîme dessus (un joueur compte une fois,
  même avec plusieurs cardinaux ou ordres). Le partage se fait en unités
  entières de ressource, moulin par moulin ; le **surnuméraire** (reste de la
  division) est laissé sur place, sur le moulin, selon le flux normal de
  [economie.md](economie.md). Il n'y a pas de répartition entre les autres
  titres.
- **Évêché sans évêque** : il reste taxable par un cardinal ou le pape.
- **Un joueur, une part** : un même joueur ne perçoit qu'une dîme par évêché
  et par tour ; ses cartes supplémentaires sont consommées sans effet. Un
  cardinal ou le pape qui est aussi évêque de l'évêché visé compte comme
  l'évêque de cet évêché. Le titre suspendu (excommunié, donjon) ne permet pas
  de poser la dîme.
- **Révolte** : comme pour toute taxe, tout territoire de l'évêché ciblé est
  éligible à la Révolte (voir [ordres-speciaux.md](ordres-speciaux.md)), la
  saison où la dîme est jouée et la saison suivante.

### Excommunication

Le **pape uniquement** peut excommunier un noble, titré ou non, par un ordre
d'hiver gratuit :

- `X E NNN` : excommunie le noble `NNN` ;
- `X L NNN` : lève l'excommunication de `NNN`.

Elle se résout en première étape de l'hiver, avant les ordres de gestion
(achat de cardinal compris), les enquêtes et les élections. Limites : **1 excommunication par hiver**, et **1 excommunié à la fois par
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

Deux cardinaux **distincts** déposent chacun `J HHH NNN` (procès sans carte) le
même hiver : `HHH` est le cardinal qui porte le procès, `NNN` le noble ou la
dame jugé. Le pape seul ne suffit jamais, mais un pape qui est aussi cardinal
compte comme cardinal. Les deux cardinaux peuvent appartenir au même joueur
(deux lignes sur sa feuille) ou à deux joueurs (une ligne chacun). `HHH` doit
être un cardinal du joueur, au titre actif (ni excommunié ni au cachot) ; un
cardinal ne porte qu'un procès par hiver. Un ordre dont `HHH` n'est pas un
noble du joueur est rejeté (`noble_not_owned`), de même qu'un `HHH` sans titre
actif de cardinal (`not_cardinal`) ou déjà engagé (`trial_limit`). Un procès
n'a lieu que si un second cardinal dépose le même : un ordre isolé reste sans
effet. La cible suit le périmètre de la carte de procès
([dames.md § Carte de procès](dames.md#carte-de-procès)) : procès direct pour
une dame éligible, excommunication préalable pour tout autre personnage. Le
jugement a lieu en toute fin de tour, après les élections et la fin
d'hiver ([hiver.md](hiver.md#ordre-de-résolution)) ; un noble excommunié le
même hiver est jugeable. Deux ordres sur des cibles différentes
ne s'additionnent pas : ils sont sans effet.

### Enquête

Un cardinal ou le pape `HHH` d'un joueur, au titre actif (ni excommunié ni au
cachot), peut jouer `Q HHH NNN` en hiver : une enquête par cardinal ou pape et
par hiver, `NNN` étant le noble enquêté. Elle cible un noble ou une dame de n'importe quel joueur
et révèle sa dignité cachée (Éon, Correspondante, Espionne, Empoisonneuse,
Sorcière) à tous les joueurs, avec les conséquences de [dames.md](dames.md).
La dignité reste portée par la dame ; seule sa visibilité change. Un ordre dont
`HHH` n'est pas un noble du joueur est rejeté (`noble_not_owned`), de même qu'un
`HHH` sans titre actif de cardinal ou de pape (`not_cardinal`) ou qui a déjà
enquêté l'hiver en cours (`inquiry_limit`). Le coût est payé par le joueur,
quel que soit `HHH`. Elle est résolue après les
excommunications et les ordres de gestion, avant les mariages, les élections
et le jugement des procès ([hiver.md](hiver.md#ordre-de-résolution)). Un Éon
ou une Sorcière démasqué est excommunié d'office dès l'enquête : il peut être
jugé le même hiver ; la Correspondante, l'Espionne et l'Empoisonneuse gardent
leurs bonus. Une cible sans dignité cachée, ou déjà révélée, consomme le coût
sans effet et sans information. Le coût est prélevé sur les réserves de la
capitale du joueur, sans prélèvement partiel (`insufficient_resources`).

Le coût, en R, est proportionnel au rang de la cible. Il se lit dans
`assets/balance.yaml` (bloc `religion.inquiry_cost`) :

| Titre de la cible | Coût |
|---|---|
| Baron | 2 |
| Comte | 3 |
| Marquis | 4 |
| Duc | 5 |
| Roi | 5 |
| Pape | 5 |
| Évêque | 3 |
| Cardinal | 4 |
| Époux ou épouse | coût du conjoint − 1 |
| Noble sans titre ni conjoint | 1 |

Les critères ne se cumulent pas : si plusieurs s'appliquent à la cible, on
retient le plus cher. Le coût « conjoint − 1 » se calcule sur le coût du
conjoint déterminé par ses propres titres (sans lui appliquer à son tour la
réduction de mariage).

### Apaisement de révolte

Un ecclésiastique — **pape, cardinal, évêque ou abbesse** — peut calmer une
révolte en retirant l'armée `NEUTRAL` née de la calamité de révolte (voir GDD
§2, « Cartes bonus et calamités ») d'un territoire. L'ordre ne consomme aucune
carte, se joue au printemps, en été ou en automne dans la soumission `special`
et s'applique avant les cartes de Révolte du même tour. `HHH` est le noble
ecclésiastique (un noble du joueur), `TER` un territoire portant une armée
`NEUTRAL` ; sans armée à calmer, l'ordre est sans effet. Un noble ne joue
qu'un apaisement par tour. Un titre suspendu (cachot) ou perdu
(excommunication) ne permet pas l'apaisement.

Deux voies :

- **`P AG HHH TER` — rite gratuit.** L'ecclésiastique agit dans la région où il
  se trouve (celle de l'armée qui le porte), pas nécessairement son évêché. Un d6 est lancé
  (`religion.appeasement_success_rolls` et `religion.appeasement_death_rolls`
  dans `assets/balance.yaml`) : 1 à 3, l'armée rebelle est retirée ; 4 ou 5,
  l'échec est sans conséquence ; 6, c'est un échec et l'ecclésiastique meurt
  (mort « martyr » : titres vacants, fiefs et prétentions réglés comme pour tout
  décès). L'abbesse n'a accès qu'à cette voie.
- **`P AP HHH TER` — apaisement payant, sans risque.** Un évêque sur
  son propre évêché, un cardinal ou le pape sur n'importe quel territoire paient
  `religion.appeasement_cost_base` puissance la taille de l'armée rebelle
  en R (2 pour une troupe, 4 pour deux, 8 pour trois, 16 pour quatre…), prélevés
  sur les stocks du joueur comme les investissements d'hiver. Sans les R,
  l'ordre est rejeté et rien n'est retiré. Le résultat est certain.

Aliases anglais : `FQ` (rite) et `PQ` (payant).

### Dissolution de mariage

> **Dépendance #18 :** ce pouvoir est spécifié ici mais son implémentation
> est différée après [l'issue #18](https://github.com/fogfactory/crown-and-borough/issues/18),
> faute de modèle de mariage/alliance à ce jour.

Le pape et un des époux (ou son propriétaire) soumettent chacun `X D NNN`
(syntaxe proposée) pour le même couple ; un seul ordre suffit si le pape
possède l'un des époux. La dissolution se résout avant la conclusion des
mariages du même hiver : les deux nobles redeviennent célibataires et peuvent
être remariés ou se présenter à une élection dès cet hiver ; aucune prétention
n'est retirée ([hiver.md](hiver.md#effets-croisés)).

Le **pape uniquement** peut jouer un ordre spécial dissolvant un mariage
existant entre deux nobles, quel que soit leur propriétaire. Les effets
exacts sur les alliances et la succession seront définis avec le modèle de
mariage de #18 ; cette section sera complétée à ce moment-là sans revenir sur
la restriction « pape uniquement » actée ici.
