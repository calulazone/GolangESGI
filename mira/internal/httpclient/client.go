// Package httpclient is the CLI's only way of talking to note storage.
// Going through the API (instead of the JSONL/Postgres store directly)
// is what guarantees every created or edited note triggers enrichment.
package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
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

func (c *Client) CreateNote(title, content string, tags []string) (*Note, error) {
	body, _ := json.Marshal(map[string]any{"title": title, "content": content, "tags": tags})
	var n Note
	if err := c.do(http.MethodPost, "/api/v1/notes", bytes.NewReader(body), &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (c *Client) ListNotes() ([]Note, error) {
	var list []Note
	if err := c.do(http.MethodGet, "/api/v1/notes", nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) Search(query string) ([]Note, error) {
	var list []Note
	path := "/api/v1/search?q=" + url.QueryEscape(query)
	if err := c.do(http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) do(method, path string, body io.Reader, out any) error {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !env.Success {
		return fmt.Errorf("api error: %s", env.Error)
	}
	if out == nil || env.Data == nil {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}
