# Résolution de l'hiver

**Milestone lié :** [Religieux](https://github.com/fogfactory/crown-and-borough/milestone/5)

**Dépend de :** [GDD § Phase d'hiver](gdd.md), [religieux.md](religieux.md),
[dames.md](dames.md), [succession.md](succession.md), [titres.md](titres.md),
[economie.md](economie.md).

L'hiver accumule des ordres qui se lisent les uns les autres : sanctions du
pape, enquêtes, mariages, élections, procès. Ce document fixe **l'ordre exact
de résolution** et les **effets croisés**. Il est la référence unique : les
specs thématiques décrivent chaque ordre, celui-ci dit quand il s'exécute et ce
qu'il voit des autres.

> **Annonce.** En hiver, la projection d'état de chaque joueur porte
> `openElections` : les élections du registre, ses propres voix et ses nobles
> éligibles ; l'aperçu d'ordres rappelle la syntaxe `K`/`V` et qu'un titre
> acquis cet hiver n'est utilisable qu'à l'hiver suivant
> ([#375](https://github.com/fogfactory/crown-and-borough/issues/375)). Aucun
> bulletin n'y figure.
>
> **État.** Les étapes 0 (instantané et registre des élections), 1
> (excommunication et levée,
> [#376](https://github.com/fogfactory/crown-and-borough/issues/376)), 2 (dont
> l'achat de cardinal `N C NNN` et la carte de cardinal `D N NNN CAR`), 5, 6
> (élections), 7 (investiture, y compris les cardinaux achetés) et 8 sont en
> place dans le moteur
> ([#374](https://github.com/fogfactory/crown-and-borough/issues/374)). Les
> étapes 3, 4 et 9 (enquête, dissolution, procès à deux cardinaux) sont en
> place. La reprise dans `gdd.md` et les règles
> joueurs relève de
> [#377](https://github.com/fogfactory/crown-and-borough/issues/377).

## Principes

1. **Étapes ordonnées.** L'hiver se résout en étapes fixes, dans l'ordre du
   tableau ci-dessous. Dans une étape, les ordres sont traités par identifiant
   de joueur croissant, puis dans l'ordre de la feuille, sauf mention contraire.
2. **Un ordre lit l'état courant de son étape.** Il voit tout ce que les étapes
   précédentes ont changé (R payés, statut d'un noble, titre perdu, mariage
   conclu). Il ne voit rien des étapes suivantes.
3. **Une perte de titre est immédiate, un gain est différé.** Un titre
   religieux perdu (excommunication, démasquage) cesse de produire voix,
   pouvoirs et candidature dès l'étape où il est perdu. Un titre religieux
   *gagné* pendant l'hiver par élection ou par achat de cardinal n'est conféré
   qu'à l'**investiture**, à la fin des élections : il ne donne ni voix, ni pouvoir,
   ni éligibilité avant l'hiver suivant. Un titre ne franchit donc jamais
   plus d'un échelon par hiver (évêque → cardinal → pape). Les cartes de
   dignité font exception : elles se jouent à tout moment, la carte de cardinal
   (`D N NNN CAR`, étape 2) prend effet aussitôt et ses voix comptent dès les
   élections du même hiver.
4. **Le registre des élections est figé au début de l'hiver.** Une élection est
   ouverte si, à l'instant où l'hiver commence, son siège est vacant et sa
   condition de déclenchement est vraie. Un siège rendu vacant pendant l'hiver
   n'est électable qu'à l'hiver suivant. Le poste de commandement peut ainsi
   annoncer les élections avant la saisie des ordres
   ([#375](https://github.com/fogfactory/crown-and-borough/issues/375)).
5. **Les positions ne bougent pas en hiver.** Contrôle, occupation et
   appartenance régionale sont ceux du début de l'hiver pendant toutes les
   étapes (aucune armée ne bouge). Constituer un fief (`T F`) ne change donc
   jamais les voix territoriales d'un évêché, qui comptent les lieux-dits
   « contrôlés **ou** occupés ».
6. **Les élections se résolvent l'une après l'autre.** Évêchés par identifiant
   de région croissant, puis conclave. Les voix se lisent toutes sur
   l'instantané des titres (principe 3 compris) : un résultat ne change les
   voix d'aucune autre élection. Seule l'éligibilité en tient compte : un noble
   élu à un évêché n'est plus éligible aux évêchés suivants du même hiver.
7. **Les morts d'hiver arrivent en dernier.** Le seul ordre d'hiver qui tue est
   le procès à deux cardinaux, jugé après toute autre résolution (comme la
   carte de procès l'est à la fin d'un tour d'action).

## Ordre de résolution

| # | Étape | Ordres et effets | Spec |
|---|---|---|---|
| 0 | **Instantané** | Fige le contrôle, l'occupation, les statuts, les mariages, les titres, et calcule le registre des élections ouvertes. | ce document |
| 1 | **Sanctions pontificales** | `X E NNN` (excommunier), `X L NNN` (lever), dans l'ordre de la feuille du pape. | [religieux.md](religieux.md#excommunication) |
| 2a | **Ordres royaux** | `W X`, `W R`, `W F`, `W G` du roi ou de la régente (autorisations, octroi gratuit) ; lus par l'étape 2. | [politique.md](politique.md#pouvoirs-royaux) |
| 2 | **Ordres de gestion** | Ordres individuels actuels : `T N`, `R N`, `D N`, `C N`, `D C`, `A N`, `R T`, `C M/C/D`, `E C`, `O N`, `P N`, `H N`, `G`, `T F`, `T A`, `T X`, `T R`, `V C`, `S R` ; plus l'achat de cardinal (`N C NNN`). Joueurs par identifiant, ordres dans l'ordre saisi. | GDD, [succession.md](succession.md), [titres.md](titres.md) |
| 3 | **Actions des cardinaux** | Enquêtes `Q HHH NNN` (dans l'ordre des joueurs, puis de la feuille) ; dépôt des ordres de procès `J HHH NNN` (jugés à l'étape 9). | [religieux.md](religieux.md#enquête) |
| 4 | **Dissolutions de mariage** | `X D NNN` du pape et demande de l'époux, par couple. | [religieux.md](religieux.md#dissolution-de-mariage) |
| 5 | **Mariages** | Ordres `M N` réciproques. | [succession.md](succession.md#conclusion-dun-mariage) |
| 6 | **Élections** | Candidatures `K E` / `K P`, votes `V E` / `V P` : évêchés par identifiant de région croissant, puis conclave, puis trône (`K R` / `V R`). Résolues dans cet ordre ; voix lues sur l'instantané des titres. | [religieux.md](religieux.md) |
| 7 | **Investiture** | Les élus deviennent évêque ou pape ; les achats de cardinal de l'étape 2 prennent effet. | ce document |
| 8 | **Fin d'hiver** | Défausses et remplissage des mains, fiefs vacants attribués par défaut, conservation des stocks, prospérité, rapatriement, territoires sans ancre redevenus neutres, rapports. | GDD § Phase d'hiver |
| 9 | **Jugement des procès** | Procès `J` déposés à l'étape 3, jugés dans l'ordre croissant du code de la cible. Mort normale (lignée, fiefs, Claims). | [dames.md](dames.md#carte-de-procès) |

Deux conséquences directes du tableau :

- une excommunication frappe **avant** qu'un évêque soit promu cardinal : le
  noble visé est déjà sans titre à l'étape 2, l'achat est rejeté sans
  prélèvement ;
- un noble excommunié ou démasqué cet hiver peut être jugé le même hiver
  (étapes 1 ou 3, puis 9).

## Effets croisés

### Titres religieux

| Événement | Effet immédiat | Effet sur les élections du même hiver |
|---|---|---|
| Excommunication (étape 1) | Titre perdu définitivement ; plus de voix ni candidature. | Le siège est vacant mais n'est pas dans le registre : élection à l'hiver suivant. Un excommunié ne peut pas être candidat. Le dénominateur de la majorité absolue papale baisse s'il était cardinal. |
| Levée (étape 1) | Le noble redevient éligible. Aucun titre rendu. | Peut être candidat dès cet hiver. |
| Démasquage par enquête (étape 3) | Éon et Sorcière : excommunication d'office, même effet que ci-dessus. Correspondante et Espionne : aucun effet religieux. | Idem excommunication. |
| Carte de cardinal (étape 2) | Gratuite, jouée sur un évêque du joueur ; effet immédiat. | Le noble est cardinal dès l'étape 2 : ses 2 voix comptent dans les élections de l'hiver, il vote au conclave s'il est ouvert et compte dans la majorité absolue. L'ouverture du conclave reste décidée par le registre figé à l'étape 0. |
| Achat de cardinal (étape 2) | R prélevés à l'étape 2 ; le plafond d'achat compte les achats en attente. | Aucun avant l'investiture : le nouveau cardinal ne vote pas au conclave, n'y est pas candidat (en tant que cardinal) et ne compte pas pour la majorité absolue. |
| Élection d'un évêque (étape 6) | Titre conféré à l'investiture. | Aucun : ses voix d'évêque ne comptent pas dans les autres élections de l'hiver. |
| Élection du pape (étape 6) | Titre conféré à l'investiture. | Aucun : ses 3 voix n'ont pas compté, ses pouvoirs commencent l'hiver suivant. |
| Procès exécuté (étape 9) | Mort ; titres vacants (hors cible excommuniée, déjà sans titre). | Siège électable l'hiver suivant. |

Les cartes jouées en saison d'action (dîme, apaisement)
sont appliquées avant l'instantané de l'hiver suivant : leurs effets sont donc
déjà visibles de toutes les étapes.

### Statuts de capture

Un statut changé à l'étape 2 (`O N`, `P N`, `H N`) pèse sur toutes les étapes
suivantes : un noble devenu `hostage` n'est plus candidat (capturé) ; un noble
mis au `dungeon` voit ses voix et pouvoirs suspendus (étape 3 et 6) ; un pape
au cachot ne peut plus excommunier. Un noble `hostage` garde voix et pouvoirs.
Les voix d'un titre reviennent toujours au **joueur propriétaire** du noble,
jamais à son geôlier.

### Mariage

- L'enquête (étape 3) lit les mariages **avant** dissolution et conclusion :
  le coût « conjoint − 1 » est celui que le joueur voyait en saisissant l'ordre.
- Une dissolution (étape 4) précède les mariages (étape 5) : un noble libéré
  peut être remarié le même hiver. Elle ne retire aucune prétention en cours.
- Les élections (étape 6) lisent les mariages après étapes 4 et 5 : un noble
  marié cet hiver n'est plus « célibataire » ; sa candidature est rejetée
  (`candidate_not_eligible`), de même qu'un noble libéré par dissolution
  devient candidat éligible.
- Le jugement (étape 9) lit les mariages finaux : une dame qui s'est mariée cet
  hiver n'est plus jugeable en procès direct ; une dissolution de l'étape 4
  lève la protection.

### Cartes de dignité et Abbesse

Une dignité jouée à l'étape 2 produit ses effets dès l'étape 3. L'Abbesse
posée à l'étape 2 apporte donc sa voix dès l'élection de l'étape 6. Une
Abbesse excommuniée (étape 1) ne vote pas.

### Argent

Tous les coûts en R sont prélevés dans l'étape de l'ordre (2 pour achats et
investissements, 3 pour l'enquête), sur les réserves de paiement du joueur,
sans prélèvement partiel en cas de rejet. Les ordres coûteux de l'étape 2
sont donc servis avant une enquête. Un ordre rejeté pour une raison de
condition ne prélève jamais.

## Règles propres aux élections

### Candidatures

- Un joueur ne présente qu'**une candidature par élection** (la première
  valide de la feuille). Un même noble peut être candidat à plusieurs
  élections, mais la candidature à une élection est rejetée
  (`candidate_not_eligible`) si le noble a été élu à un évêché plus tôt dans
  l'hiver.
- Éligibilité évaluée à l'étape 6 : homme, célibataire, libre (ni `hostage`
  ni `dungeon`), non excommunié. Élection épiscopale : en outre pas déjà
  évêque. Conclave : **évêque ou cardinal** en titre à l'instantané (un
  titre gagné cet hiver ne compte pas).
- Un siège absent du registre de l'étape 0 rejette toute candidature
  (`election_not_open`).

### Votes

- Un joueur vote une fois par élection (premier vote valide) pour un candidat
  déclaré de cette élection, quel que soit son propriétaire. Le vote pour un
  candidat inexistant est rejeté (`unknown_candidate`).
- Les voix lisent l'instantané des titres (principe 6) et le titre **le plus
  haut** de chaque noble (pape 3, cardinal 2, évêque 1). Un noble excommunié
  (étapes 1 ou 3) ou au cachot ne compte pas.
- Conclave : seuls les cardinaux de l'instantané votent, 1 voix chacun, par
  `V P` porté par un cardinal du joueur. La **majorité absolue** est
  strictement plus de la moitié de **tous** les cardinaux en titre, y compris
  ceux dont la voix est suspendue (cachot). Un cardinal au cachot bloque donc
  la majorité tant qu'il n'est pas libéré.
- Le conclave n'est ouvert que si le trône est vacant et qu'au moins deux
  cardinaux sont en titre à l'instantané.

### Résultat

- Majorité relative (évêque) ou absolue (pape). Égalité au sommet ou aucun
  candidat éligible : siège vacant, réévalué à l'hiver suivant.
- Les voix ne dépendent pas de l'ordre des élections ; seule l'éligibilité des
  candidats en dépend (ci-dessus). Un vote pour un candidat écarté est rejeté
  (`unknown_candidate`).
- Le rapport public donne, pour chaque élection, les candidats et le total de
  voix par candidat. Les bulletins individuels des joueurs restent privés.

## Pouvoirs : ce qu'ils lisent

| Ordre | Étape | Qui peut | Conditions lues |
|---|---|---|---|
| `X E` / `X L` | 1 | Pape (titre actif, non au cachot) | Cible de n'importe quel joueur ; 1 `X E` par hiver ; 1 excommunié à la fois par joueur adverse (une levée précédente de la feuille libère la place). Jamais sur soi. |
| `X D NNN` | 4 | Pape (titre actif, non au cachot) et propriétaire d'un des époux | Mariage actif à l'étape 4 (lu après les étapes 1 à 3) ; l'ordre du pape et la demande d'un propriétaire d'époux ; un seul ordre si le pape possède un époux. |
| `S R` | 2 (déclaration) / 8 (effet) | Propriétaire d'une Sorcière libre (pas au cachot) | Un rituel par hiver ; la calamité ou la saison visée est lue au tirage des calamités de la fin d'hiver ; région de la Sorcière lue à l'étape 2. |
| `N C NNN` (achat de cardinal) | 2 | Joueur propriétaire d'un évêque | Évêque non excommunié ; plafond `N - 1` incluant les achats en attente ; coût `religious.cardinal_cost` (balance). |
| `Q HHH NNN` | 3 | Cardinal ou pape titré à l'instantané, ni excommunié ni au cachot à l'étape 3 | 1 par cardinal ou pape et par hiver ; coût selon le rang de la cible lu après l'étape 2 ; consommé sans effet si aucune dignité cachée. |
| `J HHH NNN` | 3 (dépôt) / 9 (jugement) | Deux cardinaux distincts, titrés à l'instantané, ni excommuniés ni au cachot à l'étape 3 | 1 par cardinal et par hiver (`trial_limit`) ; mêmes cible et motif pour les deux ; mêmes règles d'éligibilité de cible que la carte de procès, relues au jugement. |
| `X D NNN` | 4 | Pape + un époux | Le pape et le propriétaire d'un des époux soumettent chacun `X D NNN` pour le même couple (un seul ordre si le pape est lui-même propriétaire d'un époux). Les deux nobles doivent être mariés à l'étape 4. |
| `M N` | 5 | Les deux propriétaires | Inchangé. |
| `K` / `V` | 6 | Voir ci-dessus | Voir ci-dessus. |

## Morts et leurs suites

Seul le procès à deux cardinaux tue en hiver. Les morts survenues pendant les
saisons d'action sont déjà dans l'instantané. Au jugement :

- fiefs, prétentions et cartes de la cible sont réglés comme pour toute mort
  ([succession.md](succession.md)) ; un fief laissé vacant l'est jusqu'à un
  `T A` ou à l'attribution par défaut de la fin de l'hiver suivant ;
- les titres religieux de la cible sont vacants (en pratique aucun : une cible
  non-dame doit être excommuniée, donc sans titre) ;
- l'exécution d'une dame ouvre l'éligibilité des révoltes de sa région pour le
  printemps ([ordres-speciaux.md](ordres-speciaux.md)) ;
- aucune élection n'est tenue sur un siège libéré ce jour.

## Hors périmètre

- L'ordre actuel des transferts `G` (par identifiant de joueur) n'est pas
  modifié.
- L'élection royale ([politique.md](politique.md#élection)) réutilise le moteur
  générique ([#374](https://github.com/fogfactory/crown-and-borough/issues/374)) ;
  elle suit le conclave à l'étape 6, la couronne étant conférée à l'investiture
  (étape 7).
- La syntaxe `N C` (achat de cardinal) et `X D` (dissolution) est confirmée.
