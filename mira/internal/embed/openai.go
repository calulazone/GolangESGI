package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTPEmbedder calls an OpenAI-compatible /embeddings endpoint. Most hosted
// embedding models (OpenAI, Mistral, local vLLM/Ollama servers) speak this
// same request/response shape.
type HTTPEmbedder struct {
	url    string
	apiKey string
	model  string
	client *http.Client
}

func NewHTTPEmbedder(url, apiKey, model string) *HTTPEmbedder {
	return &HTTPEmbedder{url: url, apiKey: apiKey, model: model, client: &http.Client{}}
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (e *HTTPEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embeddingRequest{Model: e.model, Input: text})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings request failed: status %d", resp.StatusCode)
	}

	var out embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("embeddings response had no data")
	}
	return out.Data[0].Embedding, nil
}
