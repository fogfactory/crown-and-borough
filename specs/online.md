# Online

**Milestone lié :** [Online Hosted](https://github.com/fogfactory/crown-and-borough/milestone/10)

**Issue principale :** [#2 — fonctionnalités online, Firebase et Firestore](https://github.com/fogfactory/crown-and-borough/issues/2)

**Plan d'implémentation :** [`online-plan.md`](online-plan.md)

**Contrats O1 :** [`fixtures/`](fixtures/) contient les documents JSON complets
utilisés pour figer les contrats de carte, d'état et de rapport. Le trigramme
est l'unique identité territoriale publique (`territories[].id`) ; `code` n'est
pas un champ territorial requis.

Le mode online actuel fournit une session en mémoire, la soumission par joueur,
une résolution synchrone et la projection privée hotseat via
`GET /api/state?player=P1`. Le plan cible un MVP jouable entre amis avec
plusieurs parties simultanées, des invitations privées, une projection serveur
filtrée et une restauration après redémarrage. Les documents de travail sont
regroupés dans [`prompts/`](prompts/) avec un nom thématique :

- [`identifiants-territoires.md`](prompts/identifiants-territoires.md)
- [`api-production.md`](prompts/api-production.md)
- [`authentification-sessions.md`](prompts/authentification-sessions.md)
- [`persistance.md`](prompts/persistance.md)
- [`front.md`](prompts/front.md)
- [`deploiement.md`](prompts/deploiement.md)

## Périmètre suivi

- identité territoriale et contrats publics ;
- API de plusieurs parties actives et soumission par joueur ;
- authentification Firebase, invitations et reprise de session/membership ;
- persistance et restauration ;
- vues privées côté serveur ;
- parcours front multi-joueur ;
- déploiement et stockage durable.

## Contrats figés du MVP hébergé

Les décisions structurantes sont communes à toutes les sous-issues online :

- plusieurs parties actives par déploiement, chacune pour deux à huit joueurs
  online ;
- les routes conservent `/api/games/{id}` pour l'évolution multi-parties ;
- `chain: null` signifie qu'aucune chaîne n'est active et
  `chain.visibility: "hidden"` qu'une chaîne existe sans détail révélé ;
- les combats portent `visibility: "exact"` ou `"general"`, avec suppression
  des forces et identifiants dans la vue générale ;
- la résolution forcée est explicite et aucune deadline automatique n'est
  ajoutée ;
- l'identité est fournie par Firebase Authentication avec un lien de connexion
  par email ; le serveur valide le JWT Firebase porté en Bearer ;
- les profils, appartenances et parties sont persistés dans Firestore ; aucun
  token Firebase ni secret de session n'est écrit dans Firestore ;
- la session persistante est fournie par Firebase côté navigateur et par les
  profils/memberships Firestore côté serveur ; aucune collection
  `sessions/{token}` n'est nécessaire ni autorisée ;
- les codes d'invitation sont opaques, de six caractères, stockés sous forme
  hachée et vérifiés uniquement par le backend ;
- le front utilise le SDK Firebase Web pour conserver la session côté client et
  `onSnapshot` uniquement sur les projections Firestore lisibles par le joueur ;
  l'état canonique, les ordres bruts et les métadonnées privées restent
  inaccessibles au client ;
- les transitions de partie et la résolution utilisent des transactions et des
  préconditions Firestore ; la cohérence ne dépend pas de `max-instances` ;
- `net/http` et `http.ServeMux` sont utilisés, tandis que les endpoints hotseat
  restent dev-only.

## Persistance Firestore

Firestore Native mode est la frontière de persistance du MVP hébergé. Le
serveur Go utilise le SDK Admin/Cloud Firestore avec les credentials ADC du
service account Cloud Run. Le moteur reste pur et ne connaît ni Firestore ni
Firebase Authentication.

Les documents lisibles par le navigateur sont des projections dédiées :

- `games/{gameId}` contient uniquement les métadonnées publiques, les slots et
  les statuts de soumission, jamais l'état canonique ni le code d'invitation ;
- `games/{gameId}/views/{uid}` contient l'état projeté pour le joueur dont le
  `uid` est dans le chemin ;
- `games/{gameId}/reports/{uid}/turns/{turn}` contient les rapports filtrés de
  ce joueur ;
- `players/{uid}` contient le profil du joueur et les métadonnées nécessaires à
  la liste de ses parties.

Les documents réservés au backend comprennent l'état moteur, les soumissions
brutes, les rapports non filtrés, les métadonnées de confidentialité et les
informations hachées d'invitation. Les règles Firestore refusent toute lecture
ou écriture cliente de ces documents. Les écritures clientes passent par l'API
Go, qui vérifie le JWT et l'appartenance à la partie.

Une soumission est enregistrée avec le numéro de tour attendu dans une
transaction. La résolution est revendiquée une seule fois avec une précondition
sur la révision de la partie ; le moteur déterministe calcule ensuite le nouvel
état et un commit conditionnel publie l'état canonique, le rapport et toutes
les projections. Une tentative concurrente est rejetée ou rejouée après
relecture, sans double résolution. Une revendication abandonnée possède un
lease interne récupérable ; ce lease n'est pas une deadline de jeu.

Le free tier Firestore et Cloud Run est un objectif de coût, pas une garantie :
les lectures des listeners, les écritures de projections, le stockage, les
logs, la région et l'activation d'un compte de facturation peuvent générer des
coûts. Des alertes de budget et une documentation des quotas sont obligatoires
avant le déploiement public.

Les fixtures distinguent volontairement les variantes de confidentialité afin
que les sous-issues O2 à O8 puissent les réutiliser sans redéfinir le contrat.

Le découpage en sous-issues, les critères de validation manuelle et les
garanties de stockage sont décrits dans [`online-plan.md`](online-plan.md).

Les extensions de règles décrites dans les autres fichiers ne doivent pas être
ajoutées à l'issue #2 lorsqu'elles ne sont pas nécessaires à la finalisation du
mode online.
