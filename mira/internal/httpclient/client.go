// Package httpclient is the CLI's and the MCP server's only way of talking
// to note storage. Going through the API (instead of the JSONL/Postgres
// store directly) is what guarantees every created or edited note
// triggers enrichment.
//
// Every method takes a context.Context so callers (the CLI, the MCP
// server) can enforce their own per-call timeout via
// context.WithTimeout - the client itself sets no default timeout beyond
// what the context provides.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{}}
}

type Note struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	Tags             []string `json:"tags,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	EnrichmentStatus string   `json:"enrichment_status"`
	CreatedAt        string   `json:"created_at"`
}

type envelope struct {
	Success bool            `json:"success"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`
}

// NotFoundError is returned when the API responds with a 404-shaped
// "not found" error, so callers (e.g. the MCP server) can distinguish
// "no such note" from a generic failure and report it cleanly.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

func (c *Client) CreateNote(ctx context.Context, title, content string, tags []string) (*Note, error) {
	body, _ := json.Marshal(map[string]any{"title": title, "content": content, "tags": tags})
	var n Note
	if err := c.do(ctx, http.MethodPost, "/api/v1/notes", bytes.NewReader(body), &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (c *Client) ListNotes(ctx context.Context, limit int) ([]Note, error) {
	var list []Note
	path := fmt.Sprintf("/api/v1/notes?limit=%d", limit)
	if err := c.do(ctx, http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) GetNote(ctx context.Context, id string) (*Note, error) {
	var n Note
	path := "/api/v1/notes/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]Note, error) {
	var list []Note
	path := fmt.Sprintf("/api/v1/search?q=%s&limit=%d", url.QueryEscape(query), limit)
	if err := c.do(ctx, http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("calling mira API: %w", err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode mira API response: %w", err)
	}
	if !env.Success {
		if resp.StatusCode == http.StatusNotFound {
			return &NotFoundError{Message: env.Error}
		}
		return fmt.Errorf("mira API error: %s", env.Error)
	}
	if out == nil || env.Data == nil {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}
