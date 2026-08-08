# Information et brouillard de guerre

**Milestone lié :** [Brouillard de guerre](https://github.com/fogfactory/crown-and-borough/milestone/9)

**Dépend de :** la vue privée côté serveur de [l'issue #2](https://github.com/fogfactory/crown-and-borough/issues/2).

## Visibilité

Les informations détaillées du gamestate — infrastructures, ressources et
tailles d'armées — ne sont visibles par un joueur que lorsqu'il dispose d'une
armée suffisamment proche. La position et le statut des nobles devront être
intégrés à la même politique de divulgation.

Le filtrage doit être effectué côté serveur, par joueur, et non uniquement dans
le front. Les ordres connus et les combats suivent déjà une politique distincte
décrite dans [`online.md`](online.md).

## Tours de guet

Les tours de guet sont réintroduites comme mécanisme d'information. Leur portée,
leur coût, leur contrôle, leur visibilité et leur interaction avec les armées
doivent être définis avant l'implémentation.
