package embed

import "context"

const Dimension = 384

// Embedder converts text into a vector of length Dimension.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

func FromEnv() Embedder {
	return NewHashingEmbedder(Dimension)
}
