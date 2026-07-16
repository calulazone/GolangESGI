# mira — TP #4


## Serveur MCP (`cmd/mira-mcp`)

### Installation

```bash
go mod tidy
go build -o mira-mcp ./cmd/mira-mcp
```

Prérequis : l'API (`go run ./cmd/api`) doit tourner avant de lancer
`mira-mcp`

### Tools exposés

| Tool | Paramètres | Rôle |
| --- | --- | --- |
| `search_notes` | `query` (string, requis), `limit` (int, défaut 10, max 100) | recherche hybride full-text + vectorielle |
| `get_note` | `id` (string, requis) | note complète : contenu, tags, résumé, statut |
| `add_note` | `title`, `content` (requis), `tags` (optionnel) | crée une note, déclenche l'enrichissement async |

Chaque tool a une description dédiée à l'usage de l'agent.

### Variables d'environnement

| Variable | Défaut | Rôle |
| --- | --- | --- |
| `MIRA_API_URL` | `http://localhost:8080` | URL de l'API mira à utiliser |

### Enregistrer le serveur dans Claude Code

Un fichier `.mcp.json` est fourni à la racine du repo :

```json
{
  "mcpServers": {
    "mira": {
      "command": "go",
      "args": ["run", "./cmd/mira-mcp"],
      "env": {
        "MIRA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Exemples de prompts

Une fois le serveur enregistré et l'API + Postgres lancés :

> Ajoute moi une note dans Mira sur une recette de tiramisu.

> Liste moi toutes mes notes.

> Recherche une note sur la tarte aux pommes


## Lancer

```bash
$env:MIRA_DATABASE_URL="postgres://mira:mira@localhost:5432/mira?sslmode=disable" 
go mod tidy
go run ./cmd/api

# dans un autre terminal
go run ./cmd/mira add "Titre" "Contenu avec plusieurs mots"
go run ./cmd/mira list
go run ./cmd/mira search "contenu"
```

## Limites connues

- Le hashing embedder n'a aucune vraie sémantique, je n'ai pas connecté un vrai modèle pour vectoriser.