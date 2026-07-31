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

- **Trêve hivernale :** Mouvements militaires gelés.
- **Conservation des stocks (Règle des 50 %) :** Sur chaque lieu-dit, les ressources non consommées durant l'année subissent une perte hivernale après investissement :

  Stock Conservé au Printemps = ceil(R_restant / 2)

  *(Arrondi à l'entier supérieur sur chaque lieu-dit).*

- **Investissements :** Dépense des réserves pour recruter des **Nobles** ou construire des **Infrastructures**.

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
- **Jonction :** Deux armées ciblant la même case au même tour fusionnent (N armées).
- **Résolution :** Force égale = Statu quo. Supériorité numérique = Victoire.

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

- **Mouvement / Attaque / Maintien / Soutien** (règles type *Diplomacy*). Le mouvement et l'attaque partagent la même mécanique de déplacement (le combat n'intervient que si la case est occupée par l'ennemi).
- **Jonction :** Regrouper des armées. L'armée reste sur sa case et la verrouille, bloquant l'entrée à l'ennemi sans se déplacer.
- **Séparation / Dispersion :** Diviser une pile d'armées. Chaque armée se voit assigner un territoire (libre et non ciblé par une attaque ce tour) vers lequel elle se déplace pacifiquement ; le territoire d'origine est autorisé.
- **Pillage :** Détruire l'infrastructure d'une case pour gagner un bonus immédiat en R.

### Chaînes d'Ordres et Liaisons

Un joueur peut programmer des séquences d'ordres successives (O1 → O2 → O3). **Chaque transition (chaque ordre) possède sa propre liaison** : unique ou boucle.

- **Liaison Unique :** En cas d'échec de Ox, la chaîne brise. L'armée passe *Sans Ordre*.
- **Liaison Boucle :** En cas d'échec de Ox, l'armée retente Ox au tour suivant jusqu'à réussite.
- **Ordre Invalide :** Tout ordre rendu physiquement ou mécaniquement impossible **brise immédiatement** la chaîne d'ordres, quel que soit le mode de liaison.
- **Position manquante :** si l'armée n'est pas sur le territoire d'origine de l'ordre quand celui-ci s'exécute → échec (single : brise ; boucle : retente).
- **Maintien en boucle** : garde indéfinie (la chaîne reste en veille jusqu'à réception d'un nouvel ordre).
- **Dispersion en boucle** : retente jusqu'à résolution intégrale ; en single, la chaîne avance même si la dispersion est partielle.

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

| Infrastructure | Nécessite un Lieu-dit ? | Effet principal |
| --- | --- | --- |
| **Moulin / Domaine** | **Oui** (Sur ou relié par adjacence) | Augmente la production de R (+1 R / niveau). |
| **Relais de Poste** | Non | Doubler la vitesse des messagers sur la case. |
| **Tour de Guet** | Non | Donne la vision T0 permanente sur la case et adjacentes. |
| **Dépôt de Vivres** | Non | Étend la portée de la ligne de ravitaillement (+2 cases). |
| **Château** | Non (Rend la case Lieu-dit) | Apporte **+1 de force défensive** fixe (sans coût de R). |

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
   ► Départage : Numéro de matricule d'armée DÉCROISSANT.
```
