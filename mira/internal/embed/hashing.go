package embed

import (
	"context"
	"hash/fnv"
	"math"
	"regexp"
	"strings"
)

type HashingEmbedder struct {
	dim int
}

func NewHashingEmbedder(dim int) *HashingEmbedder {
	return &HashingEmbedder{dim: dim}
}

var tokenRe = regexp.MustCompile(`[a-zà-ÿ0-9]+`)

func (h *HashingEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, h.dim)
	tokens := tokenRe.FindAllString(strings.ToLower(text), -1)
	for _, tok := range tokens {
		fh := fnv.New32a()
		_, _ = fh.Write([]byte(tok))
		idx := int(fh.Sum32()) % h.dim
		if idx < 0 {
			idx += h.dim
		}
		vec[idx]++
	}

	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vec, nil
	}
	for i, v := range vec {
		vec[i] = float32(float64(v) / norm)
	}
	return vec, nil
}
