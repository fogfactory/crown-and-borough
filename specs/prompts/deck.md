# Prompt : deck d'ordres speciaux, regions et calamites

```
CONTEXTE ET DECISIONS FIGEES :
- L'issue GitHub concernee est l'issue #15 :
  https://github.com/fogfactory/crown-and-borough/issues/15
- Le moteur Go est pur et deterministe. Les resolvers principaux sont
  `Resolve`, `ResolveWinter`, `ResolveTurn` et `CreateGame`.
- `ResolveTurn` prepare les soumissions, attache les chaines, appelle le
  resolver de saison, puis avance le calendrier. `ResolveWinter` ne doit pas
  avancer le calendrier lui-meme.
- `GameState.Validate` porte les invariants structurels. Les contraintes qui
  dependent de la balance sont validees par l'asset loader ou le moteur.
- `balance.yaml` est charge avec `KnownFields(true)`. Toute nouvelle cle doit
  etre declaree, validee et couverte par un test.
- `MapData` est la carte statique publique ; `StateView` est la projection de
  l'etat dynamique. Les regions geographiques sont publiques dans `MapData`.
- Les regles de confidentialite existantes restent valables. Une main, la
  pioche et la defausse ne doivent pas etre exposees a un autre joueur.

REGLES DE JEU RETENUES :
- Le deck est commun a toute la partie.
- Le deck contient uniquement trois calamites et trois cartes bonus : peste,
  mauvais temps, famine, beau temps, bonne recolte et revolte. La revolte est une
  carte jouable uniquement lorsqu'une famine active affecte la region cible.
- Les cartes impôt, mariage, assassinat, cardinal et Claim ne sont pas encore
  generees dans le deck. Leur emplacement futur doit rester possible sans
  melanger leurs regles dans cette implementation.
- Une carte est un droit de jouer un ordre special. Les cartes ne sont pas des
  ordres executables autonomes.
- La main d'un joueur ne contient que des cartes bonus. Une calamite piochee
  ne rejoint jamais une main.
- La limite initiale de main est 4 et le nombre maximal initial de commandes de
  pioche est 2 par joueur et par hiver. Ces deux valeurs sont dans la balance.
- En hiver, les joueurs peuvent abandonner des cartes et ecrire des commandes
  `T C` pour tirer. Chaque `T C` tire une carte bonus utile ou continue a tirer
  apres avoir traite une calamite. Une feuille ne peut contenir que deux `T C`.
- Si la main est deja pleine au moment d'un `T C`, la commande est un no-op
  silencieux mais elle compte comme une utilisation de pioche.
- Si une calamite est tiree, elle est programmee dans le premier slot libre de
  l'annee suivante selon l'ordre printemps, ete, hiver. Elle est retiree de la
  pioche, ne va pas dans la main et le tirage continue pour obtenir le droit
  special demande par `T C`.
- Si tous les slots de l'annee suivante sont pleins, la calamite tiree est
  defaussee et le tirage continue. La defausse complete peut donc contenir des
  calamites excedentaires et celles-ci peuvent ressortir lors d'un remelange.
- Une calamite programmee reste hors de la pioche et de la defausse jusqu'a sa
  resolution. Elle est ensuite defaussee.
- La balance ne declare pas une cle `calamities_per_year`. Elle declare une
  capacite de slot pour le printemps, l'ete et l'hiver. La configuration
  initiale est un slot par saison.
- Les calamites de l'annee sont revelees publiquement au printemps, avec leur
  saison d'application et leur region. La calamite du printemps est resolue
  pendant ce tour ; les calamites d'ete et d'hiver restent visibles jusqu'a
  leur tour.
- Un ordre bonus est soumis dans le champ de joueur `special`, independamment
  des chaines de nobles et des investissements d'hiver. Il est autorise pendant
  le printemps, l'ete, l'automne et l'hiver.
- Les cartes sont consommees avant la resolution simultanee des ordres d'armee.
  Leurs effets regionaux sont agreges puis appliques avant le ravitaillement et
  l'enumeration des intentions d'armee.
- Un ordre bonus special est un ordre de joueur, pas une ligne de chaine de
  noble. Le symbole `P` signifie deja pillage dans une chaine ; il ne faut pas
  surcharger le parseur de chaine avec `P BT TER`.
- `BT` / `FW` (beau temps / fair weather) annule uniquement le mauvais temps.
- `RA` / `AH` (recolte abondante / abundant harvest) annule uniquement la
  famine.
- Beau temps et bonne recolte donnent le meme bonus regional : +1 R par moulin
  concerne et +1 ration produite pour les armees concernees. Les deux types
  peuvent donc cumuler ce bonus lorsqu'ils sont joues dans la meme region.
- Deux cartes du meme type ne cumulent pas. La premiere carte de ce type est
  la seule carte effective ; les cartes supplementaires sont consommees mais
  deviennent inoperantes.
- Interpretation de resolution a conserver dans les tests : lorsqu'une carte
  bonus sert a annuler sa calamite correspondante, elle annule la calamite mais
  ne fournit pas son unite de bonus. Si sa calamite correspondante est absente,
  elle fournit son unite de bonus. Les unites effectives de BT et RA
  s'additionnent. Ainsi, BT + RA dans une region sans calamite donnent deux
  unites de bonus, tandis que deux BT n'en donnent qu'une.
- Le bonus et l'annulation sont regionaux et s'appliquent a toute la region,
  quel que soit le joueur controleur. Une carte jouee sur une autre region ne
  neutralise rien dans la region de la calamite.
- Les regions sont calculees par BFS multi-source depuis les N+1 villages
  neutres de la carte. Le village qui a servi de seed est la reference visible
  dans un ordre `P ... TER`.
- La peste est rare : la valeur initiale proposee est un poids 1 parmi 13 pour
  les calamites, avec `plague: 1`, `bad_weather: 6` et `famine: 6`. Les poids
  restent reglables dans la balance.
- Une armee de revolte porte l'owner reserve `NEUTRAL`. Ce reserve n'est pas
  un slot de joueur, ne figure pas dans `GameState.Players`, n'apparait pas au
  scoreboard et ne soumet jamais de commandes.

REGLE DE CODE :
- Les identifiants Go/TypeScript, enums, codes d'erreur, commentaires et
  messages techniques restent en anglais, comme dans le code existant.
- Les valeurs enum du contrat JSON restent stables et en anglais, par exemple
  `fair_weather` ou `bad_weather`.
- Les labels, noms de cartes, aides de saisie et messages utilisateur sont
  localises en francais et en anglais via les catalogues existants.
- Le parseur accepte les alias francais et anglais quelle que soit la langue
  de l'interface. La langue choisie sert a rendre le message d'erreur, pas a
  changer les donnees stockees ni la seed de la partie.
- Ne pas ajouter de logique Firestore ou de logique moteur dans React.
- Ne pas commit, push ou ouvrir de pull request sans instruction explicite.

PERIMETRE :
Implementer le deck commun, la main, la defausse, la pioche hivernale, les
slots de calamites, le decoupage regional, les deux ordres bonus, les quatre
calamites et les tests existants du socle.

1. MODELE METIER :
   - Ajouter dans `internal/models` des types explicites pour les identifiants
     de cartes et de regions ainsi qu'un enum unique de kind de carte, ou une
     decomposition equivalente clairement validee :
     `fair_weather`, `abundant_harvest`, `plague`, `bad_weather`, `revolt` et
     `famine`.
   - Fournir des methodes indiquant si un kind est un bonus ou une calamite et
     quel kind de calamite un bonus peut annuler.
   - Ajouter une structure statique de region avec au minimum un identifiant,
     le territoire seed et la liste triee des territoires.
   - Ajouter une structure de carte avec un `SpecialCardID` interne et son
     kind. Le code de carte interne n'est jamais utilise dans la syntaxe
     utilisateur : les ordres ciblent un kind (`BT`, `RA`, `FW` ou `AH`).
   - Ajouter un etat de deck contenant le catalogue des cartes de la partie,
     la draw pile, la defausse et les mains par `PlayerID`. Utiliser des IDs
     internes dans les piles et les mains pour qu'une carte n'existe qu'a un
     seul endroit.
   - Ajouter une calamite programmee avec sa carte, son kind, son annee, sa
     saison et le seed de region cible. Elle n'a pas de proprietaire de
     gameplay : le joueur qui l'a tiree ne peut ni la controler ni la modifier.
   - Ajouter une augure annuelle avec trois capacites de slots indexees par
     saison (printemps, ete, hiver), un indicateur de revelation et les
     calamites programmees. Ne pas remplacer cette representation par une
     simple limite annuelle.
   - Ajouter l'etat du deck/augures/regions a `GameState`. Preferer un bloc
     `SpecialDeck` optionnel ou une migration equivalente pour que les anciens
     etats de test sans cartes restent lisibles, mais tout etat cree par
     `CreateGame` doit initialiser le systeme complet.
   - Ne pas ajouter `NEUTRAL` a la liste des joueurs humains. Autoriser cet
     owner reserve uniquement pour les armees neutres, et refuser une chaine,
     une emission ou une soumission rattachee a cet owner.
   - Etendre `GameState.Validate` pour verifier les references, l'unicite et
     la localisation de chaque carte : draw pile, defausse, main ou calamite
     programmee, mais jamais deux emplacements a la fois.
   - Verifier que les mains ne contiennent que des bonus et que les augures
     n'occupent pas plus de capacite que les slots de leur saison. La limite
     numerique de main doit rester une validation moteur dependant de la
     balance si `Validate` ne recoit pas de balance.
   - Verifier qu'une armee `NEUTRAL` n'est pas comptee comme joueur vivant ou
     dans un score. Elle reste une armee de position : elle ne recoit pas de
     chaine, ne produit pas de soumission, peut defendre et peut etre attaquee.

2. BALANCE ET GENERATION DU DECK :
   - Etendre `assetgen.Balance`, `rawBalance`, `LoadBalance` et les tests du
     loader avec une section dediee, par exemple :

     ```yaml
     special_orders:
       hand_limit: 4
       draw_orders_limit: 2
       deck_size: 30
       calamity_percentage: 30
       calamity_slots:
         spring: 1
         summer: 1
         winter: 1
        calamity_weights:
          plague: 1
          bad_weather: 6
          famine: 6
        bonus_weights:
          fair_weather: 3
          abundant_harvest: 3
          revolt: 1
       effects:
         plague_army_divisor: 2
         plague_noble_mortality_percentage: 50
         revolt_army_count: 3
         revolt_army_min_size: 2
         revolt_army_max_size: 3
         bonus_mill_production: 1
         bonus_army_ration: 1
     ```

   - Les valeurs ci-dessus sont les valeurs initiales de la balance, sauf
     decision contraire dans la revue. Elles ne doivent pas etre recopiees
     dans les handlers ou le frontend.
   - Ne pas declarer `calamities_per_year`. La capacite est lue dans les trois
     slots de saison. Le loader doit exiger les saisons spring, summer et
     winter, et refuser autumn dans cette section.
   - Utiliser un pourcentage entier et verifier que la taille de deck permet de
     calculer un nombre entier de calamites. Refuser une configuration dont le
     produit `deck_size * calamity_percentage` n'est pas entier en pourcentage.
   - Calculer les quantites par kind a partir des poids relatifs. Si une
     repartition entiere necessite un arrondi, employer une methode deterministe
     de plus grands restes et departager les restes par ordre canonique des
     kinds. Tester explicitement que `plague` reste rare avec la configuration
     initiale.
   - Rejeter les poids negatifs, les poids tous nuls, les bornes de revolte
     invalides et un pourcentage hors de 0..100.
   - Construire le deck une seule fois dans `CreateGame`, avec des IDs
     internes stables et une liste initiale deterministe avant melange.
   - Ne jamais ajouter les cartes placeholder au deck par defaut.

3. REGIONS ET CARTE PUBLIQUE :
   - Ajouter `regions` a `mapgen.MapData` et a l'etat cree par `CreateGame`.
     La carte HTTP expose la partition statique a tous les joueurs.
   - Calculer les regions apres la generation des identifiants de territoires,
     des frontieres franchissables et des villages. Les seeds sont les
     territoires marques `Village` dans `mapgen.MapData`, et non une recherche
     dynamique d'infrastructures apres le debut de partie.
   - Trier les seeds par trigramme de territoire. Executer un BFS multi-source
     sur les adjacences franchissables. A distance egale, attribuer le
     territoire au seed dont le trigramme est le plus petit.
   - Garantir que chaque seed est dans sa region, que les regions sont
     disjointes, qu'elles couvrent tous les territoires et que chaque region
     est connexe dans le graphe franchissable.
   - Utiliser le seed comme reference stable de region dans les ordres. Un
     identifiant interne `RegionID` peut etre derive du seed, mais ne pas
     obliger le joueur a connaitre un identifiant artificiel `R1`.
   - Conserver une structure suffisamment generale pour que `religieux.md`
     puisse reutiliser exactement cette partition pour les futurs eveches.
   - Ajouter des tests de determinisme, de couverture, de connexite, de tie
     break et de presence des N+1 villages.

4. SYNTAXE DES ORDRES SPECIAUX :
    - Ne pas modifier la grammaire des chaines de nobles pour y inserer les
      ordres bonus. Ajouter une soumission de joueur `special`, dediee aux
      ordres du deck et distincte des chaines et investissements d'hiver.
      `P KIND TER` est autorise au printemps, en ete et en automne ; `D C KIND`
      et `T C` sont autorises uniquement en hiver.
   - Syntaxe francaise canonique :

     ```text
     D C BT       # abandonner une carte Beau temps
     D C RA       # abandonner une carte Recolte abondante
     T C          # tirer une carte, au plus deux fois dans la feuille
     P BT ROS     # jouer Beau temps sur l'eveche dont ROS est le village seed
     P RA ROS     # jouer Recolte abondante sur la meme region
     ```

   - `D C <KIND>` defausse une carte du kind demande. Si plusieurs cartes du
     meme kind sont en main, prendre la premiere dans l'ordre de la main, qui
     est l'ordre de pioche conserve par le serveur. La carte consommee part
     dans la defausse.
   - `T C` ne prend aucun argument et tire une carte utile. Une calamite
     declenche sa programmation puis le tirage continue dans la meme commande
     jusqu'a obtenir un bonus ou jusqu'a epuisement deterministe des cartes
     disponibles. Il ne s'agit pas d'un troisieme ordre de pioche.
   - Une troisieme occurrence de `T C` dans la feuille d'un joueur doit
     produire une erreur de soumission localisee et aucun effet partiel. Les
     deux occurrences autorisees sont comptees meme si une main pleine les
     transforme en no-op.
   - `P <KIND> <TER>` consomme une carte du kind correspondant et programme
     l'effet sur la region dont `TER` est le village seed initial. `TER` n'est
     pas un territoire arbitraire de la region : il doit etre le seed d'une
     region.
   - Un `P` sans carte correspondante est une erreur `no_card_for_kind` et
     invalide atomiquement la soumission concernee. Une carte consommee pour
     un ordre valide est defaussee meme si elle est inoperante a cause d'un
     doublon du meme kind.
   - Proposer et accepter les alias anglais suivants sans changer le contrat
     interne :

     | Concept | Francais | Anglais |
     |---|---|---|
      | Abandonner une carte | `D C <KIND>` | `D C <KIND>` (Discard Card) |
      | Tirer une carte | `T C` | `T C` (Take Card) |
      | Jouer une carte | `P ...` ou `J ...` | `P ...` ou `J ...` (Play) |
     | Beau temps | `BT` | `FW` (Fair Weather) |
     | Recolte abondante | `RA` | `AH` (Abundant Harvest) |
     | Peste, affichage | `PE` | `PL` (Plague) |
     | Mauvais temps, affichage | `MT` | `BW` (Bad Weather) |
     | Revolte, affichage | `RE` | `RV` (Revolt) |
     | Famine, affichage | `FA` | `FN` (Famine) |

    - Le parseur reconnait les formes FR et EN, en majuscules ou minuscules
      apres la normalisation existante. `D C <KIND>` signifie toujours
      defausser une carte, tandis que `T C` signifie toujours tirer. `P` et `J`
      sont acceptes pour jouer une carte ; `J` reste une jonction uniquement
      dans la grammaire des chaines.
   - Ajouter des messages a `internal/i18n/catalog.go` et aux catalogues
     frontend pour la forme, le kind inconnu, le seed inconnu, l'absence de
     carte, la limite de pioche et l'ordre special interdit dans la saison.
     Garder les codes techniques en anglais et localiser uniquement les
     messages.
   - Ajouter des tests parser pour chaque alias, les commentaires, les lignes
     vides, les arites invalides, le conflit `P N NNN` existant, le troisieme
     `T C` et les kinds calamite refuses dans `P`.

5. ETAT DE SOUMISSION ET RESOLUTION HIVERNALE :
    - Ajouter a `engine.OrdersInput` une soumission de joueur `special` pour
      les ordres du deck, independante des chaines de nobles et des
      investissements d'hiver. Elle est disponible a chaque saison et transite
      par l'API et le store.
    - La soumission `winter` conserve uniquement les investissements actuels.
      Les ordres du deck sont soumis dans le champ `special`, quelle que soit
      la saison.
    -       `P KIND TER` est traite avant les ordres d'armee au printemps, en ete et
      en automne. `D C KIND` et `T C` ne sont traites qu'en hiver.
    - Les lignes de chaque soumission `special` sont traitees dans l'ordre
      textuel, et les joueurs sont traites selon l'ordre deterministe du moteur.
      La consommation est validee avant la publication du nouvel etat.
   - Maintenir l'atomicite : parser toutes les lignes, verifier le nombre de
     `T C`, les joueurs, les seeds et les preconditions de cartes avant de
     publier le nouvel etat. Une erreur ne doit ni consommer de carte ni
     modifier la pioche, la defausse ou les investissements.
   - Resoudre la calamite du slot winter de l'annee courante dans
     `ResolveWinter`, meme si l'hiver ne resout aucun mouvement, combat ou
     ravitaillement. Le fait qu'un effet n'ait aucune cible militaire en hiver
     ne doit pas supprimer l'evenement ni la slot de calamite.
    - Appliquer les effets de saison et les bonus avant la phase a laquelle ils
      participent. Les ordres `P` sont consommes puis agreges avant le
      ravitaillement et l'enumeration des intentions d'armee. Pour l'hiver,
      conserver l'ordre des investissements et de la conservation des stocks
      deja documente, tout en faisant apparaitre les evenements de calamite et
      de carte dans le rapport d'hiver.
   - Si un bonus est joue en hiver, il interagit avec la calamite du slot winter
     de l'annee courante, pas avec la calamite qui vient d'etre programmee pour
     l'hiver suivant.
   - Ne pas faire avancer `Turn` dans `ResolveWinter`. L'avancement reste dans
     `ResolveTurn` comme aujourd'hui.

6. DECK, PIOCHE ET AUGURES :
   - Ajouter un module moteur dedie a la construction, au melange, a la pioche,
     a la defausse et a la programmation des calamites.
   - Utiliser des RNG derives par domaine, par exemple avec SHA-256 sur la
     seed, le tour, la phase et un compteur de remelange. Ne pas utiliser une
     source aleatoire globale partagee par la carte, les nobles et les effets.
   - Le premier melange doit etre reproductible depuis la seed de partie. Un
     remelange doit etre reproductible depuis la seed, le tour et le numero de
     remelange de la resolution. Le resultat doit dependre de l'etat de la
     pioche et de la defausse, pas de l'ordre d'iteration d'une map Go.
   - Lorsqu'une pioche est vide, melanger la defausse complete, y compris les
     calamites excedentaires, puis la replacer comme nouvelle pioche. Les
     calamites encore programmees ne sont jamais remelangees.
   - Pour chaque `T C`, si la main n'est pas pleine, tirer jusqu'a rencontrer
     un bonus ou jusqu'a constater deterministement qu'aucun bonus ne peut etre
     obtenu dans le cycle de cartes disponible. Cette borne de cycle evite une
     boucle infinie si la defausse ne contient que des calamites ; elle ne
     limite pas artificiellement le nombre de calamites traitees.
   - Programmer les calamites dans `YearAugury(year+1)` au premier slot libre
     parmi spring, summer et winter, en respectant les capacites de balance.
     Si aucun slot n'est disponible, envoyer la carte en defausse et produire
     un evenement interne tracable sans la rendre publique avant l'augure.
   - Au printemps, marquer l'augure de l'annee comme revelee et produire une
     liste publique ordonnee par saison. Les augures futures restent absentes
     de toutes les projections joueur.
   - Ne pas rattacher une calamite a la main ou au joueur qui l'a tiree. La
     provenance eventuelle est une information d'audit privee uniquement.

7. EFFETS DES CALAMITES :
   - Ajouter des fonctions pures ou des phases moteur clairement separees pour
     charger les calamites de la saison courante, appliquer les bonus, puis
     resoudre les phases existantes.
   - Peste :
     - prendre les armees presentes dans la region au debut de la saison ;
     - remplacer chaque taille `N` par `ceil(N / plague_army_divisor)` sans
       jamais descendre sous une troupe ;
     - inclure les armees `NEUTRAL` dans cet effet ;
     - effectuer un seul tirage deterministe par noble present dans la region
       au debut de la saison, avec une probabilite issue de la balance ;
     - representer une issue mortelle dans le modele v1 par le statut
       `dungeon`, en conservant le noble a sa position. Produire un evenement
       explicite et ne pas inventer une nouvelle regle de succession.
     - appliquer la reduction de taille avant le ravitaillement et les combats,
       et traiter la mortalite noble en fin de saison pour ne pas dependre des
       mouvements resolus pendant cette saison.
   - Mauvais temps :
     - marquer la region comme affectee avant l'enumeration des intentions ;
     - bloquer tout deplacement provenant d'une armee situee dans cette
       region : attaque, jonction et dispersion ;
     - ne laisser passer que le maintien et le soutien defensif ; un soutien
       offensif et le pillage sont refuses ;
     - ne pas bloquer automatiquement une armee situee hors de la region qui
       entre dans cette region, sauf si une regle explicite ulterieure le
       demande ;
     - en hiver, produire l'evenement et l'etat de calamite meme si le socle ne
       resout deja aucun mouvement.
   - Revolte :
     - choisir de facon deterministe le nombre de territoires eligible parmi
       les cases vides de la region, sans remplacer une armee existante ;
     - la configuration initiale vise trois armees de deux a trois troupes,
       avec bornes dans la balance ; si moins de cases sont disponibles,
       produire autant d'armees que possible ;
     - allouer chaque ID avec `resolutionContext.allocateArmyID` ;
     - creer les armees avec `OwnerID = models.NeutralPlayerID`, sans noble,
       sans chaine et sans soumission ;
     - conserver le controle territorial existant a la creation et laisser les
       regles normales de combat traiter ensuite l'armee neutre ;
     - emettre un evenement public par apparition et ne pas les inclure dans
       les scores humains.
   - Famine :
     - affecter toutes les armees presentes dans la region, quel que soit leur
       owner ;
     - annuler la contribution des moulins situes dans la region a la
       production stockable ; la production de base reste active ;
     - desactiver le bonus de rations des chateaux/villages dans la region ;
       les rations de terrain restent actives ;
     - faire porter les modifications sur les fonctions existantes
       `sourceProduction` et `rationProduction` plutot que de dupliquer la
       logique d'approvisionnement ;
     - en hiver, produire l'evenement et conserver la resolution de la
       calamite, meme si aucune phase de ravitaillement n'est executee.

8. RESOLUTION DES ORDRES BONUS :
   - Parser les ordres bonus dans une structure de joueur distincte des
     `models.Order` de chaine et des investissements d'hiver.
   - Avant la resolution simultanee, verifier et consommer les cartes presentes
     dans la main du joueur. Les cartes consommees partent dans la defausse.
   - Agreger les ordres par `(season, region seed, bonus kind)` avant d'appliquer
     un effet. Ne pas laisser l'ordre d'iteration des joueurs decider du
     gagnant ou de l'annulation.
   - Pour chaque region :
     - au plus une BT/FW est effective ; les suivantes sont inoperantes ;
     - au plus une RA/AH est effective ; les suivantes sont inoperantes ;
     - une BT effective annule uniquement `bad_weather` ;
     - une RA effective annule uniquement `famine` ;
     - une carte qui annule sa calamite ne produit pas son unite de bonus ;
     - une carte dont la calamite correspondante est absente produit une unite
       de bonus ;
     - les unites de BT et RA s'additionnent : +1 par moulin et +1 ration par
       unite effective, pour toute la region et tous les joueurs.
    - La calamite peste n'est jamais annulee par BT ou RA.
    - La carte revolte est un ordre joueur bonus : elle exige une famine active
      dans la region cible et produit l'effet de revolte selon la balance.
   - Une carte jouee sur une region differente ne modifie ni l'annulation ni le
     bonus de la region de la calamite.
   - Produire un resultat public de l'ordre bonus : kind, seed de region,
     outcome, annulation eventuelle et bonus effectif. Ne jamais exposer la
     main ou l'identifiant interne de la carte aux autres joueurs.

9. EVENEMENTS ET RAPPORTS :
   - Etendre `internal/engine/events.go` avec des types distincts pour :
     pioche bonus, defausse, calamite programmee, augure revelee, ordre bonus
     joue, calamite appliquee, calamite annulee, effet de bonus, apparition
     d'armee neutre et resultat de mortalite de peste.
   - Ne pas reutiliser `EventTypeFamine` pour une calamite famine : cet event
     decrit deja la famine de ravitaillement d'une armee. Employer un type
     distinct pour ne pas casser les rapports existants.
   - Ajouter aux events les champs metier necessaires : kind de carte, kind de
     calamite, kind de bonus, seed de region, saison, annee, taille avant/apres,
     armee neutre, noble concerne et raison.
   - Faire evoluer `WinterReport` en conservant les sections existantes et en
     ajoutant une section `Cards` pour les operations de carte. Les
     investissements et les stocks existants ne doivent pas changer de forme.
   - Ajouter a `TurnReport` une section publique `Augury` pour la revelation
     du printemps et une section `SeasonEffects` pour les calamites, annulations
     et bonus de la saison courante.
   - Ajouter une section d'activite de cartes filtrable par joueur. Les
     operations `D C`, `T C`, les cartes en main et les kinds tires doivent
     rester visibles seulement au joueur concerne. Les effets publics peuvent
     etre affiches a tous.
   - Etendre `BuildTurnReport`, `TurnReportView` et `projectReport` sans
     modifier les regles existantes de visibilite des chaines et des combats.

10. PROJECTIONS, API ET PERSISTANCE :
   - Ajouter les regions a `mapgen.MapData` afin que `GET /api/map` et
     `GET /api/games/{id}/map` les servent comme geographie commune.
   - Ajouter a `StateView` une main speciale privee contenant les kinds de
     bonus du joueur courant, sans exposer la draw pile, la defausse, les IDs
     internes des cartes ou les augures futures.
   - Exposer l'augure de l'annee courante uniquement lorsqu'elle est revelee au
     printemps. Elle reste publique pendant le reste de l'annee.
   - Conserver les donnees completes du deck, des mains et des augures dans le
     `GameState` canonique du store, mais ne jamais les placer dans une
     projection publique ou dans une erreur HTTP.
   - Faire transiter la nouvelle soumission `special` dans les structures
     `engine.OrdersInput`, `store.SubmitRequest`, `api` et les adaptateurs
     memory/Firestore. Une soumission d'action ne doit pas etre confondue avec
     une chaine de noble.
   - Conserver les verifications d'identite existantes : en production le
     joueur vient de l'acteur authentifie, jamais d'un `player` arbitraire
     fourni par le navigateur.
   - Mettre a jour la version de schema de persistance si le projet en impose
     une pour les nouveaux champs. Une restauration doit retrouver exactement
     la draw pile, la defausse, les mains, les regions et les augures.
   - Ajouter des tests de projection : P1 voit sa main, P2 ne la voit pas, les
     augures revelees sont communes, les augures futures et la defausse restent
     cachees.

11. FRONTEND ET AFFICHAGE DE LA CARTE :
   - Etendre `web/src/types.ts` avec les regions, kinds de cartes, main privee,
     augure et effets de saison. Les enums JSON restent en anglais et les
     labels passent par `messages.ts`.
   - Afficher les cartes dans l'interface, sans les traiter comme des ordres
     automatiques. La main doit montrer le nom localise du bonus, le code FR/EN
     utilisable et les lignes d'ordre possibles.
    - Integrer ce panneau a `OrdersPanel` ou dans un composant dedie reutilise
      par `App` et `online/GamePage`. Afficher `P KIND TER` toute l'annee,
      et `D C`/`T C` uniquement en hiver.
   - Les controles UI peuvent inserer une ligne dans le brouillon de texte,
     mais l'API et le parseur restent l'autorite. Ne pas executer une carte
     directement depuis un clic frontend.
   - Ajouter l'etat du brouillon special dans `App.tsx` et `online/GamePage.tsx`,
     le reinitialiser au changement de tour et l'envoyer avec les commandes.
   - Ajouter dans `MapLegend` un toggle controle par le parent pour la couche
     `Regions`. Le composant actuel est present dans le hotseat et le parcours
     online : les deux doivent conserver le meme comportement.
   - Dans `MapViewer` :
     - construire une table territoire -> region depuis `map.regions` ;
     - tracer les segments de contour d'une region lorsqu'une arete touche le
       bord de la carte ou un territoire d'une autre region ;
     - ne pas recalculer une enveloppe geometrique fragile : reutiliser les
       aretes partagees et les aretes exterieures deja derivees par le viewer ;
     - afficher un contour pointille assez visible pour chaque eveche ;
     - attribuer une palette stable et discrete gris-bleu a chaque region ;
     - conserver la selection, les frontieres passables, la couche de
       ravitaillement et les marqueurs au-dessus ou au-dessous dans un ordre
       explicite ;
     - remplacer le contour colore de controle territorial par une hachure
       fine diagonale a 45 degres dans la couleur du joueur, clippee au
       polygone du territoire ;
     - ne pas laisser la hachure masquer la selection ou les frontieres de
       region.
   - Afficher les augures de deux manieres :
     - sur la carte, une mise en evidence et un tooltip listant le kind, la
       saison et le village seed de la region ;
     - dans `ReportPanel`, une section dediee listant les slots printemps, ete
       et hiver, y compris les slots sans calamite si cela rend la lecture plus
       claire.
   - Ajouter les traductions FR/EN pour les noms de cartes, les calamites, les
     regions, les slots, les effets, les alias de commandes et les etats de
     main vide/pleine. Le scoreboard ne doit jamais afficher `NEUTRAL`.
   - Conserver l'accessibilite existante : labels de couches, focus clavier,
     contraste des hachures et largeur mobile.

12. DOCUMENTATION DE REGLES :
   - Enrichir `specs/ordres-speciaux.md` avec la regle complete et les
     dependances. Documenter explicitement que les calamites sont tirees en
     hiver, programmees par slot saisonnier, revelees au printemps et
     resolues au tour de leur saison.
   - Documenter la distinction entre :
     - les lignes d'investissement d'hiver ;
      - `D C <KIND>` pour abandonner / discard ;
      - `T C` pour tirer / take ;
      - `P BT TER` et `P RA TER` pour jouer un bonus.
   - Preciser que `P BT TER` et `P RA TER` sont des ordres de joueur, pas des
     ordres dans une chaine de noble, et que `TER` est obligatoirement un
     village seed initial.
   - Ajouter la table des alias FR/EN, la limite de deux `T C`, la limite de
     main, la defausse sequentielle, le remelange et la poursuite du tirage
     apres une calamite.
   - Ajouter la matrice de resolution BT/RA : annulation ciblee, doublons non
     cumulatifs et bonus commun cumulable lorsque les deux kinds sont effectifs.
   - Mettre a jour `specs/religieux.md` pour declarer que les eveches futurs
     reutiliseront la partition regionale statique generee par la carte.
   - Mettre a jour `specs/gdd.md` et `specs/architecture.md` lorsque les
     contrats v1 ou les phases de saison changent effectivement.
   - Mettre a jour `assets/regles-joueurs.md` et
     `assets/regles-joueurs.en.md` avec des exemples jouables et les effets des
      trois calamites et des trois cartes bonus.

13. TESTS MINIMAUX :
   - `internal/models` : kinds valides, regions couvrantes et connexes,
     localisation unique d'une carte, main bonus seulement, augures par slot,
     owner `NEUTRAL` autorise uniquement pour une armee et absence de score.
   - `internal/db/assetgen` : cles obligatoires, cles inconnues, slots des trois
     saisons, pourcentage entier, poids positifs, peste rare, bornes de revolte
     et bornes de bonus.
   - `internal/engine/mapgen` : N+1 seeds villages, BFS deterministe, egalites
     departagees par trigramme, partition connexe et reproductible.
   - `internal/engine` :
     - meme seed et meme etat donnent le meme deck et la meme pioche ;
     - remelange reproductible et defausse complete ;
     - calamite programmee hors main, slot suivant, defausse si slots pleins ;
     - repioche apres calamite dans un `T C` ;
     - main pleine no-op et deux `T C` maximum par joueur ;
     - `D C <KIND>` choisit une carte du kind dans l'ordre de main ;
     - `P` consomme une carte et invalide atomiquement sans carte ;
     - augure cachee avant printemps et revelee au printemps ;
     - calamite spring/summer/winter appliquee dans le bon tour, y compris
       l'evenement winter ;
     - peste avec `ceil(N/2)`, un tirage noble par noble et resultat
       deterministe ;
     - mauvais temps avec H et soutien defensif autorises, mouvements,
       dispersion, soutien offensif et pillage refuses ;
      - revolte jouable seulement sous condition de famine, avec nouveaux IDs,
        bornes de taille, candidats sans armee et exclusion du scoreboard ;
     - famine sur production des moulins et bonus de rations ;
     - BT uniquement contre mauvais temps, RA uniquement contre famine ;
     - doublons BT/RA non cumulatifs et BT + RA cumulables par region ;
     - bonus et calamite resolus simultanement, sans dependance a l'ordre des
       joueurs ;
     - bonus dans une autre region independant de la calamite ;
     - purete des resolvers et absence de mutation de l'etat d'entree.
   - `internal/engine/orders` : parser FR/EN, alias des kinds, arites,
     commentaire, limite `T C`, seed de region, conflit avec `P N NNN`.
   - `internal/api` et `internal/store` : contrat `special`, soumission
     atomique, hotseat, multi-parties, projection de main privee, augure
     publique seulement apres revelation et absence de fuite de defausse.
   - `web` : types et normalisation, affichage de la main, insertion des
     commandes, toggles de regions, contour pointille, hachure 45 degres,
     tooltip et section augures, modes FR/EN, mobile et accessibilite.

14. ORDRE DE REALISATION :
   - Commencer par le modele de carte/deck/augure et les invariants.
   - Ajouter la balance et la generation deterministe des regions et du deck.
   - Ajouter le parseur des ordres speciaux et les alias FR/EN.
   - Integrer la pioche, la defausse et les slots dans `ResolveWinter`.
    - Integrer la soumission `special` et les ordres `P` avant supply et
      intentions dans le resolver de chaque saison, sans noble requis.
    - Integrer les trois calamites dans supply, intentions, combats et nobles ;
      traiter la revolte comme carte bonus conditionnelle et creer ses armees
      neutres.
   - Ajouter les events, rapports, projections et adaptations du store.
   - Ajouter l'UI de main, les brouillons, la couche regions, les hachures et
     les augures.
   - Mettre a jour les regles et les specs.
   - Executer les tests Go et frontend, puis les tests de contrat et de
     determinisme avant toute revue.

15. CRITERES D'ACCEPTATION :
   - Une partie creee possede une partition N+1 regions reproductible, dont
     chaque region est rattachee a un village seed public dans la carte.
   - La balance controle la limite de main, le nombre de `T C`, la taille et la
     composition du deck, les trois capacites saisonnieres, la rarete de la
     peste et les valeurs numeriques des effets.
   - Un hiver permet de defausser par kind et de tirer au plus deux fois par
     joueur. Une calamite tiree est programmee ou defaussee, jamais ajoutee a la
     main ; le tirage continue apres son traitement.
   - `P BT TER` et `P RA TER` sont visibles dans l'UI, parsables en FR et EN,
     consomment une carte et agissent simultanement sur la region cible.
   - Beau temps n'annule que le mauvais temps, bonne recolte n'annule que la
     famine, et les deux bonus identiques se cumulent seulement lorsqu'ils
     proviennent de kinds distincts effectifs.
   - Les augures de l'annee sont reveles au printemps avec leurs trois slots,
     puis les effets s'appliquent dans le tour exact, y compris l'hiver.
   - Les cartes en main, la pioche, la defausse et les augures futures restent
     privees selon les projections ; les effets publics restent visibles.
   - `NEUTRAL` n'est ni un joueur, ni une ligne du scoreboard, ni une identite
     de soumission, mais ses armees sont coherentes avec les combats et les
     rapports.
   - Les frontieres d'eveche sont un contour pointille visible avec palette
     gris-bleu activable dans `MapLegend`. Le controle territorial est rendu
     par une hachure diagonale 45 degres de la couleur du joueur, sans ancien
     contour colore.
   - Les tests existants ne regressent pas et les commandes suivantes passent :

     ```text
     go test ./...
     go vet ./...
     cd web && npm run test
     cd web && npm run build
     cd web && npm run lint
     ```

Note : dans la reponse finale, documenter les structures ajoutees, la syntaxe
FR/EN, la strategie RNG, la confidentialite des cartes, le rendu des regions,
les tests executes et les points volontairement laisses aux futures cartes
politique, religieuses et de succession.
```
