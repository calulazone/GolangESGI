# Projet MIRA — Gestionnaire de Notes (Go) - GIRARD Lucas

Ce dépôt contient les différentes implémentations du projet **MIRA**, un gestionnaire de notes intelligent écrit en Go.

Précisions : Je préfère être honnête, je me suis aidé de l'IA sur pas mal de points pendant la réalisation de ce projet (notamment sur l'embedding, le MCP et la structure) car je trouvais le temps assez limité et les ressources en ligne de mauvaises qualités (la doc go est pas terrible par exemple). J'ai néanmoins pris le temps de comprendre chaque chose.

## Architecture du projet

Le projet est divisé en deux grandes parties :

### 1. Version Actuelle : `/mira` (API + MCP + CLI)
Cette version utilise une architecture client-serveur.
*   **MIRA API (`/mira/cmd/api`)** : API REST connectée à une base de données PostgreSQL. Elle gère la création, la lecture, la recherche et l'enrichissement asynchrone des notes (résumés, tags).
*   **MIRA MCP (`/mira/cmd/mira-mcp`)** : Serveur MCP (Model Context Protocol) permettant de connecter l'API de notes à des IA (comme Claude ou Gemini) pour leur donner la capacité d'ajouter, lire ou chercher des notes.

### 2. Ancienne Version : `/mira_old` (CLI Direct + API In-Memory)
Cette version historique montre les premières étapes du développement :
*   **MIRA CLI Direct (`/mira_old/main.go`)** : Outil CLI qui écrit et lit directement dans un fichier JSONL local (`~/.mira/notes.jsonl`) ou dans une base de données PostgreSQL sans passer par une API intermédiaire.
*   **MIRA API In-Memory (`/mira_old/api`)** : Squelette d'API REST en Go stockant les notes temporairement en mémoire vive (mais j'ai déplacé ça sur /mira).

---

## Comment lancer les projets

### Prérequis
*   Go (version `1.21` ou supérieure installée)
*   PostgreSQL

---

### Lancement de la Version Actuelle (`/mira`)

#### 1. Configurer la base de données et lancer l'API
Définissez la variable d'environnement `MIRA_DATABASE_URL` et lancez le serveur d'API :
```powershell
# Windows PowerShell
$env:MIRA_DATABASE_URL="postgres://mira:mira@localhost:5432/mira?sslmode=disable"
cd mira
go run ./cmd/api
```

#### 2. Lancer le serveur MCP (Model Context Protocol)
Assurez-vous que l'API est lancée, puis démarrez le serveur MCP :
```powershell
cd mira
go run ./cmd/mira-mcp
```
Vous pouvez également l'enregistrer dans vos clients MCP (comme Claude Desktop) à l'aide du fichier de configuration `.mcp.json` disponible à la racine de `/mira`.

---

### Lancement de l'Ancienne Version (`/mira_old`)

#### 1. Utiliser le CLI Direct
Par défaut, ce CLI stocke vos notes localement dans un fichier JSONL situé dans votre répertoire utilisateur (`~/.mira/notes.jsonl`), aucune base de données n'est requise.

```powershell
cd mira_old
# Lister les notes
go run main.go list

# Ajouter une note
go run main.go add "Titre de la note" "Contenu textuel de la note"

# Rechercher une note
go run main.go search "Titre"
```

## Démo (Vidéos)

### Vidéo CLI
<video src="videos/Vidéo-CLI.mp4" controls width="100%"></video>

### Vidéo API
<video src="videos/Vidéo-API.mp4" controls width="100%"></video>

### Vidéo MCP
<video src="videos/Vidéo-MCP.mp4" controls width="100%"></video>
