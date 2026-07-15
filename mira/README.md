# mira — TP #4

## Ce qui a changé

- **Stockage** : `internal/notes.PostgresStore` est maintenant la seule
  source de vérité (transactions sur `notes` + `note_tags`, tags mergés à
  l'enrichissement sans écraser les tags saisis par l'utilisateur).
- **API branchée sur Postgres** : `internal/http/handlers` n'utilise plus
  `MemoryStore`, il parle à `notes.NoteStore`.
- **Enrichissement async** : `internal/enrichment.Enricher` est un pool de
  workers bornés (4 par défaut) qui consomment un channel de jobs. Chaque
  job a un timeout (10s par défaut) via `context.WithTimeout`. La création
  et l'édition (PATCH) publient un job et répondent immédiatement ; le
  champ `enrichment_status` passe `pending` → `done`/`failed`.
- **Embeddings** : `internal/embed.Embedder` est une interface. Par défaut
  (`MIRA_EMBEDDINGS_URL` absent) on utilise un hashing embedder local
  dépendance-zéro (`HashingEmbedder`, dimension 384) pour que tout tourne
  hors-ligne. Pour un vrai modèle sémantique, set `MIRA_EMBEDDINGS_URL` +
  `MIRA_EMBEDDINGS_API_KEY` (compatible format OpenAI `/embeddings`) et
  adapte la dimension `vector(384)` dans la migration si besoin.
- **Recherche hybride** : `SearchHybrid` combine `ts_rank` (full-text,
  index GIN) et la similarité cosinus pgvector (`<=>`, index ivfflat) avec
  une moyenne pondérée 50/50. Si l'embedding de la requête échoue/timeout,
  fallback silencieux en full-text seul.
- **CLI** : `cmd/mira` n'écrit plus jamais directement sur le disque ou en
  base ; il passe par `internal/httpclient` qui appelle l'API HTTP. C'est
  le seul moyen de garantir que chaque note créée/modifiée déclenche
  l'enrichissement.

## Hypothèse de nommage

Le code fourni mélangeait deux modules Go (`example.com/m/v2` pour l'API,
`mira` pour le store Postgres + CLI déjà commencés). J'ai tout unifié sous
un seul module `mira`, qui semble être le nom du projet visé par le CLI.
Si ton repo réel utilise un autre chemin de module, un `sed` sur les
imports suffit.

## Lancer

```bash
export MIRA_DATABASE_URL="postgres://user:pass@localhost:5432/mira?sslmode=disable"
go mod tidy
go run ./cmd/api

# dans un autre terminal
go run ./cmd/mira add "Titre" "Contenu avec plusieurs mots pour tester l'enrichissement"
go run ./cmd/mira list
go run ./cmd/mira search "contenu"
```

## Limites connues / suite possible

- Le hashing embedder n'a aucune vraie sémantique (deux textes proches en
  sens mais différents en mots ne seront pas proches en vecteur) — c'est
  un stub pour valider la plomberie pgvector, pas un modèle de prod.
- Les heuristiques de tags/résumé/score (`internal/enrichment/heuristics.go`)
  sont volontairement simples ; à remplacer par un vrai appel LLM si besoin.
- L'index `ivfflat` s'entraîne mieux avec des données déjà présentes ;
  pense à `REINDEX INDEX idx_note_embeddings_vector;` une fois quelques
  centaines de notes insérées.
