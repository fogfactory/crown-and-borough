# Game Design Document : Crown & Borough (MVP)

## 1. Vision & Concept Cœur

Un jeu de stratégie asynchrone sur carte, croisant la tension politique de *Fief* et la résolution simultanée de *Diplomacy*.

La spécificité stratégique repose sur deux piliers système :

- **La latence de l'information et des ordres :** L'information voyage physiquement sur la carte.
- **Les contraintes logistiques exponentielles :** Concentration de troupes coûteuse et lignes de flux vulnérables.

---

## 2. Le Cœur de Loop & Saisons

Une année de jeu se compose de **4 tours** : trois tours d'action (**Printemps, Été, Automne**) et un tour de gestion (**Hiver**).

### Tours d'Action (Printemps, Été, Automne)

1. **Réception des rapports :** Le joueur prend connaissance de la carte selon l'ancienneté des messagers parvenus à ses Nobles (libres ou otages).
2. **Ordres & Diplomatie :** Négociations privées, programmation des chaînes d'ordres et des axes logistiques.
3. **Résolution serveur simultanée :**
   - *Ravitaillement & Carence :* Traçage des flux, consommation des ressources R, prélèvement sur les stocks en cas de déficit, basculement en famine si pénurie (une armée famélique combat à force 0 CE tour-ci).
   - *Propagation :* Avancée des messagers (ordres et rapports).
- *Exécution :* Application des ordres reçus ce tour-ci par les armées.
- *Combats & Retraites :* Résolution des affrontements et replis obligatoires.
- *Émission :* Départ des nouveaux rapports d'information (troupes, châteaux et villages contrôlés, tours de guet) vers le noble le plus proche (récepteur).

### Phase d'Hiver (Bilan, Conservations et Investissements)

- **Trêve hivernale :** Mouvements militaires gelés (aucune chaîne traitée, aucun combat, aucun ravitaillement).
- **Ordres d'Hiver :** le joueur soumet une liste d'ordres (même mécanique de soumission textuelle que les chaînes ; application **directe** — sans messager ni noble émetteur), une ligne = un ordre, traités dans l'ordre saisi. **Tout investissement exige le contrôle du territoire ciblé** (et la présence d'une troupe du joueur sur la case pour le recrutement d'un noble — un noble n'est jamais seul, cf. §6) :
  - `R N <territoire>` — recruter un **Noble** sur le territoire (nom = prénom tiré + "de \<nom du territoire\>", ex. "Jacques de Notombes")
  - `R T <territoire>` — recruter une **Troupe** sur le territoire
  - `C M <territoire>` — construire un **Moulin** (améliore le moulin existant)
  - `C C <territoire>` — construire un **Château**
  - `C R <territoire>` — construire un **Relais de Poste**
  - `C T <territoire>` — construire une **Tour de Guet**
  - `C D <territoire>` — construire un **Dépôt de Vivres**
  - `E C <territoire>` — désigner le château de ce territoire comme **Capitale**
  - `L N <code noble>` — libérer un **Noble prisonnier** du joueur (réapparaît libre à la capitale ; requis : une capitale désignée)
- **Coûts des investissements :**

  | Investissement | Coût (R) |
  |---|---|
  | Château | 10 |
  | Moulin | 3 |
  | Troupe | 1 |
  | Noble | 2 |
  | Relais de Poste | 2 |
  | Tour de Guet | 4 |
  | Dépôt de Vivres | 3 |
  | Libération d'un noble (`L N`) | 0 |

  *(Toutes les métriques d'équilibrage sont stockées dans `assets/balance.json`, facilement éditables.)*
- **Paiement de proche en proche :** le coût d'un investissement est prélevé d'abord sur le stock du territoire ciblé (s'il stocke), puis sur le château ou village contrôlé le plus proche, et ainsi de suite. Réserves totales insuffisantes → ordre rejeté (aucun prélèvement partiel).
- **Capitale :** la capitale d'un joueur n'est PAS un territoire fixe : c'est un **château qu'il considère comme sa capitale**. Par défaut, son **premier château** (le château de départ compte). Un ordre d'Hiver `E C <territoire>` désigne un autre de ses châteaux comme capitale. La capitale ne peut jamais être ennemie : la désignation exige le contrôle de la case. Si le château désigné est détruit, la désignation est perdue et le joueur n'a **plus de capitale** — pas de rapatriement entre-temps (les stocks restent en place) et `L N` est rejeté — jusqu'à ce qu'un ordre `E C` ou un nouveau château construit (capitale par défaut) la redésigne.
- **Conservation des stocks (Règle des 50 %) :** sur chaque château ou village, les ressources non consommées durant l'année subissent une perte hivernale après investissement :

  Stock Conservé au Printemps = ceil(R_restant / 2)

  *(Arrondi à l'entier supérieur sur chaque château ou village).*

- **Rapatriement des stocks :** les stocks sont automatiquement rapatriés vers la case de la capitale du joueur, en laissant au maximum **1 R par village** et **2 R par château** (hors capitale : elle accumule tout le surplus). Si le joueur n'a **aucune capitale désignée** (aucun château contrôlé, ou désignation perdue et non redésignée), ses stocks **ne sont pas rapatriés** (ils restent sur place).
- **Ordre de la phase :** investissements → conservation 50 % → rapatriement.
- **Départ de la partie :** de 2 à 5 joueurs. Chaque joueur commence sur un **village distinct**, où un **château est construit automatiquement** (il remplace le village ; sa capitale par défaut — son premier château), avec **un noble**, **au moins une troupe** et un **stock de ressources initial** (valeurs d'équilibrage dans `assets/balance.json` : nobles, troupes et ressources de départ).
- **Élimination & Victoire :** un joueur est **éliminé** quand il ne contrôle plus **aucun territoire** ET n'a plus **aucune troupe** (ses nobles, immortels, ne comptent pas) ; il ne soumet plus d'ordres. Le dernier joueur en lice **remporte** la partie.

---

## 3. Carte, Terrains et Villages

### Types de Terrains (Vitesse des Messagers)

La propagation des messagers (rapports et ordres) dépend du relief — les troupes, elles, se déplacent toujours d'**1 case par tour** :

| Terrain | Vitesse du Messager |
| --- | --- |
| **Plaine** | **2 cases / tour** |
| **Forêt / Colline** | **1 case / tour** |
| **Montagne / Marécage** | **0,5 case / tour** (1 case tous les 2 tours) |

*(Les coûts de trajet des messagers — rapports et ordres — sont paramétrés dans les assets d'équilibrage `assets/balance.json` : coût par case selon le terrain — plaine 0,5, forêt/colline 1, montagne/marécage 2 —, divisé par 2 sur un Relais de Poste.)*

### Graphe franchissable : frontières géométriques

Le graphe de déplacement (troupes, messagers, flux) est formé des frontières réellement partagées entre territoires. Chaque arc géométrique est conservé et reçoit une qualification :

- **Frontières franchissables :** les arcs listés dans les adjacences sont franchissables par les troupes, les messagers et les flux.
- **Frontières infranchissables :** certaines frontières géométriques (crêtes montagneuses, marécages) sont classées infranchissables ; elles ne figurent pas dans le graphe franchissable.
- **Contraintes de degré :** le graphe franchissable reste connexe et chaque territoire a un degré d'au moins 2. Son degré maximal est de 3 en montagne, marécage ou colline, et de 5 en plaine ou forêt.
- **Pas de routes au MVP :** aucune arête ne relie des territoires sans frontière géométrique commune. Une évolution ultérieure pourra ajouter le qualificatif `route` à une frontière infranchissable pour représenter un pont ou un col et la rendre franchissable ; elle est hors MVP.

### Villages et Maillage Territorial

- **Zones Sauvages (~75 % de la carte) :** Produisent **0 R stockable**. Servent de zones de transit, de combat ou d'infrastructures isolées. La production de rations du terrain s'applique néanmoins à toutes les cases.
- **Production vivrière (rations) :** chaque territoire produit instantanément, à chaque tour, `RationTerrain[terrain]` rations (plaine/forêt/colline : 1 ; montagne/marais : 0), plus **2 rations** si la case porte un château **ou** un village. La règle de structure unique signifie que le bonus d'infrastructure ne s'applique qu'une fois : un château construit sur un village le remplace. Les rations sont **non stockables**, consommées sur place par les armées présentes quelle que soit leur nationalité ou le contrôle de la case ; les rations non distribuées sont perdues.
- **Villages :** Infrastructure **rare**, **neutre à l'origine** (non constructible au MVP, réservé) : **+2 rations vivrières** ; si contrôlé, **1 R stockable/tour** et ancre de ravitaillement. Un village est porté par un territoire non contrôlé.
- **Règle de la Structure Unique :** Une seule infrastructure par case.
- **Château :** **+2 rations vivrières** ; un château construit sur une case la rend **productive** et stocke **1 R stockable/tour si elle est contrôlée** ; il sert d'ancre de ravitaillement. Construit sur un village, il le **remplace** (jamais deux structures par case).

Les infrastructures **appartiennent à leur case** : elles n'ont pas de propriétaire — celui qui contrôle la case en bénéficie.

### Nommage des territoires

Chaque territoire porte le nom d'une commune de `communes.csv`. L'affinité de
terrain de la commune est privilégiée, puis l'affinité `any`, avec un repli
déterministe si nécessaire. Le code du territoire est le trigramme de sa
commune, unique sur la carte, et chaque commune est utilisée au plus une fois.

---

## 4. Latence d'Information et Transmission des Ordres

- **Vision Temps Réel (T0) :** Accordée sur les cases contenant un **Noble** du joueur (libre ou otage) et sur les cases d'une **Tour de Guet sur une case contrôlée par le joueur** et leurs **adjacentes**. Le château d'un joueur (sur une case qu'il contrôle) n'est PAS en T0 par défaut (sauf si un noble y est présent).
- **Rapports :** chaque **troupe**, chaque **château ou village contrôlé** et chaque **Tour de Guet** produit un rapport **chaque tour**, contenant l'état des **cases adjacentes**. Le temps de transit d'un rapport est calculé vers le **noble le plus proche** du joueur (coûts par terrain, cf. §3 — dans les assets d'équilibrage). Les rapports reçus sont **consolidés** dans la vue du joueur avec la **fraîcheur** de l'information (la date d'émission du rapport).
- **Projection :** en plus des rapports consolidés, la vue du joueur consolide la **projection** : l'emplacement de ses troupes SI les chaînes d'ordres valides et actives depuis leur dernier rapport ont été réussies. La vue distingue clairement l'**observé** (fraîcheur de l'information) de la **projection**.
- **Nobles prisonniers :** un noble capturé compte dans le calcul de la vue de son **propriétaire** (il est un récepteur de rapports, il en produit, et sa case est en T0 pour lui) — **sauf** si son **geôlier** l'a mis **au cachot** : il ne produit alors plus rien (ni récepteur, ni rapport, ni T0). Deux ordres de troupes font passer un noble prisonnier d'un état à l'autre : `XXX O <noble>` (otage, état par défaut) et `XXX K <noble>` (cachot) — ils peuvent faire partie des chaînes d'ordres.
- **Ordres par Messager :** les ordres partent du **noble émetteur** (celui de l'en-tête de la feuille) et voyagent à la vitesse du terrain (cf. §3) vers le **premier territoire de la feuille** d'ordres. **L'arrivée est calculée au moment de l'émission** (temps de trajet fixé). La **première troupe du joueur émetteur présente sur ce territoire** entre le moment de l'arrivée et la fin de l'hiver suivant (départage : plus petit matricule) **remplace la chaîne d'ordres de son armée** par celle-ci — aucune troupe n'est requise à l'émission (elle peut arriver plus tard) ; une chaîne jamais reçue est **perdue** à la fin de l'hiver. Une troupe poursuit son ancienne chaîne tant qu'aucun nouvel ordre ne l'a atteinte. **Pas d'interception** des ordres au MVP.

---

## 5. Combats et Logistique

### Combats et Force

- **Force de base :** 1 troupe = 1 force. **Les ordres s'appliquent aux armées** : toutes les troupes d'une armée partagent le même ordre courant — il n'existe jamais d'ordres mixtes au sein d'une armée.
- **Pas de fusion des attaques :** deux attaques sur la même case ne se combinent PAS : chaque armée attaquante est un contendant distinct (la fusion n'existe que via l'ordre de Jonction J, déplacement pacifique) — **y compris entre armées d'un même joueur** : deux de ses armées convergeant par A sur une même case se disputent la case (comparaison des contendents ; pour converger vraiment, il faut la Jonction J). Pour cumuler des forces sur une case, il faut un **soutien S** explicite.
- **Puissance d'attaque :** taille de l'armée attaquante.
- **Puissance de soutien :** taille de l'armée soutenante. Coupure (règle Diplomacy) : le soutien est coupé si l'armée soutenante est attaquée ce tour **depuis une case différente de celle vers laquelle elle soutient** (la cible de l'attaque soutenue en offensif, la case tenue en défensif) — une attaque venue de cette case-là ne coupe PAS le soutien.
- **Résolution (manière Diplomacy) :** l'ensemble des intentions (attaques, soutiens, déplacements pacifiques) est calculé et itéré jusqu'à stabilité AVANT d'exécuter les mouvements. Force égale = Statu quo. Supériorité numérique = Victoire. Une armée dont la case est **prise** est délogée : son propre ordre de mouvement est annulé (elle bat en retraite) ; si la case tient, son mouvement s'exécute normalement.
- **Multi-contendants :** quand plusieurs armées attaquent la même case, toutes les forces s'affrontent dans une comparaison unique (chaque armée attaquante + la défense). La force **strictement la plus haute** l'emporte et occupe la case ; **toute égalité au sommet = statu quo** : la défense tient et tous les attaquants échouent, même si leur force est supérieure à la défense — y compris deux attaques à égalité entre elles, fût-ce du même joueur (ex. 1 + 1 contre 1 : statu quo ; sur une case vide, deux attaquants à égalité échouent tous les deux ; pour prendre la case, faire une attaque soutenue).
- **Déplacements pacifiques (Jonction, Dispersion) :** puissance 0 ; repoussés si leur destination est contestée par une attaque ce tour.

### Ravitaillement Exponentiel

Une armée de N troupes sur une même case consomme :

Coût en R = 2^(N-1)

*(1 troupe = 1 R | 2 troupes = 2 R | 3 troupes = 4 R | 4 troupes = 8 R)*

Avant de tracer une ligne, la production de rations vivrières de la case est
distribuée aux armées présentes, toute nationalité confondue, à raison d'une
ration par armée au plus, en commençant par l'armée la plus grosse (départage
par matricule de troupe décroissant). La demande à ravitailler est le coût
moins les rations reçues. Une armée dont la demande résiduelle est nulle est
autonome pour le tour et ne requiert aucune ligne de ravitaillement.

### Portée des Flux

- Les lignes de flux sont tracées automatiquement depuis les châteaux et villages contrôlés à travers des territoires alliés ou neutres (une case contrôlée par un ennemi bloque le flux). La demande couverte est la **demande résiduelle**, après déduction des rations vivrières de la case.
- Un château ou village neutre n'est pas une source de ligne de ravitaillement : ses rations restent territoriales et nourrissent les armées présentes sur sa case, sans alimenter les flux.
- **Portée de base :** 3 cases. Chaque **Dépôt de Vivres sur une case contrôlée par le joueur** rencontré sur le trajet prolonge la portée de +2 cases (effets cumulables).

---

## 6. Ordres, Chaînes de Commandement & Retraites

### Types d'Ordres

- **Mouvement / Attaque / Maintien** (règles type *Diplomacy*). Le mouvement et l'attaque partagent la même mécanique de déplacement (le combat n'intervient que si la case est occupée par l'ennemi).
- **Soutien :** `XXX S YYY - ZZZ` (**offensif**) ou `XXX S YYY` (**défensif**), façon *Diplomacy* et **explicite** — on désigne l'armée soutenue, de **toute nationalité** (on peut soutenir l'attaque d'un autre joueur, y compris contre soi).
  - *Offensif* `XXX S YYY - ZZZ` : l'armée en XXX soutient l'attaque de l'armée en YYY vers ZZZ. Requis : ZZZ adjacente à XXX (et YYY–ZZZ adjacentes). Sans effet si l'armée de YYY n'attaque pas ZZZ ce tour (gaspillé, mais l'ordre réussit).
  - *Défensif* `XXX S YYY` : l'armée en XXX soutient l'armée en YYY qui **tient** (ordre sans déplacement : maintien, soutien ou pillage — pas A/J/D). Requis : YYY adjacente à XXX. Sans effet si l'armée de YYY se déplace (gaspillé, mais l'ordre réussit).
  - Puissance = taille de l'armée soutenante ; **coupure** : l'armée soutenante est attaquée depuis une case **différente** de celle vers laquelle elle soutient (cf. §5). En boucle : l'offensif soutient tant que l'armée soutenue attaque ZZZ, le défensif tant que YYY est attaquée — la chaîne avance sinon.
  - On ne peut soutenir sa propre case (YYY ≠ XXX) ni un J/D.
- **Jonction :** déplacement **pacifique** (pas une attaque, puissance 0) vers une case adjacente, **obligatoirement en dernier ordre d'une chaîne** (la jonction achève toujours une chaîne). Destination contestée par une attaque ce tour → **repoussé** ; occupée par l'ennemi → impossible. Sinon la jonctionnante s'y rend et **fusionne** si une troupe alliée s'y trouve déjà, **ou** si exactement une troupe alliée y arrive au même tour **sans contestation** (aucune autre troupe n'y converge — deux jonctions mutuelles J+J fusionnent aussi ; sinon, convergence multiple → repoussé). En cas de fusion, **la chaîne de l'hôte est conservée** — celle de la jonctionnante est de toute façon consommée, J étant le dernier ordre ; l'arrivant par A est l'hôte ; un rendez-vous J+J, qui fusionne deux chaînes achevées, laisse l'armée fusionnée **Sans Ordre**.
- **Séparation / Dispersion :** Diviser une armée de troupes. Chaque troupe se voit assigner une destination (adjacente ou la case d'origine, autorisée) libre et non ciblée par une attaque ce tour, vers laquelle elle se déplace pacifiquement. La chaîne d'ordres reste sur la troupe d'origine, qui prend la première destination listée.
- **Pillage :** Détruire l'infrastructure de **la case où se trouve la troupe** ; le bonus en R (valeur d'équilibrage) est crédité au **château ou village contrôlé le plus proche** du joueur (perdu s'il n'en contrôle aucun).
- **Otage / Cachot :** `XXX O <noble>` / `XXX K <noble>` — régit l'état d'un noble **prisonnier** détenu par l'armée de XXX : *otage* (état par défaut — il produit des rapports pour son propriétaire et compte en T0) ou *au cachot* (il ne produit plus rien). Requis : noble prisonnier du joueur de l'armée, sur la case de l'armée.

### Chaînes d'Ordres et Liaisons

Un joueur peut programmer des séquences d'ordres successives (O1 → O2 → O3). **Chaque transition (chaque ordre) possède sa propre liaison** : unique ou boucle. **Il n'existe pas de modification de chaîne** : une armée qui reçoit une chaîne remplace la précédente. Une chaîne est portée par une **armée** (représentée par sa troupe au plus petit matricule) et s'applique à **toutes** les troupes de l'armée — jamais d'ordres mixtes. La **jonction (J)** ne peut figurer qu'en **dernier ordre** d'une chaîne.

- **Liaison Unique :** En cas d'échec de Ox, la chaîne brise. L'armée passe *Sans Ordre*.
- **Liaison Boucle :** En cas d'échec de Ox, l'armée retente Ox au tour suivant jusqu'à réussite.
- **Ordre Invalide :** Tout ordre rendu physiquement ou mécaniquement impossible **brise immédiatement** la chaîne d'ordres, quel que soit le mode de liaison.
- **Position manquante :** chaque ordre précise explicitement la position de la troupe ; si la troupe n'y est pas quand l'ordre s'exécute → échec (single : brise ; boucle : retente).
- **Maintien en boucle** : garde indéfinie (la chaîne reste en veille jusqu'à réception d'un nouvel ordre).
- **Dispersion en boucle** : retente jusqu'à résolution intégrale ; en single, la chaîne avance même si la dispersion est partielle.
- **Progression simultanée :** la progression des chaînes prend en compte toutes les chaînes du tour pour résoudre combats, retraites et jonctions.

### Armées "Sans Ordre" (IA Défensive Auto-équilibrée)

Une armée sans ordre apporte un **soutien défensif** (équivalent d'un `XXX S YYY`) à l'**armée alliée la plus proche** qui possède le **moins de soutien**, stabilisant automatiquement la ligne de front.

- **Candidats :** armées alliées **en défense** (leur case est contestée par une attaque ennemie) ou **elles-mêmes sans chaîne**. Une armée qui attaque n'est jamais ciblée.
- **Exclusivement défensif :** le soutien automatique renforce la **défense** de la cible, jamais sa force d'attaque.
- **Puissance :** taille complète de l'armée soutenante (même règle que l'ordre S — les ordres s'appliquent aux armées).
- **Sélection (déterministe) :** par matricule croissant — (1) cible la plus proche (distance BFS, départage ID croissant), (2) moins soutenue (ordres S reçus + soutiens automatiques déjà assignés ce tour, compteurs mis à jour au fil de la sélection), (3) matricule croissant.
- **Coupure :** le soutien automatique d'une armée est coupé si cette armée est attaquée ce tour depuis une case différente de celle qu'elle soutient (même règle que S). L'armée soutenante n'émet aucun mouvement et reste Sans Ordre.

### Capacité des Nobles

Un noble ne peut émettre qu'**une seule chaîne d'ordres par tour** (remplacer une chaîne compte comme une émission). Un noble **prisonnier** ne peut émettre aucune chaîne.

### Nobles : Chevauchée, Capture et Libération

- **Chevauchée :** un noble se déplace AVEC les troupes de sa case : quand une armée quitte sa case (A, J, ou retraite), le noble la suit — l'armée se déplace toujours en bloc, la dispersion D est la seule façon de la séparer. En cas de **dispersion (D)**, une **astérisque** sur une destination désigne les nobles qui s'y rendent :
  - `XXX D YYY XXX*` — tous les nobles en XXX
  - `XXX D YYY*JEA ZZZ*` — Jean en YYY, les autres en ZZZ
  - `XXX D YYY*JEA*ANN ZZZ*` — Jean et Anne en YYY, les autres en ZZZ
  `*` seul = tous les nobles restants ; `*CODE` = le noble désigné (trigramme du prénom) ; chaque noble au plus une fois. **Si des nobles chevauchent l'armée et que l'ordre ne les mentionne pas TOUS** (aucune astérisque, ou nobles non couverts) → l'ordre est **INVALIDE** (il n'existe aucune troupe d'origine par défaut).
- **Jamais seul :** un noble ne peut pas être seul : toute destination désignée porte au moins une troupe, et un noble ne peut être recruté que sur une case contenant une troupe du joueur. (Cas limite toléré : armée détruite sur une case restée vide, le noble y demeure seul.)
- **Jamais une armée :** le noble ne compte pas dans les forces, ne consomme pas de ravitaillement et n'est jamais détruit dans un combat.
- **Capture :** un noble n'est capturé que si l'armée qu'il chevauche est **détruite au combat** (pas de retraite possible, ou collision) ET qu'une troupe **ennemie** occupe sa case : il est alors **récupéré par l'armée gagnante** — il devient **prisonnier**, chevauche cette armée (il se déplace selon les ordres du joueur qui la possède) et ne peut plus émettre de chaîne. Armée détruite sur une case restée vide ou alliée : le noble reste sur place.
- **Libération :** le **propriétaire** peut libérer un noble prisonnier par un ordre d'Hiver `L N <code noble>` ; le noble réapparaît libre à la **capitale de son propriétaire** (requis : une capitale désignée, sinon rejet ; coût : `assets/balance.json`, défaut 0).

### Phase de Retraite

Une armée défaite doit battre en retraite **en bloc** sur une case adjacente valide (non occupée, sans combat ce tour-ci, et différente de la case d'origine de l'attaquant). Les retraites sont traitées par matricule croissant : une armée évite une case déjà choisie par une retraite précédente si une autre case est valide.

- Si deux armées se replient sur la même case **sans autre option** : **les deux sont détruites**.
- Si aucune case n'est valide : **l'armée est détruite**.

---

## 7. Infrastructures et Maillage Productif

Les infrastructures **appartiennent à leur case** : elles n'ont pas de propriétaire — celui qui contrôle la case en bénéficie.

| Infrastructure | Nécessite un château/village ? | Effet principal | Coût (R) |
| --- | --- | --- | --- |
| **Moulin / Domaine** | **Oui** (sur le château/village ou directement adjacent) | Augmente la production de R **stockable** du château/village (+1 R / niveau, sur le château/village ou directement adjacent). | 3 |
| **Relais de Poste** | Non | Doubler la vitesse des messagers sur la case (tout messager, même ennemi, au MVP). | 2 |
| **Tour de Guet** | Non | Donne la vision T0 permanente sur la case et adjacentes. | 4 |
| **Dépôt de Vivres** | Non | Prolonge de +2 cases la portée des lignes de ravitaillement **sur une case contrôlée par le joueur** passant par sa case (cumulable). | 3 |
| **Château** | Non (Rend la case productive) | Apporte **+1 de force défensive** fixe (sans coût de R), **même sans garnison** : une case-château n'est jamais « vide » pour un attaquant ; +2 rations vivrières ; rend la case productive (**1 R stockable/tour si elle est contrôlée**). Construit sur un village, il le remplace. | 10 |
| **Village** | — | Non constructible au MVP (réservé) : **+2 rations vivrières par tour (non stockables)** ; si contrôlé, **1 R stockable/tour** et ancre de ravitaillement ; un château construit sur un village le remplace. | — |

### Dépendance et Moulins Orphelins

Un Moulin doit se trouver **sur un château/village ou directement adjacent à un château/village** : un moulin hors de ces cases est **orphelin** et ne produit rien (règle MVP ; le maillage de moulins en chaîne — infrastructures intermédiaires et dépendances de proche en proche — est post-MVP).

---

## 8. Contrôle Territorial et Carence Alimentaire

### Contrôle des Territoires

- Tout territoire bascule sous le contrôle d'un joueur dès qu'une de ses troupes s'y arrête (l'occupe en fin de résolution). Seuls les châteaux et villages produisent et stockent du R, mais le contrôle gouverne aussi la construction et le passage des flux.
- **Rémanence :** Le contrôle est conservé même après le départ de la troupe, jusqu'à l'arrêt d'une troupe ennemie.

### Algorithme de Carence Alimentaire (Famine)

Si la production instantanée de R stockable ne suffit pas à couvrir toutes les
demandes **résiduelles**, après déduction des rations vivrières produites sur
les cases :

```
[Déficit de Ravitaillement]
       │
       ▼
[1. Épuisement des Stocks (châteaux et villages contrôlés du joueur)]
   ► Ordre : Du plus PETIT stock au plus GRAND.
   ► Départage : Ordre alphabétique du trigramme du territoire.
       │
       ▼ (Si stocks totalement épuisés)
[2. Armées en Famine (Force = 0)]
   ► Ordre : Armées les PLUS ÉLOIGNÉES de leur source.
   ► Départage : Les armées les PLUS GROSSES d'abord
     (les plus coûteuses en nourriture).
   ► Départage final : Numéro de matricule de troupe DÉCROISSANT.
```

**Effets de la famine :** une armée famélique ne peut que se déplacer à force 0 (elle ne se bat ni en attaque, ni en défense, ni en soutien ; si elle est battue, elle bat en retraite normalement) ; si elle se trouve sur une case avec une infrastructure, elle la **pille automatiquement** (détruit l'infrastructure de la case — celle-ci ne peut être repillée ensuite). Le pillage lui rapporte le bonus R du pillage **moins la demande résiduelle de l'armée** : si le gain est positif ou nul, l'armée se nourrit et n'est plus famélique (l'excédent est crédité au château ou village contrôlé le plus proche du joueur, perdu s'il n'en contrôle aucun) ; sinon elle reste famélique sans rien gagner (l'infrastructure est détruite quand même).
