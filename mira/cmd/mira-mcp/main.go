package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/internal/httpclient"
)

const (
	defaultLimit = 10
	maxLimit     = 100
	apiTimeout   = 15 * time.Second
)

func main() {
	// All logging goes to stderr - stdout is reserved for the MCP
	// protocol on the stdio transport.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	apiURL := os.Getenv("MIRA_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	client := httpclient.New(apiURL)
	logger.Info("mira-mcp starting", "api_url", apiURL)

	server := mcp.NewServer(&mcp.Implementation{Name: "mira", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "search_notes",
		Description: "Recherche des notes mira par pertinence, en combinant recherche " +
			"plein-texte et similarite vectorielle (recherche hybride). Utilise cet outil " +
			"quand l'utilisateur demande de retrouver une note existante par sujet ou par " +
			"mots-cles, plutot que par identifiant exact.",
	}, withRecover(logger, "search_notes", searchNotesHandler(client, logger)))

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_note",
		Description: "Retourne une note complete de mira a partir de son identifiant : " +
			"titre, contenu integral, tags, resume et statut d'enrichissement " +
			"(pending/done/failed). Utilise cet outil quand tu connais deja l'identifiant " +
			"exact de la note, par exemple apres un search_notes ou un list_recent_notes.",
	}, withRecover(logger, "get_note", getNoteHandler(client, logger)))

	mcp.AddTool(server, &mcp.Tool{
		Name: "add_note",
		Description: "Cree une nouvelle note dans mira avec un titre et un contenu. " +
			"Declenche automatiquement l'enrichissement asynchrone de la note (tags, " +
			"resume, embedding) cote serveur ; la note est renvoyee avec le statut " +
			"'pending' immediatement, l'enrichissement suit en arriere-plan.",
	}, withRecover(logger, "add_note", addNoteHandler(client, logger)))

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_recent_notes",
		Description: "Liste les notes mira les plus recemment creees, triees du plus " +
			"recent au plus ancien. Utilise cet outil pour un apercu general recent, " +
			"plutot que search_notes qui necessite une requete de recherche precise.",
	}, withRecover(logger, "list_recent_notes", listRecentNotesHandler(client, logger)))

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Error("mira-mcp stopped", "err", err)
		os.Exit(1)
	}
}

func withRecover[In, Out any](
	logger *slog.Logger,
	toolName string,
	fn func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (res *mcp.CallToolResult, out Out, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered in tool handler", "tool", toolName, "panic", r)
				err = fmt.Errorf("%s: internal error", toolName)
			}
		}()
		return fn(ctx, req, in)
	}
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func friendlyErr(toolName string, err error) error {
	var notFound *httpclient.NotFoundError
	if errors.As(err, &notFound) {
		return fmt.Errorf("%s: note introuvable", toolName)
	}
	return fmt.Errorf("%s: l'appel a l'API mira a echoue (%v)", toolName, err)
}

type SearchNotesInput struct {
	Query string `json:"query" jsonschema:"le texte a rechercher ; recherche hybride full-text et vectorielle sur le titre et le contenu des notes"`
	Limit int    `json:"limit,omitempty" jsonschema:"nombre maximum de resultats a retourner (defaut 10, maximum 100)"`
}

type NoteList struct {
	Notes []httpclient.Note `json:"notes" jsonschema:"les notes trouvees, des plus pertinentes/recentes aux moins pertinentes/recentes"`
	Count int               `json:"count" jsonschema:"nombre de notes retournees"`
}

func searchNotesHandler(client *httpclient.Client, logger *slog.Logger) func(context.Context, *mcp.CallToolRequest, SearchNotesInput) (*mcp.CallToolResult, NoteList, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in SearchNotesInput) (*mcp.CallToolResult, NoteList, error) {
		query := strings.TrimSpace(in.Query)
		if query == "" {
			return nil, NoteList{}, errors.New("search_notes: le parametre 'query' est requis et ne peut pas etre vide")
		}

		callCtx, cancel := context.WithTimeout(ctx, apiTimeout)
		defer cancel()

		list, err := client.Search(callCtx, query, clampLimit(in.Limit))
		if err != nil {
			logger.Warn("search_notes failed", "query", query, "err", err)
			return nil, NoteList{}, friendlyErr("search_notes", err)
		}
		return nil, NoteList{Notes: list, Count: len(list)}, nil
	}
}

type GetNoteInput struct {
	ID string `json:"id" jsonschema:"identifiant unique de la note a recuperer (par exemple celui retourne par add_note ou search_notes)"`
}

func getNoteHandler(client *httpclient.Client, logger *slog.Logger) func(context.Context, *mcp.CallToolRequest, GetNoteInput) (*mcp.CallToolResult, httpclient.Note, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in GetNoteInput) (*mcp.CallToolResult, httpclient.Note, error) {
		id := strings.TrimSpace(in.ID)
		if id == "" {
			return nil, httpclient.Note{}, errors.New("get_note: le parametre 'id' est requis et ne peut pas etre vide")
		}

		callCtx, cancel := context.WithTimeout(ctx, apiTimeout)
		defer cancel()

		note, err := client.GetNote(callCtx, id)
		if err != nil {
			logger.Warn("get_note failed", "id", id, "err", err)
			return nil, httpclient.Note{}, friendlyErr("get_note", err)
		}
		return nil, *note, nil
	}
}

type AddNoteInput struct {
	Title   string   `json:"title" jsonschema:"titre court de la note"`
	Content string   `json:"content" jsonschema:"contenu integral de la note"`
	Tags    []string `json:"tags,omitempty" jsonschema:"tags optionnels a associer a la note, en plus de ceux generes automatiquement par l'enrichissement"`
}

func addNoteHandler(client *httpclient.Client, logger *slog.Logger) func(context.Context, *mcp.CallToolRequest, AddNoteInput) (*mcp.CallToolResult, httpclient.Note, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in AddNoteInput) (*mcp.CallToolResult, httpclient.Note, error) {
		title := strings.TrimSpace(in.Title)
		content := strings.TrimSpace(in.Content)
		if title == "" {
			return nil, httpclient.Note{}, errors.New("add_note: le parametre 'title' est requis et ne peut pas etre vide")
		}
		if content == "" {
			return nil, httpclient.Note{}, errors.New("add_note: le parametre 'content' est requis et ne peut pas etre vide")
		}

		callCtx, cancel := context.WithTimeout(ctx, apiTimeout)
		defer cancel()

		note, err := client.CreateNote(callCtx, title, content, in.Tags)
		if err != nil {
			logger.Warn("add_note failed", "title", title, "err", err)
			return nil, httpclient.Note{}, friendlyErr("add_note", err)
		}
		return nil, *note, nil
	}
}

type ListRecentNotesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"nombre maximum de notes a retourner (defaut 10, maximum 100)"`
}

func listRecentNotesHandler(client *httpclient.Client, logger *slog.Logger) func(context.Context, *mcp.CallToolRequest, ListRecentNotesInput) (*mcp.CallToolResult, NoteList, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListRecentNotesInput) (*mcp.CallToolResult, NoteList, error) {
		callCtx, cancel := context.WithTimeout(ctx, apiTimeout)
		defer cancel()

		list, err := client.ListNotes(callCtx, clampLimit(in.Limit))
		if err != nil {
			logger.Warn("list_recent_notes failed", "err", err)
			return nil, NoteList{}, friendlyErr("list_recent_notes", err)
		}
		return nil, NoteList{Notes: list, Count: len(list)}, nil
	}
}
