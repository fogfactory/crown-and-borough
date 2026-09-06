# Plan d’implémentation TDD — deck, régions, calamités et ordres exécutables

Ce plan accompagne [`deck.md`](deck.md). Chaque étape correspond à un commit autonome. Aucun commit ne doit être créé sans revue et accord explicite. Avant chaque demande de validation, les tests indiqués doivent être exécutés et l’étape doit être mise à jour dans ce fichier.

## Statuts

- `[ ]` à faire
- `[~]` en cours
- `[x]` terminé et validé par revue

## Règles de travail

- Ne jamais mélanger deux étapes dans un commit.
- Chaque commit doit compiler et conserver les tests précédents au vert.
- Les tests sont écrits avant l’implémentation correspondante.
- Les DTO parsés et persistés restent séparés des interfaces d’exécution moteur.
- Les ordres exécutables consomment les cartes dans leur `Apply` après vérification des préconditions.
- Les effets simultanés sont enregistrés comme intentions avant agrégation.
- Les mises à jour documentaires sont faites dans l’étape qui introduit le contrat concerné.

## Étapes et commits

### `[x] 1. Tests de caractérisation des ordres d’hiver`

**Commit :** `test(engine): characterize winter order execution` (couvert par les tests existants)

**Contenu réalisé :**

- Compléter les tests de `ResolveWinter` sans modifier le comportement.
- Couvrir l’ordre des lignes au sein d’un joueur.
- Couvrir l’ordre déterministe des joueurs.
- Couvrir les rejets sans paiement partiel.
- Couvrir les mutations successives visibles par les ordres suivants.
- Couvrir la pureté de l’état d’entrée et l’absence d’avancement du calendrier.

**Vérifications :** `go test ./internal/engine/...`

### `[x] 2. Interface d’exécution des ordres d’hiver`

**Commit :** `refactor(engine): execute winter orders through an interface`

**Contenu réalisé :**

- Introduire une interface moteur `ExecutableOrder` avec `Apply`.
- Conserver `models.WinterOrder` comme DTO sérialisable.
- Ajouter une factory DTO → ordre exécutable.
- Placer chaque implémentation d’ordre dans son propre fichier moteur.
- Limiter `resolutionContext` aux données, index et primitives génériques.
- Supprimer les méthodes métier `resolveXXX` de `resolutionContext`.
- Faire gérer à chaque `Apply` les préconditions et le rejet sans mutation partielle.
- Documenter la séparation DTO/implémentation dans `specs/architecture.md`.

**Vérifications :** tests moteur complets et `go vet ./...`

### `[x] 3. Tests unitaires des ordres d’hiver`

**Commit :** `test(engine): add winter order unit tests`

**Contenu réalisé :**

- Ajouter un fichier de test dédié à chaque implémentation `ExecutableOrder`.
- Tester directement `Apply` dans le package `engine`.
- Ajouter une suite table-driven regroupant plusieurs causes de rejet par ordre.
- Vérifier la raison de rejet et l’absence de mutation de l’état dans chaque cas.
- Couvrir préconditions, mutations, rejets, événements et paiements.
- Conserver les tests d’intégration existants de `ResolveWinter`.

**Vérifications :** `go test ./internal/engine/...`

### `[ ] 3bis. Ordre d’hiver d’annoblissement`

**Commit prévu :** `feat(engine): add ennoblement winter order`

**Contenu attendu :**

- Ajouter un ordre d’hiver dédié, proposé en syntaxe `A N XXX`.
- Autoriser le recrutement gratuit d’un noble sur n’importe quelle armée ciblée.
- N’autoriser l’ordre que si le joueur ne possède aucun noble.
- Ne pas exiger le contrôle du territoire ni une infrastructure de peuplement.
- Allouer un noble avec les mêmes règles d’identité et de nommage que le recrutement normal.
- Ajouter des tests unitaires table-driven pour absence de noble, armée inexistante, propriétaire incorrect et succès gratuit.
- Documenter la règle en français et en anglais.

**Vérifications :** `go test ./internal/engine/...`, `go vet ./...`

### `[x] 4. Contrats métier deck, cartes, régions et NEUTRAL`

**Commit :** `feat(models): add special deck and region state` (tests écrits en premier dans l’arbre de travail)

**Contenu réalisé :**

- Tester puis ajouter `CardKind`, `SpecialCardID`, `RegionID`, `Region`, `SpecialCard`, `SpecialDeck`, `Calamity` et `YearAugury`.
- Étendre `GameState.Validate` : références, unicité et localisation exclusive des cartes.
- Garantir que les mains ne contiennent que des bonus.
- Laisser la connexité, la couverture et la cohérence géographique des régions à `mapgen`.
- Autoriser `NeutralPlayerID` uniquement pour les armées.
- Exclure `NEUTRAL` des joueurs, chaînes, soumissions et scores.
- Préserver la lecture des anciens états sans cartes.

**Vérifications :** `go test ./internal/models/...`

### `[x] 5. Balance spéciale et génération canonique du deck`

**Commit :** `feat(assetgen): load and generate special deck balance` (tests écrits en premier dans l’arbre de travail)

**Contenu réalisé :**

- Ajouter `special_orders` à `Balance` et `rawBalance`.
- Tester les clés obligatoires/inconnues, slots, pourcentage divisible, poids et bornes.
- Implémenter les plus grands restes avec tie-break canonique.
- Construire les cartes avec IDs stables avant mélange.
- Ne pas générer les cartes placeholder.
- Mettre à jour `assets/balance.yaml` et la documentation de balance.

**Vérifications :** `go test ./...`, `go vet ./...`

### `[x] 6. Partition régionale déterministe`

**Commit :** `feat(mapgen): generate public static regions` (tests écrits en premier dans l’arbre de travail)

**Contenu réalisé :**

- Tester N+1 villages, BFS multi-source, couverture, disjonction et connexité.
- Tester le tie-break par trigramme et la reproductibilité.
- Ajouter les régions à `mapgen.MapData`.
- Intégrer la partition à `CreateGame`.
- Fixer et documenter la source canonique des régions entre carte, état et persistence.
- Mettre à jour `specs/architecture.md`, `specs/gdd.md` et `specs/religieux.md`.

**Vérifications :** `go test ./internal/engine/mapgen/... ./internal/engine/...`

### `[x] 7. Parser spécialisé FR/EN`

**Commit :** `feat(engine/orders): parse special orders into executable commands` (tests écrits en premier dans l’arbre de travail)

**Contenu réalisé :**

- Tester `D C <KIND>` pour défausse, `T C` pour pioche et `P KIND TER` pour jeu.
- Tester le rejet de `J KIND TER` dans le parser deck et la conservation de `J` pour la jonction.
- Tester aliases de kinds FR/EN, casse, commentaires, arités, seeds et kinds interdits.
- Ajouter les DTO de soumission spéciale, distincts des chaînes et investissements.
- Ajouter les codes d’erreur et messages localisés.
- Ajouter la factory des ordres spéciaux.
- Mettre à jour `specs/ordres-speciaux.md` et `specs/architecture.md`.

**Vérifications :** tests orders et `go vet ./...`

### `[~] 8. Registre des cartes et consommation dans Apply`

**Commit prévu :** `feat(engine): add conditional deck card orders` (tests écrits en premier dans l’arbre de travail)

**Contenu attendu :**

- Ajouter `CardDefinition` et un registre déterministe.
- Associer chaque kind bonus à son implémentation d’ordre, dans un fichier dédié.
- Déclarer par carte les saisons autorisées et une condition d’application extensible.
- Pour la v1, traiter `revolt` comme bonus conditionnel à une famine active dans la région.
- Conserver les ratios de balance `plague:1`, `bad_weather:6`, `famine:6` et `revolt:1`, `fair_weather:3`, `abundant_harvest:3`.
- Faire consommer la carte par `Apply` après validation de la saison et des préconditions.
- Défausser les cartes valides, y compris les doublons inopérants.
- Rejeter atomiquement une absence de carte ou une condition non satisfaite.
- Enregistrer les intentions avant agrégation BT/RA.
- Préparer les ordres `P` pour les saisons autorisées sans exiger de noble.
- Faire transiter ces ordres dans `OrdersInput.Special`, distinct de `Chains` et `Winter`.
- Évaluer les conditions sur un snapshot indépendant de l’ordre des joueurs.
- Appliquer leurs intentions avant le ravitaillement et les intentions d’armée.
- Tester l’ordre de phase avant les ordres d’armée et l’absence de noble requis.

**Vérifications :** tests moteur de cartes et d’atomicité

### `[~] 9. Pioche hivernale, défausse et augures`

**Commit prévu :** `feat(engine): resolve winter card draws and calamity scheduling` (tests écrits en premier dans l’arbre de travail)

**Contenu attendu :**

- Ajouter le pipeline deck/main/défausse dans `ResolveWinter` pour les commandes `D C` et `T C`.
- Implémenter la limite de deux `T C`, la main pleine et le tirage après calamité.
- Faire transiter `P` dans la soumission `special`, séparée des investissements d’hiver.
- Appliquer les ordres `P` des saisons d’action avant supply et intentions d’armée.
- Programmer les calamités dans le premier slot libre de l’année suivante.
- Tirer et stocker leur région cible de manière déterministe lors de la programmation.
- Révéler au printemps le kind, la saison et la région de toutes les calamités de l’année ; les augures futures restent cachées.
- Implémenter le remélange déterministe de la défausse complète.
- Révéler l’augure au printemps uniquement.
- Mettre à jour `specs/gdd.md` et `specs/ordres-speciaux.md`.

**Vérifications :** tests moteur, déterminisme et `go vet ./...`

### `[x] 10. Calamités et résolution saisonnière`

**Commit :** `feat(engine): resolve seasonal calamities`

**Contenu réalisé :**

- Tester peste, mauvais temps, famine, suppression du noble mort et armées neutres.
- Préserver les chaînes émises avant le tour de peste, tout en supprimant les chaînes émises pendant le tour courant avec le noble mort.
- Tester la révolte comme carte jouable uniquement lorsqu’une famine affecte sa région.
- Fixer l’ordre des phases : tirage des soumissions deck, calamité, annulation, effets de cartes, ravitaillement, intentions, résolution et effets de fin.
- Intégrer les effets dans `Resolve`, `ResolveWinter` et `ResolveTurn` avant la résolution simultanée des ordres d’armée.
- Utiliser `sourceProduction` et `rationProduction` pour la famine.
- Documenter l’effet militaire exact de la famine avant son implémentation.
- Mettre à jour `specs/gdd.md` et `specs/ordres-speciaux.md`.

**Vérifications :** tests moteur complets, déterminisme et pureté

### `[x] 11. Rumeurs publiques`

**Commit :** `feat(engine): add public winter rumors`

**Contenu attendu :**

- Produire une rumeur uniquement si au moins deux joueurs distincts ont tiré une carte bonus pendant l’hiver.
- Appliquer une probabilité de 50 % avec un RNG dérivé déterministe de la seed, du tour et des joueurs concernés.
- Produire une rumeur indicative du kind tiré sans exposer le joueur, son ordre, son ID de carte ou sa main.
- Ajouter plusieurs formulations FR/EN par kind dans les catalogues existants.
- Ajouter `Rumors` à `WinterReport` et préserver la possibilité d’un filtrage futur par score d’espionnage.
- Ajouter `EventTypeRumor`, `WinterReport.Rumors`, une clé de texte localisable et le kind sans joueur ni ID de carte.
- Tester absence de rumeur avec un seul joueur, tirage raté à 50 %, déterminisme et absence d’identité dans le payload.

**Vérifications :** `go test ./internal/engine/...`

### `[~] 12. Événements et rapports`

**Commit prévu :** `feat(engine): add card and calamity events`

**Contenu attendu :**

- Ajouter les événements deck, augure, bonus, calamité et armée neutre.
- Ne pas réutiliser `EventTypeFamine`.
- Ajouter `Cards` à `WinterReport`.
- Ajouter `Augury` et `SeasonEffects` à `TurnReport`.
- Ajouter les événements et rapports pour pioche, défausse, programmation, calamité, bonus, révolte, armée neutre et mort du noble.
- Ne jamais exposer les IDs internes de cartes dans les rapports publics.
- Tester le tri et la visibilité des rapports.
- Mettre à jour `specs/architecture.md`.

**Vérifications :** `go test ./internal/engine/...`

### `[~] 13. API, store et confidentialité`

**Commit prévu :** `feat(api/store): expose private cards and public auguries`

**Contenu attendu :**

- Faire transiter `special` dans API, store, memory et Firestore.
- Faire porter `special` les ordres `P` de toute saison et `D C`/`T C` de l’hiver.
- Ajouter la main privée dans `StateView` et les augures publiques révélées.
- Vérifier P1/P2, absence de pioche/défausse/IDs internes et restauration complète.
- Masquer pioche, défausse, IDs internes et augures futures.
- Vérifier identité authentifiée, hotseat, multi-parties et restauration.
- Mettre à jour les contrats dans `specs/architecture.md`.

**Vérifications :** tests API/store et `go test ./...`

### `[~] 14. Frontend cartes, régions et augures`

**Commit prévu :** `feat(web): add cards, regions and augury UI`

**Contenu attendu :**

- Ajouter les types JSON, la main et les états d’augure.
- Ajouter le brouillon spécial dans les parcours hotseat et online.
- Factoriser un composant de commandes de deck partagé si les deux parcours divergent.
- Afficher `D C` et `T C` en hiver, et `P KIND TER` pendant toutes les saisons.
- Vérifier le comportement dans les deux modes, sans exiger de noble pour une carte.
- Ajouter toggle régions, contours pointillés, palette et hachures 45°.
- Ajouter la section Rumeurs au `ReportPanel`, avec traductions FR/EN, mobile et accessibilité.

**Vérifications :** `cd web && npm run test && npm run build && npm run lint`

### `[~] 15. Règles joueurs rendues depuis la balance`

**Commit prévu :** `feat(assetgen): render balance-backed player rules`

**Contenu attendu :**

- Transformer `assets/regles-joueurs.md` et `.en.md` en templates paramétrés.
- Injecter depuis `balance.yaml` la taille du deck, la composition par kind, la limite de main, la limite de tirage, les capacités de slots et les bornes d’effets.
- Rendre les nombres avec la même méthode de répartition que le moteur.
- Tester les deux langues et l’absence de valeurs de balance codées en dur.
- Conserver les documents de règles indépendants des projections privées de partie.

**Vérifications :** tests assetgen/API de rendu des règles

### `[x] 16. Documentation joueur et règles finales`

**Commit prévu :** `docs(rules): document special deck and calamities`

**Contenu attendu :**

- Finaliser `specs/ordres-speciaux.md`.
- Mettre à jour `assets/regles-joueurs.md` et `assets/regles-joueurs.en.md` avec les placeholders et leur rendu.
- Documenter exemples jouables, aliases, deck, composition calculée, main, défausse, tirage, slots, régions, effets et annulations.
- Vérifier que les exemples documentés sont acceptés par les tests parser.
- Documenter les futures cartes politiques, religieuses et de succession comme hors périmètre.

**Vérifications :** tests de contrat documentaire/parser

### `[~] 17. Validation finale bout-en-bout`

**Commit prévu :** `test(contract): add end-to-end determinism and privacy suite`

**Contenu attendu :**

- Ajouter les tests de déterminisme complet, pureté, confidentialité et restauration.
- Vérifier l’absence de `NEUTRAL` dans le scoreboard.
- Vérifier les contrats JSON et les exemples FR/EN.
- Exécuter toute la suite avant la review finale.

**Vérifications :**

```text
go test ./...
go vet ./...
cd web && npm run test
cd web && npm run build
cd web && npm run lint
```
