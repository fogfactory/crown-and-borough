# Online

**Milestone lié :** [Online Hosted](https://github.com/fogfactory/crown-and-borough/milestone/10)

**Issue principale :** [#2 — finaliser les fonctionnalités online et la vue
privée](https://github.com/fogfactory/crown-and-borough/issues/2)

**Plan d'implémentation :** [`online-plan.md`](online-plan.md)

Le mode online actuel fournit une session en mémoire, la soumission par joueur
et une résolution synchrone. Le plan cible un MVP jouable entre amis avec une
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

Le découpage en sous-issues, les critères de validation manuelle et les
garanties de stockage sont décrits dans [`online-plan.md`](online-plan.md).

Les extensions de règles décrites dans les autres fichiers ne doivent pas être
ajoutées à l'issue #2 lorsqu'elles ne sont pas nécessaires à la finalisation du
mode online.
