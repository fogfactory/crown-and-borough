# Prompt : persistance, sauvegarde et restauration d'une partie

```
Tu travailles sur "Crown & Borough", un jeu de stratégie asynchrone par tours.
L'API REST complète existe (P3.1 : chi, multi-parties en mémoire), l'auth
(P3.2 : sessions en mémoire, tokens). Le moteur est pur, déterministe et
sérialisable (round-trip JSON de GameState testé dès P1.1/P1.3).
Références : specs/roadmap.md (P3.3 — "Persistance : JSON d'abord (1 fichier
par partie), migration Postgres/sqlc ensuite") et specs/architecture.md (§5).

PÉRIMÈTRE : persistance des PARTIES (état + méta) dans un répertoire de
données, sauvegarde après chaque mutation, restauration au démarrage.
Les SESSIONS (P3.2) restent en mémoire (choix documenté : un redémarrage
déconnecte les joueurs, mais les parties survivent — ils se réinscrivent
et reprennent leur slot par nom + code d'invitation, cf. P3.2). La
migration Postgres est HORS périmètre.

RÈGLE DE CODE : code EXCLUSIVEMENT en anglais (identifiants, commentaires,
messages, enums). Seules les chaînes de contenu de jeu (noms, labels UI)
sont en français.

1. FORMAT (1 fichier par partie) :
   - Répertoire : variable d'environnement DATA_DIR (défaut : ./data),
     créé au démarrage si absent
   - Un fichier par partie : data/game-<id>.json — contenu :
        { "id", "name", "seed", "status", "winner", "inviteCode",
          "createdAt", "updatedAt",
          "players": [{ "id", "globalPlayerId", "name", "color" }],
          "turn", "season",
          "state": <GameState sérialisé tel quel, round-trip JSON existant>,
           "history": <anneau d'historique des états passés>,
          "reports": [<TurnReport...>],
          "submissions": { playerID: <OrdersInput> } }
     (soumissions en cours de tour persistées aussi : un redémarrage en
     plein tour ne perd pas les ordres déjà soumis)
   - Les IDs de parties sont des UUID (crypto/rand, v4) — pas de collision
     entre fichiers
   - Version du format : champ "version": 1 en tête (migration future)

2. SAUVEGARDE (interne, atomique) :
   - Après CHAQUE mutation d'une partie (création, join, soumission,
     résolution, élimination) : écriture dans un fichier temporaire du
     même répertoire (game-<id>.json.tmp) puis rename atomique (os.Rename)
     — jamais d'écriture partielle visible
   - Sérialisation sous le même mutex que la mutation (aucune écriture
     concurrente sur le même fichier)
   - fsync du fichier avant le rename (durabilité raisonnable au MVP)
   - Les fichiers .tmp orphelins (crash pendant l'écriture) sont nettoyés
     au démarrage (glob *.tmp → suppression)

3. RESTAURATION (au démarrage) :
   - Chargement de TOUS les fichiers game-*.json du répertoire → map des
     parties (même structure que P3.1)
- Fichier corrompu (JSON invalide, version inconnue, état incohérent
      vérifié par les invariants de base : troupes sur territoires existants,
      IDs uniques, players cohérents) → renommé game-<id>.json.corrupt
     (conservé pour debug), log d'erreur, la partie est perdue mais le
     serveur DÉMARRE (une partie corrompue ne bloque pas les autres)
   - Vérification croisée : la session d'inviteCode reste unique APRÈS le
     chargement (duplication de code → garder la première, log la seconde)

4. INTÉGRATION (internal/server) :
   - Le store de parties (P3.1) prend un paramètre DataDir ; les méthodes
     mutatrices (create/join/submit/resolve) sauvegardent avant de rendre
     la main ; les méthodes de lecture ne sauvegardent jamais
   - POST /api/games/{id}/join et les soumissions : la sauvegarde fait
     partie de la même opération atomique en mémoire (mutex) + disque
   - Le chemin de la résolution (P3.1) : le rapport est stocké en mémoire
     PUIS le fichier est réécrit une seule fois (pas une sauvegarde par
     sous-étape)
   - Aucune I/O disque dans le moteur (internal/engine reste pur) : la
     persistance vit uniquement dans internal/server

5. TESTS (internal/server/persist_test.go) :
   - Sauvegarde : création → fichier présent, contenu JSON valide ;
     soumission + résolution → fichier réécrit (état + rapports + 
     soumissions à jour)
   - Restauration : dossier avec 2 parties (une en cours de tour avec des
     soumissions enregistrées) → démarrage → les 2 parties sont
     disponibles, l'état et les soumissions sont intacts ; une partie
     reprend le tour où elle était (soumissions non perdues)
   - Atomicité : crash simulé (écrire un .tmp et planter avant rename —
     teste la logique de nettoyage) → au démarrage suivant, la partie
     précédente est intacte, le .tmp a disparu
   - Corrompu : fichier invalide → renommé .corrupt, serveur démarre, les
     autres parties chargées ; version inconnue → idem
   - Concurrence (-race) : mutations simultanées sur la même partie →
     fichier final cohérent (jamais de .tmp résiduel après l'opération)
   - REDÉMARRAGE COMPLET : jouer 3 tours → restart → les 3 rapports sont
     consultables, le 4e tour se joue normalement ; deux joueurs de
     navigateurs différents (P3.2) reprennent leur slot par nom + code
     après réinscription
   - Invariant : les sessions meurent au restart (401) mais les parties
     survivent (reprise de slot par nom + code)

Critères d'acceptation :
- make test passe (avec -race) ; make vet passe
- Un redémarrage du serveur ne perd AUCUNE partie (état, rapports,
  soumissions en cours) ; les fichiers sont lisibles et éditables à la main
  (format JSON simple, documenté)
- Le moteur (internal/engine) n'est PAS modifié

Note : documente dans ta réponse finale les choix ouverts que tu as tranchés
(fichier par partie vs un fichier unique, soumissions persistées, sessions
en mémoire, .corrupt conservé, fsync) pour mise à jour des specs (ne modifie
pas les specs toi-même). Ne commite pas : je m'en charge.
```
