# API Notes

Squelette d’API REST en Go pour gérer des notes en mémoire.

## Structure

- cmd/api : point d’entrée HTTP
- internal/core : modèles métiers
- internal/http/handlers : handlers et routage
- internal/store : stockage en mémoire

## Routes

- POST /api/v1/notes : créer une note
- GET /api/v1/notes : lister les notes
- GET /api/v1/notes/{id} : récupérer une note
- PATCH /api/v1/notes/{id} : modifier partiellement une note
- DELETE /api/v1/notes/{id} : supprimer une note
- GET /api/v1/search?q=... : rechercher dans le titre ou le contenu

## Exemples curl

Créer une note :

```bash
curl -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Titre","content":"Contenu","tags":["go"]}'
```

Lister les notes :

```bash
curl http://localhost:8080/api/v1/notes
```

Récupérer une note :

```bash
curl http://localhost:8080/api/v1/notes/note-1
```

Rechercher :

```bash
curl "http://localhost:8080/api/v1/search?q=contenu"
```

## Codes d’erreur possibles

- 400 : payload invalide ou paramètres manquants
- 404 : ressource introuvable
- 405 : méthode non autorisée
- 500 : erreur interne

## Lancer l’API

```bash
go run ./cmd/api
```
