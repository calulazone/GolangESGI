package enrichment

import (
	"sort"
	"strings"
	"unicode"
)

var stopwordsFR = map[string]struct{}{
	"le": {}, "la": {}, "les": {}, "un": {}, "une": {}, "des": {}, "de": {}, "du": {},
	"et": {}, "ou": {}, "a": {}, "à": {}, "au": {}, "aux": {}, "en": {}, "dans": {},
	"pour": {}, "par": {}, "sur": {}, "avec": {}, "sans": {}, "ce": {}, "cette": {},
	"ces": {}, "il": {}, "elle": {}, "ils": {}, "elles": {}, "je": {}, "tu": {}, "nous": {},
	"vous": {}, "que": {}, "qui": {}, "est": {}, "sont": {}, "être": {}, "avoir": {}, "pas": {},
}

func extractTags(content string) []string {
	freq := map[string]int{}
	for _, word := range tokenize(content) {
		if _, stop := stopwordsFR[word]; stop || len(word) < 4 {
			continue
		}
		freq[word]++
	}

	type kv struct {
		word  string
		count int
	}
	ranked := make([]kv, 0, len(freq))
	for w, c := range freq {
		ranked = append(ranked, kv{w, c})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].word < ranked[j].word
	})

	const maxTags = 5
	tags := make([]string, 0, maxTags)
	for i := 0; i < len(ranked) && i < maxTags; i++ {
		tags = append(tags, ranked[i].word)
	}
	return tags
}

func summarize(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	// First sentence, capped to keep summaries short.
	end := strings.IndexAny(content, ".!?")
	summary := content
	if end != -1 {
		summary = content[:end+1]
	}
	const maxLen = 160
	runes := []rune(summary)
	if len(runes) > maxLen {
		summary = string(runes[:maxLen]) + "…"
	}
	return summary
}

func scoreContent(content string) float64 {
	words := tokenize(content)
	if len(words) == 0 {
		return 0
	}
	unique := map[string]struct{}{}
	for _, w := range words {
		unique[w] = struct{}{}
	}
	lengthScore := float64(len(words)) / 200
	if lengthScore > 1 {
		lengthScore = 1
	}
	diversityScore := float64(len(unique)) / float64(len(words))
	score := 0.6*lengthScore + 0.4*diversityScore
	if score > 1 {
		score = 1
	}
	return score
}

func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
