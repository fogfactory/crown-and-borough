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

1. **Réception des rapports :** Le joueur prend connaissance de la carte selon l'ancienneté des messagers parvenus à sa Capitale ou à ses Nobles.
2. **Ordres & Diplomatie :** Négociations privées, programmation des chaînes d'ordres et des axes logistiques.
3. **Résolution serveur simultanée :**
   - *Traçage des flux :* Établissement automatique des lignes de ravitaillement.
   - *Propagation :* Avancée des messagers (ordres et rapports).
   - *Exécution :* Application des ordres reçus ce tour-ci par les armées.
   - *Combats & Retraites :* Résolution des affrontements et replis obligatoires.
   - *Ravitaillement & Carence :* Consommation des ressources R, prélèvement sur les stocks en cas de déficit, basculement en famine si pénurie.
   - *Émission :* Départ des nouveaux rapports d'information du terrain vers la Capitale.

### Phase d'Hiver (Bilan, Conservations et Investissements)

- **Trêve hivernale :** Mouvements militaires gelés (aucune chaîne traitée, aucun combat, aucun ravitaillement).
- **Ordres d'Hiver :** le joueur soumet une liste d'ordres (même mécanique que les tours d'action), une ligne = un investissement, traités dans l'ordre saisi :
  - `R N <territoire>` — recruter un **Noble** sur le territoire (nom = prénom tiré + "de \<nom du territoire\>", ex. "Jacques de Notombes")
  - `R A <territoire>` — recruter une **Armée** sur le territoire
  - `C M <territoire>` — construire un **Moulin** (améliore le moulin existant)
  - `C C <territoire>` — construire un **Château**
  - `C R <territoire>` — construire un **Relais de Poste**
  - `C T <territoire>` — construire une **Tour de Guet**
  - `C D <territoire>` — construire un **Dépôt de Vivres**
  - `E C <territoire>` — désigner le château de ce territoire comme **Capitale**
- **Coûts des investissements :**

  | Investissement | Coût (R) |
  |---|---|
  | Château | 10 |
  | Moulin | 3 |
  | Armée | 1 |
  | Noble | 2 |
  | Relais de Poste | 2 |
  | Tour de Guet | 4 |
  | Dépôt de Vivres | 3 |

  *(Toutes les métriques d'équilibrage sont stockées dans `assets/balance.json`, facilement éditables.)*
- **Paiement de proche en proche :** le coût d'un investissement est prélevé d'abord sur le stock du territoire ciblé (s'il stocke), puis sur le lieu-dit contrôlé le plus proche, et ainsi de suite. Réserves totales insuffisantes → ordre rejeté (aucun prélèvement partiel).
- **Capitale :** la capitale d'un joueur n'est PAS un territoire fixe : c'est un **château qu'il considère comme sa capitale**. Par défaut, son **premier château**. Un ordre d'Hiver `E C <territoire>` désigne un autre de ses châteaux comme capitale. La capitale ne peut jamais être ennemie : la désignation exige le contrôle de la case. Si le château désigné est détruit, la désignation est perdue (le prochain château construit redevient capitale par défaut).
- **Conservation des stocks (Règle des 50 %) :** sur chaque lieu-dit, les ressources non consommées durant l'année subissent une perte hivernale après investissement :

  Stock Conservé au Printemps = ceil(R_restant / 2)

  *(Arrondi à l'entier supérieur sur chaque lieu-dit).*

- **Rapatriement des stocks :** les stocks sont automatiquement rapatriés vers la case de la capitale du joueur, en laissant au maximum **1 R par lieu-dit** et **2 R par château** (hors capitale : elle accumule tout le surplus). Si le joueur ne contrôle plus aucun château, ses stocks **ne sont pas rapatriés** (ils restent sur place).
- **Ordre de la phase :** investissements → conservation 50 % → rapatriement.
- **Départ de la partie :** chaque joueur commence sur un **lieu-dit distinct**, où un **château est construit automatiquement** (sa capitale par défaut — son premier château), avec **au moins une armée** et un **stock de ressources initial** (valeurs d'équilibrage dans `assets/balance.json` : armées et ressources de départ).

---

## 3. Carte, Terrains et Lieux-dits

### Types de Terrains (Vitesse des Messagers)

La vitesse de déplacement des armées et la propagation des messagers dépendent du relief :

| Terrain | Déplacement Armée | Vitesse du Messager |
| --- | --- | --- |
| **Plaine / Route** | 1 case / tour | **2 cases / tour** |
| **Forêt / Colline** | 1 case / tour | **1 case / tour** |
| **Montagne / Marécage** | 1 case / 2 tours | **0,5 case / tour** (1 case tous les 2 tours) |

### Lieux-dits et Maillage Territorial

- **Zones Sauvages (~75 % de la carte) :** Produisent **0 R**. Servent de zones de transit, de combat ou d'infrastructures isolées.
- **Lieux-dits (~25 % de la carte) :** Seules cases générant des ressources R de base et permettant le stockage local.
- **Règle de la Structure Unique :** Une seule infrastructure par case.
- **Château comme Lieu-dit Synthétique :** Un château construit sur une zone sauvage transforme la case en Lieu-dit.

---

## 4. Latence d'Information et Transmission des Ordres

- **Vision Temps Réel (T0) :** Accordée uniquement sur les cases contenant la **Capitale**, un **Noble**, ou une **Tour de Guet**.
- **Information Différée (T-x) :** Le joueur voit l'état le plus récent apporté par ses messagers sur le reste de la carte.
- **Ordres par Messager :** Les ordres émis depuis la Capitale voyagent à la vitesse du terrain jusqu'à l'armée ciblée. Une armée poursuit son ancienne chaîne tant qu'aucun nouvel ordre ne l'a atteinte.

---

## 5. Armées, Combats et Logistics

### Combats et Force

- **Force de base :** 1 armée = 1 force.
- **Pas de fusion des attaques :** deux attaques alliées sur la même case ne se combinent PAS : chaque pile attaquante est un contendant distinct (la fusion n'existe que via l'ordre de Jonction J, déplacement pacifique).
- **Puissance d'attaque :** taille de la pile attaquante (les armées d'une même pile ciblant la même case s'additionnent). Des piles d'origines différentes ne se combinent jamais : pour cumuler des forces sur une case, il faut un **soutien S**.
- **Puissance de soutien :** taille de la pile soutenante (coupée si la pile est attaquée ce tour).
- **Résolution (manière Diplomacy) :** l'ensemble des intentions (attaques, soutiens, déplacements pacifiques) est calculé et itéré jusqu'à stabilité AVANT d'exécuter les mouvements. Force égale = Statu quo. Supériorité numérique = Victoire.
- **Multi-contendants :** quand plusieurs joueurs attaquent la même case, toutes les forces s'affrontent dans une comparaison unique (chaque pile attaquante + la défense). La force **strictement la plus haute** l'emporte et occupe la case ; **toute égalité au sommet = statu quo** : la défense tient et tous les attaquants échouent, même si leur force est supérieure à la défense — y compris deux attaques alliées à égalité entre elles (ex. 1 + 1 contre 1 : statu quo ; pour prendre la case, faire une attaque soutenue).
- **Déplacements pacifiques (Jonction, Dispersion) :** puissance 0 ; repoussés si leur destination est contestée par une attaque ce tour.

### Ravitaillement Exponentiel

Une pile de N armées sur une même case consomme :

Coût en R = 2^(N-1)

*(1 armée = 1 R | 2 armées = 2 R | 3 armées = 4 R | 4 armées = 8 R)*

### Portée des Flux

- Les lignes de flux sont tracées automatiquement depuis les lieux-dits à travers des territoires alliés ou neutres.
- **Portée de base :** 3 cases (étendue à 5 cases via un Dépôt de Vivres).

---

## 6. Ordres, Chaînes de Commandement & Retraites

### Types d'Ordres

- **Mouvement / Attaque / Maintien / Soutien** (règles type *Diplomacy*). Le mouvement et l'attaque partagent la même mécanique de déplacement (le combat n'intervient que si la case est occupée par l'ennemi). Le **soutien** renforce le contendant **allié** de la case ciblée : l'attaque d'un allié qui s'y déplace, sinon la défense de l'allié qui l'occupe ; sa puissance = la taille de la pile soutenante ; il est **coupé** si la pile soutenante est elle-même attaquée ce tour ; en boucle, il s'interrompt quand plus aucune attaque ne cible la case.
- **Jonction :** Déplacement **pacifique** (pas une attaque) vers une case adjacente. Si une armée alliée s'y trouve (déjà présente, ou arrivée au même tour par une attaque ou une autre jonction), les armées **fusionnent**. Une case occupée par l'ennemi rend la jonction impossible.
- **Séparation / Dispersion :** Diviser une pile d'armées. Chaque armée se voit assigner une destination (adjacente ou la case d'origine, autorisée) libre et non ciblée par une attaque ce tour, vers laquelle elle se déplace pacifiquement. La chaîne d'ordres reste sur l'armée d'origine, qui prend la première destination listée.
- **Pillage :** Détruire une infrastructure de **la case où se trouve l'armée** pour gagner un bonus immédiat en R.

### Chaînes d'Ordres et Liaisons

Un joueur peut programmer des séquences d'ordres successives (O1 → O2 → O3). **Chaque transition (chaque ordre) possède sa propre liaison** : unique ou boucle. **Il n'existe pas de modification de chaîne** : une armée qui reçoit une chaîne remplace la précédente.

- **Liaison Unique :** En cas d'échec de Ox, la chaîne brise. L'armée passe *Sans Ordre*.
- **Liaison Boucle :** En cas d'échec de Ox, l'armée retente Ox au tour suivant jusqu'à réussite.
- **Ordre Invalide :** Tout ordre rendu physiquement ou mécaniquement impossible **brise immédiatement** la chaîne d'ordres, quel que soit le mode de liaison.
- **Position manquante :** chaque ordre précise explicitement la position de l'armée ; si l'armée n'y est pas quand l'ordre s'exécute → échec (single : brise ; boucle : retente).
- **Maintien en boucle** : garde indéfinie (la chaîne reste en veille jusqu'à réception d'un nouvel ordre).
- **Dispersion en boucle** : retente jusqu'à résolution intégrale ; en single, la chaîne avance même si la dispersion est partielle.
- **Progression simultanée :** la progression des chaînes prend en compte toutes les chaînes du tour pour résoudre combats, retraites et jonctions.

### Armées "Sans Ordre" (IA Défensive Auto-équilibrée)

Une armée sans ordre apporte son soutien à l'armée alliée la plus proche qui possède le moins de soutien, stabilisant automatiquement la ligne de front.

### Capacité des Nobles

Un noble ne peut émettre ou modifier qu'**une seule chaîne d'ordres par tour** (T0).

### Phase de Retraite

Une armée défaite doit battre en retraite sur une case adjacente valide (non occupée, sans combat ce tour-ci, et différente de la case d'origine de l'attaquant).

- Si deux armées se replient sur la même case sans autre option : **les deux sont détruites**.
- Si aucune case n'est valide : **l'armée est détruite**.

---

## 7. Infrastructures et Maillage Productif

| Infrastructure | Nécessite un Lieu-dit ? | Effet principal | Coût (R) |
| --- | --- | --- | --- |
| **Moulin / Domaine** | **Oui** (Sur ou relié par adjacence) | Augmente la production de R (+1 R / niveau). | 3 |
| **Relais de Poste** | Non | Doubler la vitesse des messagers sur la case. | 2 |
| **Tour de Guet** | Non | Donne la vision T0 permanente sur la case et adjacentes. | 4 |
| **Dépôt de Vivres** | Non | Étend la portée de la ligne de ravitaillement (+2 cases). | 3 |
| **Château** | Non (Rend la case Lieu-dit) | Apporte **+1 de force défensive** fixe (sans coût de R). | 10 |

### Dépendance et Lieux-dits Orphelins

Une infrastructure productive (Moulin) hors lieu-dit doit être reliée de proche en proche à un lieu-dit source. Si une infrastructure intermédiaire est détruite (Pillage), les infrastructures en aval deviennent **orphelines** et coupent leur production.

---

## 8. Contrôle Territorial et Carence Alimentaire

### Contrôle des Lieux-dits

- Un lieu-dit bascule sous le contrôle d'un joueur dès qu'une de ses armées s'y arrête.
- **Rémanence :** Le contrôle est conservé même après le départ de l'armée, jusqu'au passage d'un ennemi.

### Algorithme de Carence Alimentaire (Famine)

Si la production instantanée ne suffit pas à alimenter toutes les armées :

```
[Déficit de Ravitaillement]
       │
       ▼
[1. Épuisement des Stocks Locaux]
   ► Ordre : Du plus PETIT stock au plus GRAND.
   ► Départage : Ordre alphabétique du trigramme du territoire.
       │
       ▼ (Si stocks totalement épuisés)
[2. Armées en Famine (Force = 0)]
   ► Ordre : Armées les PLUS ÉLOIGNÉES de leur source.
   ► Départage : Les piles les PLUS GROSSES d'abord
     (les plus coûteuses en nourriture).
   ► Départage final : Numéro de matricule d'armée DÉCROISSANT.
```

**Effets de la famine :** une armée famélique ne peut que se déplacer à force 0 (elle ne se bat ni en attaque, ni en défense, ni en soutien) ; si elle se trouve sur une case avec une infrastructure, elle la **pille automatiquement** (détruit l'infrastructure la plus récente). Le pillage lui rapporte le bonus R du pillage **moins sa consommation** (2^(N-1)) : si le gain est positif ou nul, la pile se nourrit et n'est plus famélique (l'excédent est conservé sur place) ; sinon elle reste famélique sans rien gagner.
