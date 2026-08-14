# Online

**Milestone lié :** [Online Hosted](https://github.com/fogfactory/crown-and-borough/milestone/10)

**Issue principale :** [#2 — finaliser les fonctionnalités online et la vue
privée](https://github.com/fogfactory/crown-and-borough/issues/2)

**Plan d'implémentation :** [`online-plan.md`](online-plan.md)

**Contrats O1 :** [`fixtures/`](fixtures/) contient les documents JSON complets
utilisés pour figer les contrats de carte, d'état et de rapport. Le trigramme
est l'unique identité territoriale publique (`territories[].id`) ; `code` n'est
pas un champ territorial requis.

Le mode online actuel fournit une session en mémoire, la soumission par joueur,
une résolution synchrone et la projection privée hotseat via
`GET /api/state?player=P1`. Le plan cible un MVP jouable entre amis avec une
seule partie active par déploiement, des invitations privées, une projection
serveur filtrée et une restauration après redémarrage. Les documents de travail
sont regroupés dans
[`prompts/`](prompts/) avec un nom thématique :

- [`identifiants-territoires.md`](prompts/identifiants-territoires.md)
- [`api-production.md`](prompts/api-production.md)
- [`authentification-sessions.md`](prompts/authentification-sessions.md)
- [`persistance.md`](prompts/persistance.md)
- [`front.md`](prompts/front.md)
- [`deploiement.md`](prompts/deploiement.md)

## Périmètre suivi

- identité territoriale et contrats publics ;
- API d'une partie active et soumission par joueur ;
- authentification, invitations et reprise d'un emplacement ;
- persistance et restauration ;
- vues privées côté serveur ;
- parcours front multi-joueur ;
- déploiement et stockage durable.

## Contrats figés du MVP

Les décisions structurantes sont communes à toutes les sous-issues online :

- une seule partie active par déploiement, pour deux à huit joueurs online ;
- les routes conservent `/api/games/{id}` pour l'évolution multi-parties ;
- `chain: null` signifie qu'aucune chaîne n'est active et
  `chain.visibility: "hidden"` qu'une chaîne existe sans détail révélé ;
- les combats portent `visibility: "exact"` ou `"general"`, avec suppression
  des forces et identifiants dans la vue générale ;
- la résolution forcée est explicite et aucune deadline automatique n'est
  ajoutée ;
- les tokens sont Bearer, en mémoire, sans mot de passe ni expiration au MVP ;
- `DATA_DIR` est l'interface unique de stockage, avec un backend filesystem ou
  un backend snapshot pour GCS FUSE ;
- `net/http` et `http.ServeMux` sont utilisés, tandis que les endpoints hotseat
  restent dev-only.

Les fixtures distinguent volontairement les variantes de confidentialité afin
que les sous-issues O2 à O8 puissent les réutiliser sans redéfinir le contrat.

Le découpage en sous-issues, les critères de validation manuelle et les
garanties de stockage sont décrits dans [`online-plan.md`](online-plan.md).

Les extensions de règles décrites dans les autres fichiers ne doivent pas être
ajoutées à l'issue #2 lorsqu'elles ne sont pas nécessaires à la finalisation du
mode online.
