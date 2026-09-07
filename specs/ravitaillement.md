# Ravitaillement et famine

**Milestone lié :** [v1](https://github.com/fogfactory/crown-and-borough/milestone/1)

Cette page indexe les règles de logistique du socle actuel. La règle complète
reste dans [`gdd.md`](gdd.md), sections 3 et 5 ; l'architecture d'exécution est
décrite dans [`architecture.md`](architecture.md).

## Règles de référence

- Une armée de `N` troupes demande `2^(N - 1)` rations.
- La production locale est consommée sur place, au plus une ration par armée
  et par tour.
- Les châteaux et villages contrôlés sont des sources de ravitaillement.
- Le flux traverse les cases alliées, neutres ou contrôlées par un autre joueur
  et ne s'arrête que devant une case occupée par une armée adverse. Un château,
  un village ou un dépôt adverse sans armée ne bloque pas le flux.
- La portée de base est de trois cases ; un dépôt contrôlé ajoute deux cases.
- En déficit, les stocks sont épuisés puis les armées passent en famine selon
  la distance, la taille et le trigramme territorial.
- Une armée affamée agit à force zéro, même avec un noble commandant, et peut
  piller automatiquement une infrastructure située sur sa case.
- Si le pillage est insuffisant ou impossible, l'armée perd une troupe, jusqu'à
  un minimum de 1 troupe ; elle reste affamée pour ce tour.

## Périmètre

Les changements de règles qui touchent les stocks, les rations, la famine ou la
production doivent être rattachés au milestone v1 ou au thème
[`economie.md`](economie.md), sans réintroduire une notion de délai de
transmission.
