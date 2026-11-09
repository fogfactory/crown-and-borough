# Règles du jeu — Crown & Borough

Crown & Borough est un jeu de stratégie médiévale au tour par tour, sur une
carte de territoires. Chacun écrit ses ordres en secret, puis tout le monde
est résolu **en même temps**, saison après saison. Les nombres cités viennent
de la configuration du serveur ; en cas de doute, c'est le moteur qui fait
foi.

---

## L'essentiel pour gagner

![La carte et le poste de commandement au printemps](/rules/fr/overview.webp)

- **On gagne avec des titres.** Chaque fief, la couronne et la papauté
  rapportent 1 point d'influence. Les territoires, les armées et l'or ne
  rapportent rien par eux-mêmes : ils servent à fonder et à tenir des fiefs.
- **Atteins l'objectif avant la fin.** La partie s'arrête dès qu'un joueur, ou
  un couple allié, atteint son objectif d'influence. Sinon, c'est le meilleur
  score à la dernière année qui l'emporte.
- **Tes nobles donnent les ordres.** Chaque noble écrit une seule chaîne
  d'ordres par saison pour l'une de tes armées. Plus tu as de nobles, plus tu
  peux agir.
- **Une grosse armée coûte très cher.** Le coût en vivres double à chaque
  troupe supplémentaire, et une armée coupée de son ravitaillement s'affame.
- **L'hiver, on bâtit.** Pas de combat en hiver : on y construit, on fonde des
  fiefs, on marie ses nobles et on élit évêques, pape et roi.
- **Le mariage est une arme.** Un mariage de tête unit deux maisons pour une
  victoire commune ; les autres mariages t'apportent l'influence des titres de ton
  conjoint.

---

## 1. Une année en quatre saisons

Une partie dure de 1 à 50 années (10 par défaut). Chaque année compte quatre
tours : **printemps, été, automne** (les saisons d'action) et **hiver**.
L'interface affiche l'année historique `1000 + année`.

**Mise en place.** Une partie en ligne réunit de 2 à 8 joueurs (jusqu'à 16 en
local). Chacun démarre avec :

- une **capitale** : un château sur un territoire non montagneux, avec
  {{starting_resources}} R de stock et une armée de {{starting_troops}}
  troupes ;
- {{starting_nobles}} nobles libres ;
- {{starting_outposts}} avant-postes d'une troupe sur des territoires voisins.

**Une saison d'action**, c'est :

1. tu écris une chaîne d'ordres par noble, et tu joues tes cartes ;
2. le moteur résout tout le monde ensemble : soutiens, combats, mouvements,
   retraites et transferts ;
3. le contrôle des territoires est mis à jour ;
4. le ravitaillement est calculé sur ces positions finales : revenus,
   production, rations et famine ;
5. chacun reçoit son rapport de tour.

**L'hiver** est une trêve : pas de chaîne, pas de mouvement, pas de combat.
Tu y investis, tu y gères tes nobles et tes titres (section 5).

**Ce qui est public, ce qui est secret.** La carte, les armées, les stocks,
les infrastructures et la position des nobles sont visibles par tous. Tes
**intentions** restent secrètes : en ligne, seuls les participants d'une
chaîne ou d'un combat en voient le détail. Les autres ne voient que le
résultat.

> L'annexe D déroule un tour complet sur un exemple.

---

## 2. Commander : nobles et chaînes

### La chaîne d'ordres

Une **chaîne** est une suite d'ordres écrite à l'avance pour **une armée
entière**. Chaque noble libre ou otage en émet **une par saison**, pour
n'importe quelle armée de son joueur, même éloignée. L'interface ajoute
d'elle-même le nom du noble en tête : tu n'écris que les lignes d'ordres, une
par ligne, sous la forme `POSITION SYMBOLE [cibles]`.

![Deux chaînes en saisie, et leurs flèches sur la carte](/rules/fr/chain.webp)

- **Une ligne par saison.** Une armée exécute une seule ligne à chaque saison
  d'action : avec `ROS A BOI` puis `BOI A ATL`, elle atteint BOI cette saison
  et ATL la suivante. La chaîne reste attachée à l'armée jusqu'à son terme.
- **single ou loop.** Une ligne simple casse la chaîne si elle échoue. Une
  ligne entre parenthèses, `(…)`, est retentée chaque saison jusqu'à sa
  réussite.
- **Une nouvelle chaîne remplace l'ancienne.** Si deux de tes nobles visent la
  même armée la même saison, aucune des deux chaînes n'est reçue.
- **Sans chaîne, l'armée est « Sans Ordre ».** Elle reste sur place et se
  défend normalement.
- **Les erreurs sont refusées d'emblée.** Une soumission erronée (syntaxe,
  cases non adjacentes, etc.) est rejetée en entier, avec la ligne à
  corriger. L'interface les signale pendant la saisie.

### Les ordres

| Ordre | Syntaxe | Effet |
|---|---|---|
| Attaque | `XXX A YYY` | Entre sur `YYY` adjacente, en combattant si besoin. |
| Soutien défensif | `XXX S YYY` | Renforce l'armée qui reste sur `YYY`. |
| Soutien offensif | `XXX S YYY - ZZZ` | Renforce l'attaque de `YYY` vers `ZZZ`. |
| Maintien | `H XXX` | Reste sur place. Utile pour patienter : `(H XXX)`. |
| Jonction | `XXX J YYY` | Rejoint pacifiquement `YYY` ; dernier ordre de la chaîne. |
| Pillage | `P XXX` | Dégrade l'infrastructure de la case et rapporte {{pillage_bonus}} R. |
| Dispersion | `XXX D DEST1 DEST2 …` | Envoie une troupe vers chaque destination, dans l'ordre. |
| Transfert | `XXX T YYY N` | Envoie `N` R par le réseau de ravitaillement. |

L'essentiel à retenir :

- **Attaque et jonction** visent une case voisine par une frontière
  franchissable. Une jonction ne combat jamais : elle échoue si la case est
  disputée.
- **Un soutien** renforce une armée de n'importe quel joueur. Le soutien
  défensif ne compte que si l'armée soutenue reste sur sa case et n'attaque
  pas. Une attaque contre l'armée qui soutient, venue d'ailleurs que de la
  cible, **coupe** le soutien.
- **Une dispersion** répartit une armée sans combattre. Les troupes qui ne
  peuvent pas partir restent sur place, et la chaîne suit le premier groupe
  listé.
- **Un transfert** envoie des ressources à un château, un village ou une armée
  d'un autre joueur, à portée de ravitaillement. Une armée adverse sur le
  trajet le bloque. Une armée qui transfère ne fait rien d'autre ce tour-là.
- **Une jonction ou une dispersion est annulée** si sa case de départ est
  attaquée. On ne fuit pas un combat.

Les détails (nobles dans une dispersion, plafonds de transfert, cas limites)
sont en annexe C.

---

## 3. Combattre

### Qui l'emporte

Une case ne porte jamais qu'une seule armée. Sa **force** se calcule ainsi :

- sa **taille** en troupes ;
- **+{{noble_command_bonus}}** si un de tes nobles libres est sur sa case ;
- **+ chaque soutien**, qui pèse la taille de l'armée qui soutient (avec le
  même bonus de noble) ;
- en défense seulement, **+{{castle_defense_bonus}}** pour un château ou un
  village fortifié actif, et **+{{city_defense_bonus}}** pour une cité, le
  château d'une capitale de fief.

**La plus forte l'emporte, à condition d'être seule en tête.** Une égalité au
sommet produit un statu quo : personne n'entre.

- Deux armées qui s'attaquent mutuellement : la plus forte déloge l'autre.
- Des attaques en cercle (A→B, B→C, C→A) réussissent toutes si rien ne s'y
  oppose.
- Tu ne déloges jamais ta propre armée, et les soutiens que tu donnes à un
  adversaire ne l'aident jamais contre toi.

### Battre en retraite

L'armée vaincue recule en entier vers une case voisine, dans cet ordre de
préférence :

1. une case vide **ancrée** chez toi : un membre de l'un de tes fiefs, ou ta
   capitale ;
2. une autre case vide, sans château actif, qui n'a pas été attaquée ce
   tour ;
3. une de tes armées voisines, qui l'absorbe. Les armées qui ne sont pas
   affamées passent en premier, puis la plus petite.

Sans destination possible, l'armée est **détruite**. Ses nobles sont alors
capturés par l'armée ennemie présente, et deviennent ses otages.

> **Astuce.** Un fief offre toujours un repli sûr, même sur une case
> combattue ce tour-ci. Les tie-breaks complets sont en annexe C.

### Contrôle et occupation

- **Hors fief, le contrôle est éphémère.** Une case est à toi tant qu'une de
  tes armées y stationne. Dès qu'elle part, la case redevient neutre.
- **Un fief est solide.** Ses membres restent à toi même si une armée ennemie
  s'y arrête : elle les **occupe** sans les prendre. Seule la prise de la
  **capitale du fief** fait tomber tout le fief d'un coup.
- **Une case occupée par l'ennemi** ne sert ni de source ni de dépôt, et
  n'accepte aucun investissement d'hiver. Son revenu continue de te revenir.
- **Ta capitale** reste à toi, même vide.

### Les nobles au combat

Les nobles suivent leur armée dans tous ses mouvements. Ils ne comptent ni
dans les pertes ni dans le ravitaillement. Le bonus de commandement ne vient
que d'un noble **présent** sur la case : commander à distance n'en donne
aucun.

---

## 4. Nourrir ses armées

### Le coût d'une armée

Le ravitaillement est calculé **à la fin de chaque saison d'action**, jamais
en hiver. Une armée de `N` troupes demande `{{cost_base}}^(N − 1)` rations :

| Taille | 1 | 2 | 3 | 4 | 5 |
|---|---:|---:|---:|---:|---:|
| Rations | {{army_cost.1}} | {{army_cost.2}} | {{army_cost.3}} | {{army_cost.4}} | {{army_cost.5}} |

### D'où vient la nourriture

1. **Le terrain de la case** nourrit l'armée qui s'y trouve : plaine
   {{ration_terrain.plain}}, forêt {{ration_terrain.forest}}, colline
   {{ration_terrain.hill}}, montagne {{ration_terrain.mountain}}, marécage
   {{ration_terrain.swamp}}.
2. **Le reste vient de tes sources** : tes châteaux, tes villages, tes caches
   (une case qui garde du stock) et tes moulins isolés. Elles nourrissent les
   armées situées à **{{supply_range}} cases** ou moins. Chacun de tes dépôts
   de vivres sur le trajet ajoute {{depot_range_bonus}} cases. Seule une
   armée adverse arrête le flux.

![Fiche d'un territoire : ressources, armée, nobles présents](/rules/fr/territory.webp)

Sélectionne une armée ou une source pour voir sa ligne ou sa zone de
ravitaillement. Le résumé du trésor affiche le risque de famine prévu.

### Revenus et production

- **Revenu territorial.** Chaque territoire contrôlé rapporte
  {{territory_income}} R par saison d'action, plus {{village_income}} R par
  niveau de village. Ce revenu va à la capitale de son fief, sinon à ta
  capitale. Il n'est jamais intercepté.
- **Moulin.** Un moulin de niveau `N` produit `N` R par saison d'action. Il
  verse sa production à ton château voisin, sinon à ton village voisin, sinon
  la garde sur sa propre case.
- **Village.** Un village a un niveau de 0 (ruine) à {{village_max_level}}.
  Neutre, il remplit son propre stock, que tu récupères en le prenant.

### La famine

Une armée qui manque de rations est **affamée** jusqu'au ravitaillement
suivant. Sa **valeur ravitaillée** `s` est la plus grande taille qu'elle
aurait pu nourrir. Une armée affamée :

- se défend à la force `s − 1`, sans bonus de noble ;
- **ne peut ni attaquer, ni soutenir, ni transférer** (elle peut recevoir un
  transfert) ;
- peut être visée par une carte Révolte (section 8).

Si elle a encore faim au ravitaillement suivant, elle pille d'abord
l'infrastructure de sa case. Si ce n'est pas assez, elle **perd des troupes**
jusqu'à revenir à `s`. À `s = 0`, elle disparaît et ses nobles meurent avec
elle.

> **Bon à savoir.** Tu as toujours une saison complète pour réagir :
> rapproche l'armée d'une source, divise-la, ou envoie-lui des vivres. Les
> groupes nés d'une dispersion restent affamés jusqu'au ravitaillement
> suivant, mais chacun, plus petit, se nourrit plus facilement.

### Infrastructures et pillage

Une case porte une seule infrastructure. Un château, un village fortifié et un
dépôt de vivres ne fonctionnent que s'ils sont **actifs** : membres d'un fief,
capitale d'un joueur, ou tenus par une armée. Sinon ils sont **inertes**.

| Infrastructure | Rôle | Si elle est pillée |
|---|---|---|
| Château | Défense +{{castle_defense_bonus}}, source, reçoit le revenu | Il tombe en ruine |
| Village | Source, revenu +{{village_income}} R par niveau | Il perd 1 niveau (0 = ruine) |
| Village fortifié | Comme un village, plus la défense d'un château | Sa muraille tombe |
| Moulin | `N` R par saison | Il perd 1 niveau, et disparaît au niveau 1 |
| Dépôt de vivres | Portée +{{depot_range_bonus}} | Il disparaît |

Une ruine ne peut pas être pillée et ne produit rien. Le stock d'une
infrastructure pillée s'ajoute au butin du pilleur.

---

## 5. L'hiver : bâtir et investir

![Le panneau des ordres d'hiver](/rules/fr/winter.webp)

L'hiver, tu remplis une feuille d'ordres directs, sans chaîne. Les principaux
investissements :

| Pour… | Ordre | Coût |
|---|---|---|
| Recruter une troupe | `R T XXX` | {{costs.troop}} R |
| Construire ou améliorer un moulin | `C M XXX` | {{costs.mill_levels.0}} / {{costs.mill_levels.1}} / {{costs.mill_levels.2}} R |
| Construire un château, ou fortifier un village | `C C XXX` | {{costs.castle}} R |
| Construire un dépôt de vivres | `C D XXX` | {{costs.supply_depot}} R |
| Fonder un fief | `T F NNN XXX YYY ZZZ…` | {{costs.fief_per_territory}} R par territoire |
| Annexer un territoire à un fief | `T X NNN XXX YYY` | {{costs.annex_per_territory}} R |
| Monter un fief d'un rang | `T R NNN XXX` | {{costs.rank_per_territory}} R par territoire |
| Désigner ta capitale | `E C XXX` | gratuit |
| Envoyer des R à un autre joueur | `G XXX YYY N` | gratuit |

La liste complète est en annexe B. Ce qu'il faut savoir :

- **On paie sur place, puis au plus proche.** Le coût est pris sur le stock de
  la case visée, puis sur tes sources les plus proches. Si le total ne suffit
  pas, l'ordre est refusé sans rien prélever.
- **On ne bâtit que chez soi.** Le territoire doit être contrôlé par toi, et
  non occupé par l'ennemi.
- **Les stocks fondent en hiver.** Chaque château, village et moulin ne garde
  que son stock divisé par {{winter_stock_divisor}}, arrondi au supérieur
  (5 R deviennent 3 R). Les dépôts gardent tout. Ce qui reste est ensuite
  rapatrié vers ta capitale, en laissant au plus {{village_stock_cap}} R dans
  chaque village et {{castle_stock_cap}} R dans chaque château.
- **La prospérité naît des pertes.** Chaque tranche de
  {{prosperity_loss_threshold}} R perdue pendant l'hiver, sur toute la carte,
  fait naître ou grandir un village près de là où le stock a été perdu.

Les ordres d'hiver se résolvent par étapes, dans un ordre fixe : sanctions du
pape, ordres royaux, gestion, enquêtes et procès déposés, dissolutions,
mariages, élections, investiture, fin d'hiver, puis les jugements et les
assassinats. Le détail est en annexe B.

---

## 6. Nobles, fiefs et couronne

### Le deck de nobles

Les nobles se piochent dans un **deck commun** à tous les joueurs, qui mêle
des cartes de noble (un prénom, un sexe) et des cartes de dignité.

- **Piocher** : `T N`, une fois par hiver.
- **Recruter** : `R N CCC XXX` fait apparaître le noble de la carte `CCC` sur
  `XXX`, qui doit être un château ou un village à toi, avec une de tes armées.
  Le noble agit à partir de la saison suivante.
- **Plafond** : tu possèdes au plus {{noble_limit}} nobles vivants. Chaque
  bâtard relève ce plafond de 1, jusqu'à {{noble_limit_max}}.
- **Main** : {{special_orders.hand_limit}} cartes au plus, tous decks
  confondus. `D C CCC` défausse une carte.

Les cartes de noble (`R N`), de prétention (`C N`) et de dignité (`D N`) se
jouent en toute saison.

### Otages et prisonniers

Un noble capturé devient **otage** de l'armée qui l'a pris. Un bâtard va
directement au **donjon**. En hiver, le geôlier peut basculer un prisonnier
entre otage et donjon (`O N` / `P N`), le céder ou le rendre (`H N`).

- Un **otage** peut toujours émettre une chaîne pour son joueur, mais son
  geôlier la lit.
- Au **donjon**, il ne commande rien, et ses dignités et ses titres
  religieux sont suspendus.

### Les fiefs

Un **fief** se fonde l'hiver avec `T F` : au moins 3 territoires contrôlés et
contigus, dont le premier, la **capitale**, porte un château. Son titre suit
sa taille : **baronnie** (3), **comté** (4), **marquisat** (5), **duché** (6
et plus). Le château de la capitale devient une **cité**.

- Un fief vaut **1 point d'influence**, quel que soit son rang.
- Ses membres restent à toi même occupés. Seule la prise de sa capitale le
  fait tomber (section 3).
- Tu l'agrandis avec `T X` et le fais monter d'un rang avec `T R`, une fois
  par hiver.
- Si son titulaire meurt ou si la capitale change de main, le fief devient
  **vacant**. Il produit et compte toujours. Désigne un nouveau titulaire avec
  `T A`, sinon le premier noble de ta lignée le reçoit en fin d'hiver.

**La ligne de succession** est l'ordre de recrutement de tes nobles, les
bâtards toujours en dernier. Un noble ne reçoit un titre que si les nobles
**du même sexe** placés avant lui détiennent déjà un titre au moins
équivalent. Le clergé ne bloque personne.

### Mariages et alliances

![L'arbre de lignée et alliances](/rules/fr/lineage.webp)

Deux nobles libres et célibataires, de sexe différent et de deux joueurs
différents, se marient l'hiver si **chaque joueur** écrit l'ordre :
`M N NNN MMM` d'un côté, `M N MMM NNN` de l'autre. Sans réciproque, le
mariage est refusé.

Chaque mariage a un **poids** : celui de l'époux le plus faible, d'après son
rang dans la lignée et son titre, plus 1 pour chaque autre mariage entre les
deux mêmes maisons.

- **Le mariage de tête** est celui de plus grand poids pour les deux
  maisons. Il forme une **alliance complète** : les deux scores
  s'additionnent pour une **victoire commune**, mais aucune des deux maisons
  ne peut gagner seule.
- **Les autres mariages** sont des **mariages d'influence** : chaque maison
  gagne autant de points que les titres détenus par le conjoint de l'autre
  maison : 1 par fief dont il est titulaire, +1 s'il est roi.
- Le mariage d'un bâtard n'est pas une alliance.

La vue **Lignée et alliances** montre les familles, les mariages, les
prétentions, et propose un simulateur de victoire.

### Prétentions (Claims)

Un de tes nobles recruté pendant un mariage entre l'un des tiens et le noble
`CCC` d'un autre joueur peut réclamer les titres de `CCC` : `C N HHH CCC`, avec une
carte de prétention (`CLM`). À la mort de `CCC`, ses fiefs passent au premier
héritier vivant, du Claim le plus ancien au plus récent. Un Claim est public.

### Les dignités

Une carte de dignité se joue avec `D N NNN CCC` sur n'importe quel noble, le
tien ou un autre. Une dignité ne rapporte aucun point d'influence : elle vaut
par ses effets.

- **Le bâtard** (`BAS`) se joue sur tout noble. Il relève ton plafond de
  nobles, ferme la ligne de succession, ne peut pas être roi, et son mariage
  n'est pas une alliance.
- **Les dignités de dame** ne se jouent que sur une dame, qui en porte au plus
  une visible et une cachée. Une dignité **bloquée** interdit le mariage ; une
  dignité **libre** le permet.

| Dignité | Code | Mariage | Effet |
|---|---|---|---|
| D'Arc | `ARC` | Bloquée | +1 de force à l'armée qu'elle commande. |
| Châtelaine | `CTL` | Libre | Dans un château, révèle les ordres émis sur son fief. |
| Abbesse | `ABB` | Bloquée | Liée à une région : révèle les chaînes émises dans cette région. |
| Herboriste | `HRB` | Bloquée | Protège de la peste sa case et ses voisines ; son armée mange 2 rations de moins. |
| Astrologue | `AST` | Libre | Montre les 3 prochaines calamités ; `V C NNN I` en raye une par hiver. |
| Chevalier d'Éon | `EON` | Libre (célibataire) | Cachée : la dame se présente comme un homme. |
| Correspondante | `COR` | Libre | Cachée : otage d'un joueur, révèle tous ses ordres. |
| Espionne | `ESP` | Libre | Cachée : otage d'un joueur, révèle toute sa main. |
| Empoisonneuse | `EMP` | Libre | Cachée : les armées adverses de sa région mangent 1 ration de plus. |
| Sorcière | `SOR` | Libre | Cachée : `S R` oriente une calamité vers sa région. |

Les dignités cachées ne sont connues que de leur joueur, jusqu'à ce qu'une
enquête, une capture ou un procès les révèle. Les règles fines de chaque
dignité sont en annexe C.

### La couronne

**Le roi** est un homme, non bâtard, qui ne peut pas être pape. La couronne
vaut **1 point d'influence**, en plus de ses fiefs. Le roi règne à vie et
garde ses pouvoirs s'il est otage. S'il est au donjon, la **reine** exerce
la régence à sa place.

- **Ordres royaux** (`W X`, `W R`, `W F`, `W G`) : autoriser l'annexion d'un
  territoire d'un autre fief, rendre gratuite une montée de rang ou une
  fondation de fief, octroyer un fief de la couronne. Chaque type sert au plus
  {{royal.orders_per_winter}} fois par hiver.
- **Rente royale** : 1 R par fief de la carte à chaque saison d'action ; la
  reine en touche la moitié.
- **Taxe royale** : la carte Impôts jouée par le roi sur un fief qui n'est pas
  le sien (section 8).

**Élire un roi.** Si le trône est vacant, que 3 nobles titrés au moins
peuvent voter et que le clergé peut oindre, une élection s'ouvre. Présente un
candidat avec `K R NNN` (un homme libre détenteur d'un fief, non excommunié),
puis vote avec `V R NNN`. Chaque noble titré donne une voix à son joueur.
L'élu doit être seul en tête, avec **3 voix au moins**, dont l'**onction** :
deux voix d'évêques, ou une voix de cardinal ou de pape.

**À la mort du roi**, son premier héritier de Claim éligible reçoit la
couronne. Sinon, la reine assure la régence et le trône est vacant.

---

## 7. L'Église

![Le marque-page Royaume : pape, cardinaux et évêques](/rules/fr/realm.webp)

La carte est divisée en **évêchés**, les mêmes régions que celles des
calamités. La **papauté** vaut **1 point d'influence** ; les évêques et les
cardinaux ne rapportent aucun point, mais ils donnent des voix et des
pouvoirs.

### Évêques

Un évêché sans évêque ouvre une élection en hiver dès que **chacun de ses
territoires** est tenu par un joueur.

- **Candidature** : `K E NNN BBB`, où `BBB` est le chef-lieu de l'évêché. Le
  candidat est un homme libre, célibataire, non excommunié.
- **Vote** : `V E NNN BBB`. Ton joueur pèse 1 voix par territoire tenu dans
  l'évêché (2 pour le chef-lieu), plus 1 par évêque, 2 par cardinal, 3 pour
  le pape et 1 par abbesse de cet évêché.
- Le mieux placé l'emporte, s'il est seul en tête. Sinon, le siège reste
  vacant jusqu'à l'hiver suivant.

### Cardinaux et pape

Un **cardinal** est un évêque promu, de deux façons : par la carte `CAR`
(`D N NNN CAR`), effective aussitôt, ou par l'achat `N C NNN` pour
{{religion.cardinal_cost}} R, effectif à la fin de l'hiver. Le nombre de
cardinaux est plafonné selon le nombre de joueurs.

Dès que 2 cardinaux sont en jeu et que le trône pontifical est vide, un
**conclave** s'ouvre. Les candidats sont des évêques ou des cardinaux
(`K P NNN`). Seuls les cardinaux votent, avec une voix chacun (`V P NNN`).
L'élu doit obtenir **plus de la moitié** de tous les cardinaux.

Un titre gagné en hiver ne sert qu'à partir de l'hiver suivant. Un titre
religieux se perd à la mort ou par excommunication, jamais par capture.

### Les pouvoirs de l'Église

| Pouvoir | Qui | Ordre | Effet |
|---|---|---|---|
| Excommunier / lever | Pape | `X E NNN` / `X L NNN` | Prive le noble de vote et de candidature ; un évêque ou un cardinal perd son titre. Une par hiver, un excommunié à la fois par joueur. |
| Enquêter | Cardinal ou pape | `Q HHH NNN` | Révèle à tous la dignité cachée de `NNN`, et à toi seul le commanditaire d'un assassinat. Coûte de 1 à 5 R selon le rang de la cible. |
| Procès | Deux cardinaux | `J HHH NNN` chacun | Exécute une dame non mariée qui porte une dignité (hors Abbesse), un excommunié ou un commanditaire d'assassinat. Le roi y échappe. |
| Dissoudre un mariage | Pape, à la demande d'un époux | `X D NNN` des deux côtés | Les époux redeviennent célibataires. |
| Dîme | Évêque, cardinal ou pape | `P DI HHH XXX` | Carte Impôts : détourne la production des moulins d'un évêché (section 8). |
| Apaiser une révolte | Ecclésiastique ou abbesse | `P AG HHH TER` / `P AP HHH TER` | Retire une armée rebelle (section 8). |

---

## 8. Cartes spéciales et calamités

Les cartes se jouent au printemps, en été ou en automne, dans le champ
**cartes spéciales**, sans noble. Elles sont résolues avant les ordres
d'armée. En hiver, ta main se complète d'elle-même, jusqu'à
{{special_orders.draw_orders_limit}} cartes par hiver tous decks confondus.

Le deck compte {{special_orders.deck_size}} cartes.

### Les calamités

| | Calamité | Effet sur sa région |
|---|---|---|
| <img src="/icons/reaper-scythe.svg" width="22" alt=""> | Peste ({{special_orders.card.plague}}) | Divise les armées par {{special_orders.effects.plague_army_divisor}} et peut tuer des nobles. |
| <img src="/icons/raining.svg" width="22" alt=""> | Mauvais temps ({{special_orders.card.bad_weather}}) | Bloque les mouvements qui partent de la région ou y entrent (sauf maintien et soutien défensif) ; les moulins ne produisent rien. |
| <img src="/icons/desert-skull.svg" width="22" alt=""> | Mauvaise récolte ({{special_orders.card.famine}}) | Supprime les rations de terrain et le revenu territorial. |

Une calamité piochée est programmée pour l'année suivante, au plus une par
saison. Elle est annoncée dès son tirage, et le **présage du printemps**
révèle la saison et la région de toutes les calamités de l'année.

### Les cartes bonus

| | Carte | Ordre | Effet |
|---|---|---|---|
| <img src="/icons/sun.svg" width="22" alt=""> | Beau temps ({{special_orders.card.fair_weather}}) | `P BT XXX` | Annule le mauvais temps ; sinon, double la production des moulins de la région. |
| <img src="/icons/wheat.svg" width="22" alt=""> | Bonne récolte ({{special_orders.card.abundant_harvest}}) | `P RA XXX` | Annule la mauvaise récolte ; sinon, double les rations et le revenu de la région. |
| <img src="/icons/uprising.svg" width="22" alt=""> | Révolte ({{special_orders.card.revolt}}) | `P RE TER` | Lève des rebelles, ou retourne une armée affamée (voir ci-dessous). |
| | Impôts ({{special_orders.card.seigneurial_tax}}) | `P TX …` / `P DI …` | Taxe seigneuriale, taxe royale ou dîme. |
| | Souterrain ({{special_orders.card.tunnel}}) | `P SO …` | Fait évader un noble, ou fait ignorer les murailles à une attaque. |
| | Embuscade ({{special_orders.card.ambush}}) | `P EM AAA BBB` | Si l'attaque gagne, capture un noble de l'armée vaincue. |
| | Assassinat ({{special_orders.card.assassination}}) | `P AS HHH NNN` | Tue `NNN`, en secret. Jouable aussi en hiver. |
| | Justice ({{special_orders.card.justice}}) | `P JU HHH` | Exécute `HHH` s'il a commandité un assassinat. Jouable aussi en hiver. |

Pour Beau temps et Bonne récolte, `XXX` est le chef-lieu de la région. Le
bonus ne s'applique qu'une fois par région et par saison.

**Révolte.** Elle se joue sur un territoire :

- dont la région subit une mauvaise récolte ;
- ou dont le fief (ou l'évêché) a été taxé cette saison ou la précédente ;
- ou dont la région a vu une dame exécutée par procès à la saison
  précédente.

Chaque carte lève de {{special_orders.effects.revolt_army_min_size}} à
{{special_orders.effects.revolt_army_max_size}} troupes rebelles neutres, qui
combattent l'occupant. Jouée sur une **armée affamée**, la carte ne lève
aucun rebelle : si le total des jets atteint la taille de l'armée, celle-ci
devient neutre et ses nobles libres meurent.

**Impôts.** La même carte a trois usages :

- **Taxe seigneuriale** (`P TX XXX`, ou `P TX HHH XXX` par le titulaire) :
  double le revenu territorial d'un de tes fiefs pendant la saison.
- **Taxe royale** (`P TX KKK XXX`, par le roi) : crée pour toi le même
  montant que le revenu d'un fief qui n'est pas le tien.
- **Dîme** (`P DI HHH XXX`, ou `P DI HHH` pour le pape sur tous les
  évêchés) : détourne vers ta capitale la production des moulins de
  l'évêché.

Toute taxe rend le fief ou l'évêché éligible à la Révolte pendant deux
saisons.

**Apaisement.** Un ecclésiastique retire une armée rebelle, sans carte, une
fois par saison :

- `P AG HHH TER`, le rite gratuit, dans sa région : un d6 réussit sur 1 à 3,
  échoue sur 4 ou 5, et tue l'ecclésiastique sur 6 ;
- `P AP HHH TER`, l'apaisement payant et sans risque :
  {{religion.appeasement_cost_base}}^taille R.

**Assassinat et Justice.** L'assassinat n'est pas signé : seuls le
commanditaire et ceux qui enquêtent sur la victime ou le commanditaire
connaissent le coupable. Justice ne frappe que juste : jouée sur un innocent,
elle est perdue.

**Rumeurs.** Le rapport signale, sans nommer personne, les cartes qui
circulent dans les mains.

---

## 9. Fin de partie et victoire

![Le panneau des scores et l'objectif de chacun](/rules/fr/scores.webp)

**Influence.** Ton score d'influence est la somme :

- de tes **titres** : 1 point par fief (vacant compris), 1 pour la couronne
  et 1 pour la papauté ;
- de tes **mariages d'influence** : les titres détenus par le conjoint de chaque maison
  alliée par un mariage secondaire ;
- de l'influence de ton **allié de tête**, si tu en as un.

Les dignités, les évêchés, les cardinalats, les territoires, les armées et
les ressources ne rapportent rien.

**Objectif.** Il dépend du nombre de joueurs, et s'affiche dans le panneau des
scores.

| Joueurs | 2 | 3 | 4 | 5 | 6 | 7 | 8 |
|---|---:|---:|---:|---:|---:|---:|---:|
| Seul | 3 | 5 | 6 | 8 | 9 | 11 | 12 |
| En alliance de tête | — | 6 | 8 | 9 | 11 | 13 | 15 |

**Fin de partie.** La partie s'arrête après le tour où :

- un joueur sans alliance de tête atteint l'objectif solo ;
- ou un couple allié atteint ensemble l'objectif d'alliance ;
- ou un seul joueur reste en lice ;
- ou la dernière année est jouée. Le meilleur score l'emporte alors.

**Victoire.** Le gagnant, ou le couple gagnant, obtient une **victoire
majeure**. Parmi les maisons liées au vainqueur par des mariages, celle du
lien le plus fort obtient une **victoire mineure**, sauf égalité. Les autres
échouent. Les
égalités se départagent par la victoire solo, puis le score, la couronne et
le nombre de territoires. Les ex æquo restants gagnent ensemble.
Une partie où personne ne détient de titre n'a aucun vainqueur.

**Élimination.** Un joueur qui n'a plus ni fief, ni capitale, ni armée est
éliminé. Ses nobles seuls ne le maintiennent pas en jeu.

---

## 10. Messages entre maisons

Le chat de la partie permet aux maisons de s'écrire sans passer par les
ordres. Il n'existe qu'en partie online : il est absent en hotseat.

- **Les canaux.** Le **salon public** réunit toutes les maisons. Une
  **conversation privée** relie deux maisons, et chaque paire de maisons n'en
  a qu'une ; il n'y a pas de groupe. Tu peux ouvrir une nouvelle conversation
  avec toute maison encore en jeu, et continuer celles qui existent même si
  l'interlocutrice est éliminée. Un joueur éliminé peut toujours écrire.
- **La signature.** Chaque message est signé du nom de ta maison, avec son
  blason et sa couleur. Ton compte n'apparaît jamais.
- **Qui lit quoi.** Tout le monde lit le salon public. Une conversation privée
  n'est lue que par ses deux maisons. L'**hôte qui ne joue aucune maison**,
  l'observateur, lit le salon et toutes les conversations privées, sans y
  écrire : si tu joues dans une partie, sache que ce que tu écris en privé
  peut être lu par cet hôte. Un hôte qui joue une maison lit comme les autres.
- **Historique et non-lus.** L'historique est conservé ; les 50 derniers
  messages s'affichent, et tu peux charger les plus anciens. Une pastille
  compte les messages non lus de chaque canal. Des séparateurs (« Printemps
  1011 ») marquent les tours.
- **Limites.** Un message compte 500 caractères au plus, et tu peux en
  envoyer un par seconde. Un texte plus long, jusqu'à 4 000 caractères, est
  découpé et envoyé morceau par morceau ; tu peux l'annuler en cours d'envoi.
- **Indépendance vis-à-vis des ordres.** Envoyer un message ne soumet rien et
  ne change pas ta soumission : tes ordres restent à soumettre séparément.

---

## Annexe A — Aide-mémoire des ordres

`XXX`, `YYY`, `ZZZ` désignent des territoires ; `NNN`, `HHH` des nobles ;
`CCC` une carte. Les ordres d'armée sont gratuits.

**Chaînes d'armée** (printemps, été, automne)

| Syntaxe | Ordre |
|---|---|
| `XXX A YYY` | Attaque ou déplacement vers une case adjacente |
| `XXX S YYY` | Soutien défensif de l'armée sur `YYY` |
| `XXX S YYY - ZZZ` | Soutien offensif de l'attaque `YYY` → `ZZZ` |
| `H XXX` | Maintien |
| `XXX J YYY` | Jonction pacifique (dernier ordre) |
| `P XXX` | Pillage |
| `XXX D DEST1 DEST2 …` | Dispersion (`*` ou `*NNN` après une destination y envoie des nobles) |
| `XXX T YYY N` | Transfert de `N` R |
| `( … )` | Ligne en boucle, retentée jusqu'à sa réussite |

**Cartes de noble** (toute saison)

| Syntaxe | Ordre |
|---|---|
| `R N CCC XXX` | Recruter le noble de la carte `CCC` sur `XXX` |
| `C N HHH CCC` | Prétention de `HHH` sur les titres de `CCC` (carte `CLM`) |
| `D N NNN CCC` | Conférer à `NNN` la dignité `CCC` (`ABB` prend en plus une région) |

**Cartes spéciales** (printemps, été, automne)

| Syntaxe | Carte |
|---|---|
| `P BT XXX` · `P RA XXX` | Beau temps · Bonne récolte, sur le chef-lieu d'une région |
| `P RE TER` | Révolte |
| `P TX XXX` · `P TX HHH XXX` · `P TX KKK XXX` | Taxe seigneuriale · par le titulaire · taxe royale |
| `P DI HHH XXX` · `P DI HHH` | Dîme d'un évêché · dîme du pape |
| `P AG HHH TER` · `P AP HHH TER` | Apaisement : rite gratuit · payant |
| `P SO NNN` · `P SO AAA - BBB` | Souterrain : évasion · attaque sous les murailles |
| `P EM AAA BBB` | Embuscade |
| `P AS HHH NNN` | Assassinat (aussi en hiver) |
| `P JU HHH` | Justice (aussi en hiver) |

---

## Annexe B — Ordres d'hiver

| Syntaxe | Ordre | Condition principale | Coût |
|---|---|---|---|
| `T N` | Piocher une carte de noble | une fois par hiver, main non pleine | 0 |
| `D C CCC` | Défausser une carte de ta main | — | 0 |
| `R N CCC XXX` | Recruter un noble | château ou village à toi, avec une de tes armées | 0 |
| `C N HHH CCC` | Prétention | carte `CLM` ; `HHH` né d'un mariage avec `CCC` | 0 |
| `D N NNN CCC` | Conférer une dignité | carte en main | 0 |
| `R T XXX` | Recruter une troupe | un de tes nobles libres sur `XXX` ou à côté | {{costs.troop}} |
| `C M XXX` | Moulin : construire ou améliorer | nouveau : case vide voisine d'un château ou d'un village ; niveau 3 au plus | {{costs.mill_levels.0}} / {{costs.mill_levels.1}} / {{costs.mill_levels.2}} |
| `C C XXX` | Château, ou fortifier un village | — | {{costs.castle}} |
| `C D XXX` | Dépôt de vivres | — | {{costs.supply_depot}} |
| `E C XXX` | Désigner ta capitale | un de tes châteaux | 0 |
| `G XXX YYY N` | Envoyer des R à un autre joueur | de ton château ou village vers le sien | 0 |
| `O N NNN` · `P N NNN` | Prisonnier en otage · au donjon | un prisonnier que tu détiens | 0 |
| `H N NNN XXX [O\|P]` | Remettre un noble à l'armée d'un autre joueur | ton noble, ou un prisonnier que tu détiens | 0 |
| `T F NNN XXX YYY ZZZ…` | Fonder un fief | 3 territoires contigus ou plus, à toi, capitale avec château | {{costs.fief_per_territory}} / territoire |
| `T A NNN XXX` | Attribuer un fief vacant | — | 0 |
| `T X NNN XXX YYY` | Annexer `YYY` au fief `XXX` | `YYY` voisin, tenu par ton armée ; accord du roi s'il est dans un autre fief | {{costs.annex_per_territory}} |
| `T R NNN XXX` | Monter le fief d'un rang | une fois par hiver, pas au-delà du duché | {{costs.rank_per_territory}} / territoire |
| `M N NNN MMM` | Mariage | ordre réciproque de l'autre joueur | 0 |
| `W X` · `W R` · `W F` · `W G` | Ordres royaux | roi ou régente | 0 |
| `K R NNN` · `V R NNN` | Candidature et vote pour le trône | trône vacant | 0 |
| `K E NNN BBB` · `V E NNN BBB` | Candidature et vote à un évêché | évêché vacant | 0 |
| `K P NNN` · `V P NNN` | Candidature et vote au conclave | 2 cardinaux au moins | 0 |
| `N C NNN` | Promouvoir ton évêque cardinal | — | {{religion.cardinal_cost}} |
| `X E NNN` · `X L NNN` | Excommunier · lever l'excommunication | pape | 0 |
| `X D NNN` | Dissoudre un mariage | pape et demande d'un époux | 0 |
| `Q HHH NNN` | Enquête | cardinal ou pape, une par hiver | 1 à 5 |
| `J HHH NNN` | Procès | deux cardinaux sur la même cible | 0 |
| `V C NNN I` | Rayer une calamité annoncée | Astrologue | 0 |
| `S R NNN CAL` · `S R NNN N` | Rituel de la Sorcière | un seul rituel par hiver, tous joueurs confondus | 0 |

**Ordre de résolution.** Chaque étape voit le résultat des précédentes :

1. l'état est figé : un siège libéré cet hiver ne s'élit qu'au suivant ;
2. excommunications et levées ;
3. ordres royaux, puis gestion (investissements, achats de cardinal, cartes
   de noble) ;
4. enquêtes et dépôt des procès ;
5. dissolutions de mariage ;
6. mariages ;
7. élections : évêchés, conclave, puis trône ;
8. investiture des élus ;
9. fin d'hiver : mains, fiefs vacants, conservation et rapatriement des
   stocks, prospérité ;
10. jugement des procès, puis Assassinat et Justice.

Dans une étape, les joueurs passent dans l'ordre de leur numéro, et tes
ordres dans l'ordre de ta feuille.

---

## Annexe C — Cas particuliers

### Chaînes et ordres

- **Soutien offensif d'une de tes armées.** S'il ouvre la chaîne, ton armée
  soutenue doit avoir pour ordre en cours l'attaque annoncée.
- **Soutien défensif.** Toute armée qui ne quitte pas sa case le reçoit, y
  compris une armée en maintien, en soutien, en pillage ou Sans Ordre. Une
  armée qui attaque y renonce, même si son attaque échoue.
- **Jonction.** Si une de tes attaques gagne sur la case visée, la jonction
  fusionne avec l'armée victorieuse.
- **Auto-capture.** Une armée peut attaquer ton propre château vide pour s'y
  installer, sans que la défense du château la repousse.
- **Dispersion.** Les destinations sont adjacentes ou égales à la case de
  départ, et peuvent se répéter. Une destination tenue par l'ennemi ou
  disputée ne consomme pas de troupe. `*` envoie tous les nobles restants avec
  le groupe, `*NNN` un seul noble. Les nobles non affectés restent sur place
  tant qu'une troupe y demeure. Pour garder la chaîne sur place, liste
  d'abord la case de départ : `BRI D BRI ATL NOR`. En boucle, la dispersion
  retente jusqu'à remplir chaque destination.
- **Transfert.** Le destinataire est un château, un village, ou une armée
  d'un autre joueur qui contrôle sa propre case. Le montant est plafonné à
  `{{cost_base}}^(N − 1)` pour une armée de `N` troupes. En boucle, un stock
  insuffisant envoie ce qui reste et termine l'ordre.
- **Mauvais temps.** Un mouvement bloqué par le mauvais temps ne casse pas la
  chaîne : la ligne attend la saison suivante.

### Combat

- Une attaque sur une case tenue par ton armée échoue, mais bloque les autres
  attaques.
- Dans un face-à-face, les soutiens du joueur adverse ne comptent pas.
- Une attaque repoussée bloque encore sa destination, sauf si elle a perdu un
  face-à-face.
- Un château en ruine n'apporte aucune défense, même comme capitale de fief.

### Retraites

- À égalité dans une catégorie, la case la plus proche d'un de tes châteaux
  ou villages l'emporte, puis le trigramme le plus petit.
- Une armée hôte doit être à toi et ne pas avoir été délogée. Elle gagne
  `N − 1` troupes (ou 1 si `N = 1`).
- La case d'où vient l'attaquant est toujours exclue. Un château ancré, à
  n'importe quel joueur, bloque une retraite ; un château inerte ne la bloque
  pas.
- Deux armées sans autre issue que la même case vide sont détruites.

### Ravitaillement

- La réserve de tes sources sert d'abord les armées les plus proches, puis
  les plus petites. Une armée que la réserve ne couvre pas prend juste de quoi
  nourrir sa valeur `s`, et laisse le reste aux suivantes.
- Sans capitale, chaque territoire verse son revenu au château le plus
  proche, sinon au village le plus proche.
- Un moulin sans contrôleur produit sur sa propre case. L'armée qui le prend
  trouve ce stock.
- L'amélioration d'un moulin se paie d'abord sur son propre stock, puis sur
  celui de l'infrastructure qu'il alimente.
- En fin d'hiver, le stock d'un moulin rejoint l'infrastructure qu'il
  alimente. Les stocks hors château, village, moulin et dépôt sont perdus.
- **Prospérité** : chaque fondation rénove un village en ruine, sinon fonde
  un village sur la case libre la plus proche, sinon transforme un dépôt,
  sinon fait monter un village d'un niveau.

### Ruines et capitales

- `C C` sur un château en ruine le reconstruit. Sur une autre ruine, toute
  construction la remplace.
- Le château en ruine d'une capitale de fief reste la capitale : le fief
  tient, sans défense, et seul `C C` peut y être construit.
- Une armée affamée qui pille sa case peut faire tomber en ruine la capitale
  d'un fief.

### Nobles, titres et dignités

- **Bâtard** : il ne reçoit un nouveau fief que s'il est le dernier de ta
  lignée. Une carte de bâtard jouée sur un héritier annule son Claim.
- **Dignités** : leurs bonus sont suspendus au donjon. Une dame otage les
  garde, et son geôlier profite aussi des bonus passifs.
- **Chevalier d'Éon** : démasquée (par capture, enquête, ou Claim visant son
  mariage), elle est excommuniée d'office.
- **Sorcière** : `S R NNN CAL` appelle une calamité (`PE`, `MT` ou `FA`) sur
  sa région si elle est tirée ; `S R NNN N` fixe la saison de la première
  calamité tirée (1, 2 ou 3). Démasquée, elle est excommuniée.
- **Reine et régente** : la reine est l'épouse du roi. Elle garde son titre
  veuve, et le perd si elle se remarie. Régente, elle exerce les ordres
  royaux et la taxe royale.
- **Fief de la couronne** : un fief dont le joueur n'a plus de noble passe à
  la couronne, s'il y a un roi. Seul `W G` l'octroie.
- **Mariage et élections** : un noble marié ne peut pas devenir évêque ni
  pape. Il peut devenir roi.
- **Enquête sur un mort** : elle coûte le prix du titre qu'il portait à sa
  mort.

### Cartes

- Deux cartes identiques sur la même région : la première annule la
  calamité, la seconde applique le bonus. Les suivantes sont perdues.
- Deux taxes sur un même fief ne se cumulent pas ; la taxe du seigneur passe
  avant la taxe royale. Si la capitale taxée tombe pendant la saison, la taxe
  est annulée.
- Dîmes concurrentes : l'évêque passe avant les cardinaux, qui passent avant
  le pape. Plusieurs cardinaux se partagent la production.
- Révolte annulée : si une Bonne récolte annule la mauvaise récolte, les
  révoltes en attente sont annulées et la carte revient.
- Les armées neutres perdent une troupe chaque saison où leur case ne les
  nourrit pas.
- Souterrain (évasion) : le noble réapparaît libre sur ta capitale, sinon sur
  ton plus petit château.
- Embuscade : le noble capturé est le premier de la ligne de succession
  parmi ceux de l'armée vaincue.
- Ordre de résolution des cartes : Souterrain, puis Assassinat, puis Justice.

---

## Annexe D — Un tour joué, de bout en bout

**La situation.** Hugues tient ROS, sa capitale, avec 2 troupes et son noble
HUG, et FOU, avec 1 troupe et son noble ODA. Brune tient ATL avec 2 troupes et
son noble MIA. NOR est vide.

<svg viewBox="0 0 540 280" width="100%" role="img" aria-label="ROS et FOU (Hugues) sont adjacents à ATL (Brune), elle-même adjacente à NOR (sans contrôleur, vide)" style="max-width:480px;margin:16px auto;display:block;font-family:system-ui,sans-serif">
  <line x1="90" y1="70" x2="300" y2="130" stroke="#b7a786" stroke-width="2"/>
  <line x1="90" y1="190" x2="300" y2="130" stroke="#b7a786" stroke-width="2"/>
  <line x1="300" y1="130" x2="460" y2="130" stroke="#b7a786" stroke-width="2"/>
  <circle cx="90" cy="70" r="30" fill="#f3ead9" stroke="#3f6b52" stroke-width="3"/>
  <text x="90" y="76" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">ROS</text>
  <text x="90" y="114" text-anchor="middle" font-size="11" fill="#3f6b52">capitale de Hugues</text>
  <text x="90" y="128" text-anchor="middle" font-size="11" fill="#594b3c">2 troupes · HUG</text>
  <circle cx="90" cy="190" r="30" fill="#f3ead9" stroke="#3f6b52" stroke-width="3"/>
  <text x="90" y="196" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">FOU</text>
  <text x="90" y="234" text-anchor="middle" font-size="11" fill="#3f6b52">garnison de Hugues</text>
  <text x="90" y="248" text-anchor="middle" font-size="11" fill="#594b3c">1 troupe · ODA</text>
  <circle cx="300" cy="130" r="30" fill="#f3ead9" stroke="#3a5a8c" stroke-width="3"/>
  <text x="300" y="136" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">ATL</text>
  <text x="300" y="174" text-anchor="middle" font-size="11" fill="#3a5a8c">tenu par Brune</text>
  <text x="300" y="188" text-anchor="middle" font-size="11" fill="#594b3c">2 troupes · MIA</text>
  <circle cx="460" cy="130" r="30" fill="#f8f0e2" stroke="#3a5a8c" stroke-width="2" stroke-dasharray="4 3"/>
  <text x="460" y="136" text-anchor="middle" font-size="16" font-weight="700" fill="#30291f">NOR</text>
  <text x="460" y="174" text-anchor="middle" font-size="11" fill="#3a5a8c">sans contrôleur</text>
  <text x="460" y="188" text-anchor="middle" font-size="11" fill="#594b3c">vide</text>
</svg>

**Ce que Hugues écrit.** Ses deux nobles émettent chacun une chaîne :

```text
HUG
ROS A ATL        # attaque ATL depuis ROS
```

```text
ODA
FOU S ROS - ATL  # soutient l'attaque ROS -> ATL
```

**Ce que Brune écrit.** Rien : son armée est Sans Ordre, mais elle se défend
normalement.

**La résolution.** L'attaque pèse 2 + 1 de soutien = 3, contre 2 en défense.
Les bonus de noble s'appliquent des deux côtés et ne changent rien. Hugues
entre dans ATL. L'armée de Brune recule vers NOR, case vide et non attaquée,
et MIA la suit.

**Le rapport.** Brune, qui a combattu, voit le détail : 3 contre 2, avec le
soutien de FOU. Un joueur extérieur voit seulement qu'ATL a changé de mains.

---

*Crown & Borough s'inspire de [Fief](https://boardgamegeek.com/boardgame/107704/fief)
pour son cadre féodal et de [Diplomacy](https://boardgamegeek.com/boardgame/483/diplomacy)
pour ses ordres simultanés et ses soutiens.*
