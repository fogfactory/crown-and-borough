# Ravitaillement et famine

**Milestone lié :** [v1](https://github.com/fogfactory/crown-and-borough/milestone/1)
et [Économie & Fiefs](https://github.com/fogfactory/crown-and-borough/milestone/19)
pour le timing de fin de tour ([#208](https://github.com/fogfactory/crown-and-borough/issues/208)).

Cette page indexe les règles de logistique du socle actuel. La règle complète
reste dans [`gdd.md`](gdd.md), sections 3 et 5 ; l'architecture d'exécution est
décrite dans [`architecture.md`](architecture.md).

## Règles de référence

- Le ravitaillement est résolu en **fin de tour d'action**, après la
  résolution des ordres (intentions, combats, déplacements, retraites,
  jonctions, dispersions, transferts) et la mise à jour du contrôle
  territorial : revenu territorial, production des moulins, rations de
  terrain, puis famine se calculent tous sur les positions et le contrôle
  définitifs du tour, territoires tout juste capturés compris.
- Une armée de `N` troupes demande `2^(N - 1)` rations.
- Seul le terrain produit des rations locales (plaine 2, forêt 1, colline 1,
  montagne 0, marécage 1) ; un château ou un village n'en ajoute pas. La
  famine supprime les rations de terrain de sa région ; la Récolte abondante
  les double.
- La production locale est consommée sur place par l'armée présente jusqu'à
  hauteur de sa demande ; le surplus est perdu.
- Les châteaux et villages contrôlés sont des sources de ravitaillement.
- Une case contrôlée qui contient un stock positif est également une source ;
  l'armée présente consomme d'abord ce stock local.
- Hors fief et hors capitale, un château ou un dépôt sans armée dessus est
  **inerte** ([#215](https://github.com/fogfactory/crown-and-borough/issues/215)) :
  il n'est plus ni source ni dépôt utilisable pour personne, exactement comme
  s'il était occupé contre son contrôleur — sauf qu'il n'a alors même plus de
  contrôleur du tout. Un village garde en revanche sa production neutre même
  abandonné (voir [economie.md](economie.md#revenu-territorial)).
- Le flux traverse les cases alliées, neutres ou contrôlées par un autre joueur
  et ne s'arrête que devant une case occupée par une armée adverse. Un château,
  un village ou un dépôt adverse sans armée ne bloque pas le flux.
- Une case **occupée contre son contrôleur** (titres.md, notamment dans un
  fief) n'est plus elle-même une source ni un dépôt utilisable, ni pour le
  contrôleur ni pour l'occupant, même si elle continue de bloquer ou de
  laisser passer le flux traversant selon la règle ci-dessus.
- La portée de base est de trois cases ; un dépôt contrôlé, non occupé, ajoute
  deux cases.
- En déficit, les stocks sont épuisés puis les armées passent en famine selon
  la distance, la taille et le trigramme territorial.
- Une armée qui termine ainsi le tour en déficit est marquée **affamée**, un
  statut qui persiste pendant tout le tour suivant : elle combat et se défend
  à force zéro, même avec un noble commandant, et ne peut émettre de transfert
  de ressources (elle peut toujours en recevoir). Ce premier tour de déficit
  ne lui coûte ni pillage de l'infrastructure de sa case, ni perte de troupe,
  ce qui laisse au joueur tout le tour suivant pour la déplacer ou lui envoyer
  des ressources.
- Son statut est recalculé à la prochaine résolution de ravitaillement, sur
  les mêmes règles : elle redevient valide dès qu'elle atteint une source
  suffisante. Si elle est encore en déficit à ce moment alors qu'elle est déjà
  affamée, elle pille automatiquement une infrastructure située sur sa case ;
  si le pillage est insuffisant ou impossible, elle perd une troupe, jusqu'à
  un minimum de 1 troupe, et reste affamée pour le tour d'après.

La projection de ravitaillement affichée au joueur (`/supply`) applique les
calamités de la saison en cours et les cartes bonus de son brouillon sur les
positions et le contrôle actuels, comme si les armées ne bougeaient pas d'ici
la résolution ; les cartes des autres joueurs, inconnues, n'y figurent pas.

Les stocks peuvent exister sur toute case pendant les tours d'action, notamment
après un transfert. En hiver, ceux d'un château ou d'un village sont conservés
à `ceil(stock / 2)`, ceux d'un dépôt de vivres intégralement, et les autres sont
perdus. Les stocks hors château et village ne paient pas les investissements
d'hiver.

## Périmètre

Les changements de règles qui touchent les stocks, les rations, la famine ou la
production doivent être rattachés au milestone v1 ou au thème
[`economie.md`](economie.md), sans réintroduire une notion de délai de
transmission.
