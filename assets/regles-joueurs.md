# Règles du jeu — Crown & Borough

**Version sociable (joueurs humains).** Ce document décrit les règles
actuellement actives dans le serveur de jeu. Les valeurs chiffrées proviennent
d'`assets/balance.json`, qui reste la source des nombres à jouer ; en cas de
divergence, le moteur fait foi.

---

## 1. Aperçu

Crown & Borough est un jeu de stratégie médiévale par tours, sur une carte de
territoires reliés par un graphe. Chaque joueur programme des **chaînes
d'ordres** pour ses armées ; toutes les chaînes sont résolues **simultanément**.

Les deux piliers de la tension :

- la résolution simultanée des intentions, des soutiens et des combats ;
- la logistique exponentielle : les grandes concentrations de troupes sont
  coûteuses et vulnérables au ravitaillement.

Une partie accepte de **2 à 16 joueurs**. Le serveur v1 est une session
« hotseat » : la carte et les données chiffrées (propriétaires, tailles
d'armées, stocks, infrastructures, nobles) sont visibles par tous.

Chaque joueur commence sur un territoire distinct : un **château** y est
construit gratuitement (il devient la **capitale**), avec 10 R de stock, une
armée d'une troupe et un noble libre.

### Inspirations

Le projet s'inspire notamment de [Fief](https://boardgamegeek.com/boardgame/107704/fief)
pour son cadre féodal et ses enjeux territoriaux, ainsi que de
[Diplomacy](https://boardgamegeek.com/boardgame/483/diplomacy) pour la
programmation simultanée des ordres, les soutiens et la résolution des
affrontements. Ces jeux sont des inspirations de conception, pas des sources
de règles applicables à Crown & Borough.

---

## 2. Cycle du jeu et saisons

Une année comprend **quatre tours** : printemps, été, automne et **hiver**. Le
compteur de tour progresse d'une unité à chaque saison, hiver compris (un
nouveau printemps après l'hiver).

### Tours d'action (printemps, été, automne)

1. Chaque joueur prépare et soumet ses chaînes d'ordres pour ses nobles libres
   et ses armées.
2. Le moteur vérifie les soumissions : une erreur de syntaxe ou de réception
   empêche la résolution de la soumission concernée, sans modifier l'état.
3. Le moteur résout simultanément le ravitaillement, les intentions, les
   soutiens, les combats, les déplacements, les retraites, les jonctions, les
   dispersions et la progression des chaînes.
4. Le contrôle territorial, la position des nobles et les événements sont mis à
   jour, puis un **rapport de tour** est produit.

### Phase d'hiver

L'hiver est une **trêve de gestion** : aucune chaîne d'action, aucun mouvement,
aucun combat et aucun ravitaillement. Le joueur soumet une liste
d'investissements directs, traités dans l'ordre saisi (voir section 5).

---

## 3. Écrire une chaîne

Une chaîne est composée du **trigramme du noble émetteur** (ligne d'en-tête)
puis d'**une ligne par ordre**.

> Dans l'IHM web, l'en-tête du noble est **ajouté automatiquement avant
> l'envoi** : vous écrivez uniquement les lignes d'ordres.

Chaque ligne d'ordre est de la forme `POSITION SYMBOLE [cibles...]`. Les
commentaires commencent par `#`, les lignes vides et la casse sont normalisées
par le parser.

Exemple de chaîne complète :

```text
LOU              # en-tête : noble émetteur (ajouté par l'IHM dans le web)
T01 A T02        # attaquer T02 depuis T01
T02 S T03 - T04  # soutien offensif
T05 J T06        # jonction (doit être le dernier ordre)
```

### Liaison des ordres

- **single** : une ligne sans parenthèses. La chaîne s'arrête au premier échec,
  et le suffixe est abandonné.
- **loop** : la ligne entière est entre parenthèses, `(…)`. L'ordre est retenté
  à chaque résolution jusqu'à sa réussite ; un maintien en loop met l'armée en
  veille. Une erreur mécaniquement impossible casse toujours la chaîne.

La non-adjacence d'un ordre est contrôlée lors de son exécution : les ordres
antérieurs restent valides et le suffixe est abandonné.

### Réception

- La chaîne est attachée **immédiatement et atomiquement** à l'armée présente
  sur la position de son premier ordre ; elle remplace la chaîne précédente de
  cette armée.
- Un noble libre ou otage n'émet qu'**une seule chaîne par tour**. Une chaîne
  ciblant une armée qui ne lui appartient pas, un noble au cachot ou un noble
  ayant déjà émis est rejetée.
- Si **plusieurs chaînes ciblent la même armée au même tour**, leur réception
  concurrente est invalidée : aucune n'est reçue et l'armée ne reçoit pas de
  nouvelle chaîne pour ce tour.
- Une armée sans chaîne est **Sans Ordre** : elle ne reçoit aucune action
  automatique.

---

## 4. Aide-mémoire des ordres

Les ordres ci-dessous sont disponibles au printemps, en été et en automne.
`XXX`, `YYY`, `ZZZ` sont des trigrammes de territoires ; `NNN` un trigramme de
noble. Aucun ne coûte de ressource en saison d'action.

| Symbole | Syntaxe | Effet |
|---|---|---|
| `A` | `XXX A YYY` | Attaque ou déplacement vers `YYY` adjacente. |
| `S` | `XXX S YYY` | Soutien défensif de l'armée qui tient `YYY`. |
| `S` | `XXX S YYY - ZZZ` | Soutien offensif de l'attaque de `YYY` vers `ZZZ`. |
| `H` | `H XXX` | Maintien sur `XXX`. |
| `J` | `XXX J YYY` | Jonction pacifique vers `YYY` adjacente ; **doit être le dernier ordre**. |
| `P` | `P XXX` | Pillage de l'infrastructure de la case occupée. |
| `D` | `XXX D DEST1 DEST2 ...` | Dispersion pacifique : les destinations sont traitées dans leur ordre d'apparition, peuvent se répéter et les troupes arrivant sur une même case sont empilées. |

### Attaque (`A`) et jonction (`J`)

`YYY` doit être **adjacent** à `XXX` par une frontière franchissable.
L'armée entière se déplace vers `YYY`. Une attaque peut y combattre une armée
ennemie ; la jonction ne combat pas et est repoussée si la destination est
contestée. La jonction doit être le dernier ordre de la chaîne.

### Soutien (`S`)

Un soutien renforce une armée de **n'importe quelle nationalité** :

- **défensif** (`XXX S YYY`) : renforce l'armée qui tient `YYY`, si `YYY` est
  adjacent à `XXX` (on ne se soutient pas soi-même) ;
- **offensif** (`XXX S YYY - ZZZ`) : renforce l'attaque de `YYY` vers `ZZZ`.

Il ne compte que si l'armée soutenue accomplit l'action annoncée. Une attaque
venue d'une case différente de la cible soutenue peut **couper** un soutien.

### Maintien (`H`) et pillage (`P`)

`H XXX` : l'armée reste sur place et peut recevoir un soutien défensif.
`P XXX` : détruit l'infrastructure de la case occupée ; un bonus de pillage
(2 R) est crédité à la source alliée la plus proche et peut réduire une famine.

### Dispersion (`D`)

`XXX D DEST1 DEST2 ...` traite les destinations dans leur ordre d'apparition,
avec au plus une troupe par destination :

- une destination est adjacente à `XXX` ou égale à `XXX` ; les destinations
  peuvent se répéter ;
- une destination occupée, combattue ou sans troupe disponible ne consomme pas
  de troupe ; une destination suivante peut néanmoins recevoir une troupe ;
- les troupes qui ne peuvent pas être envoyées restent sur la case d'origine ;
  une liste plus courte que l'armée laisse donc un résidu sur place ;
- les troupes arrivées sur une même destination sont empilées dans une seule
  armée ;
- les nobles explicitement affectés suivent le groupe produit : `*` affecte
  tous les nobles restants, `*NNN` affecte le noble `NNN` ; les nobles non
  mentionnés restent à l'origine tant qu'une troupe y demeure ;
- si toutes les troupes quittent l'origine et qu'un noble présent n'a pas de
  groupe produit, l'ordre est invalide à l'exécution ;
- en `single`, les destinations non traitées produisent une dispersion
  partielle et la chaîne progresse ; en `loop`, le résidu retente jusqu'à
  l'arrivée d'une armée sur chaque destination ; si l'armée est épuisée avant
  d'avoir traité toutes les destinations, l'ordre est invalide.

Exemples :

```text
BRI D ATL ATL              # deux troupes empilées dans l'armée arrivée à ATL
BRI D ATL                  # une troupe vers ATL, le résidu reste sur BRI
BRI D ATL*HUG NOR          # HUG vers ATL, l'autre unité vers NOR
(BRI D ATL NOR)            # dispersion en boucle
```

---

## 5. Ordres d'hiver

L'hiver n'accepte **aucune chaîne ni mouvement** : uniquement des
investissements directs, une ligne par ordre, appliqués dans l'ordre saisi.

| Investissement | Syntaxe | Condition | Coût (R) |
|---|---|---|---|
| Recruter un noble | `R N XXX` | `XXX` contrôlé, avec un château ou un village et une armée du joueur | 2 |
| Recruter une troupe | `R T XXX` | `XXX` contrôlé, et un noble libre du joueur sur `XXX` ou adjacent | 1 |
| Construire ou améliorer un moulin | `C M XXX` | `XXX` contrôlé ; moulin sur ou adjacent à un château ou un village | 3 |
| Construire un château | `C C XXX` | `XXX` contrôlé | 10 |
| Construire un dépôt de vivres | `C D XXX` | `XXX` contrôlé | 3 |
| Désigner une capitale | `E C XXX` | un château contrôlé sur `XXX` | 0 |
| Placer un noble en otage | `O N NNN` | `NNN` est un prisonnier adverse détenu par le joueur | 0 |
| Placer un noble au donjon | `P N NNN` | `NNN` est un prisonnier adverse détenu par le joueur | 0 |
| Libérer un noble | `L N NNN` | `NNN` est détenu par le joueur ; la capitale de son propriétaire contient une armée de celui-ci | 0 |

### Otage et donjon

Les ordres `O N NNN` et `P N NNN` ciblent un noble **prisonnier adverse** détenu
sur la case d'une armée du joueur. `O` le place en statut `hostage` (otage) et
`P` en statut `dungeon` (donjon). La capture produit par défaut le statut
`hostage`. Un noble otage peut émettre une nouvelle chaîne tant qu'il est
détenu ; un noble au cachot ne le peut pas. Les ordres peuvent faire passer un
prisonnier d'un statut à l'autre.

Les investissements qui ciblent un territoire exigent le **contrôle de ce
territoire**. Une
construction remplace la structure existante uniquement quand la règle le
prévoit : un **château construit sur un village remplace le village** et
conserve le stock de la case. Un moulin seul (orphelin) ne produit rien.

**Paiement** : le coût est prélevé d'abord sur le stock de la case ciblée, puis
sur la source contrôlée la plus proche ; si la réserve totale est insuffisante,
**aucun paiement partiel** n'est effectué et l'investissement est rejeté
(signalé dans le rapport, coût non perdu).

**Fin de l'hiver** :

- chaque stock restant est conservé à hauteur de `ceil(stock / 2)` ;
- les stocks hors capitale sont rapatriés vers la capitale, en laissant au
  maximum **1 R par village** et **2 R par château** ;
- sans capitale, les stocks restent sur place.

---

## 6. Armées, combats et logistique

### Armées et force

Une armée est l'unique entité de force d'un territoire : elle porte un
propriétaire et une taille en troupes. Toutes ses troupes partagent la même
chaîne ; il n'existe pas d'ordres mixtes au sein d'une armée.

- la force d'une attaque est la **taille** de l'armée attaquante ;
- la force d'un soutien est la taille de l'armée soutenante ;
- un château apporte un bonus défensif fixe de **+1**, même sans armée ;
- la plus haute force **strictement unique** gagne ; une égalité au sommet
  produit un **statu quo**, y compris sur une case vide ;
- une armée délogée perd son déplacement et doit **battre en retraite** ;
- une retraite part en bloc vers une case adjacente libre, non combattue
  pendant le tour et différente de l'origine de l'attaquant (départage par
  trigramme croissant ; deux armées sans alternative sur la même case sont
  détruites).

Le contrôle d'un territoire suit l'armée qui s'y arrête ; un contrôle acquis
reste acquis après le départ de l'armée, jusqu'à l'arrêt d'une armée ennemie.

### Ravitaillement exponentiel

Une armée de `N` troupes demande :

```text
coût = 2^(N - 1)  rations
```

| Taille | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Coût en rations | 1 | 2 | 4 | 8 | 16 |

La production vivrière de la case est distribuée aux armées présentes, toutes
nationalités confondues, au plus **une ration par armée** et en commençant par
la plus grosse. Le reste constitue la demande à ravitailler.

**Production vivrière de la case** : 1 ration en plaine, forêt ou colline ;
0 ration en montagne ou marécage ; **+2 rations** supplémentaires si la case
porte un château ou un village.

**Sources de ravitaillement** : les **châteaux et villages contrôlés**. Un
château ou un village produit **1 R stockable par tour**. Le flux traverse les
cases alliées ou neutres et s'arrête devant une case ennemie. La portée de base
est de **3 cases** ; chaque dépôt de vivres contrôlé rencontré sur le trajet
ajoute **2 cases**. Un village neutre conserve son stock, inaccessible au joueur
avant capture.

### Stocks et famine

En cas de déficit :

1. les stocks des châteaux et villages contrôlés sont épuisés (du plus petit au
   plus grand, trigramme territorial en départage) ;
2. les armées restantes passent en **famine**, en commençant par les plus
   éloignées de leur source, puis les plus grosses, puis le trigramme
   décroissant.

Une armée en famine **combat et se défend à force 0** pour le tour. Si elle se
trouve sur une infrastructure, elle la **pille automatiquement** ; le bonus de
pillage, diminué de sa demande résiduelle, peut la sortir de famine.

L'endpoint `GET /api/supply?territory=XXX` permet de prévisualiser le
ravitaillement d'une armée ou la zone atteinte depuis une source contrôlée
(uniquement hors hiver).

### Infrastructures

Une case ne porte qu'**une seule infrastructure**.

| Infrastructure | Condition | Effet v1 | Coût |
|---|---|---|---|
| Moulin | Sur ou adjacent à un château ou un village | +1 R stockable par niveau au point de production associé | 3 |
| Dépôt de vivres | Aucune | +2 cases de portée de ravitaillement lorsqu'il est contrôlé | 3 |
| Château | Aucune | +1 défense, +2 rations, produit 1 R stockable par tour, ancre de ravitaillement | 10 |
| Village | Généré neutre, **non constructible** | +2 rations, produit 1 R stockable par tour, ancre après capture | — |

---

## 7. Nobles : capture, déplacement et capacité

Les nobles **chevauchent les armées** : ils suivent les déplacements, les
attaques, les jonctions, les dispersions et les retraites. Un noble ne compte ni
dans la force, ni dans le ravitaillement, ni dans les pertes d'un combat. Un
noble peut rester seul sur une case après la perte de son armée.

**Capacité de commandement** :

- un noble **libre ou otage** émet au plus **une chaîne par tour** (nouvelle
  chaîne = nouveau tour) ;
- un noble **au cachot** (`dungeon`) ne peut pas émettre de nouvelle chaîne ;
- la chaîne s'applique à l'armée entière, dont le noble émetteur est le porteur.

> Il n'existe pas, dans cette version, de limite de **nobles portés par une
> armée** : une armée transporte tous les nobles présents sur sa case.

**Capture** : lorsqu'une armée portant des nobles est **détruite** sur une case
occupée par une armée ennemie, les nobles qu'elle portait sont capturés et
deviennent par défaut `hostage`. Un noble otage peut continuer à émettre une
chaîne ; seul son passage au cachot lui retire cette capacité. Le joueur qui le
détient aura accès au détail de ces chaînes dans les parties en ligne.

**Libération** : pendant l'hiver, `L N NNN` est émis par le joueur qui détient
le prisonnier, et non par son propriétaire. Si la capitale du propriétaire
existe et contient une armée de celui-ci, le noble réapparaît **libre dans cette
capitale** ; sinon l'ordre est rejeté.

Un joueur qui ne possède aucun noble libre ou otage apte à émettre n'a pas à
soumettre de chaînes pendant une saison d'action.

Les nobles affectés lors d'une **dispersion** doivent tous être répartis entre
les destinations (`*` ou `*NNN`), voir section 4.

---

## 8. Victoire

La règle de victoire prévue (documentée dans les spécifications) :

> Un joueur est **éliminé** lorsqu'il ne contrôle plus aucun territoire et ne
> possède plus aucune armée. Les nobles seuls ne maintiennent pas un joueur en
> lice. Le **dernier joueur vivant gagne** la partie.

**État actuel du serveur** : cette règle de fin de partie **n'est pas encore
appliquée par le moteur**. Les parties sont donc ouvertes : le cycle des saisons
et la résolution continuent tant que les joueurs soumettent des ordres. La
détection de l'élimination et la condition de victoire seront activées dans une
version ultérieure.
