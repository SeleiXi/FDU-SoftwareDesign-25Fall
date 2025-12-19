package spellcheck

import (
	"sort"
	"strings"

	"github.com/agnivade/levenshtein"
)

// ThirdPartyAdapter implements Checker using the github.com/agnivade/levenshtein library.
// This demonstrates dependency isolation by wrapping a third-party library for distance calculation.
type ThirdPartyAdapter struct {
	dictionary []string
}

// NewThirdPartyAdapter creates a new adapter with the default dictionary.
func NewThirdPartyAdapter() *ThirdPartyAdapter {
	return &ThirdPartyAdapter{dictionary: defaultDictionary}
}

// Check validates a word and returns suggestions using the third-party levenshtein library.
func (a *ThirdPartyAdapter) Check(word string) (bool, []string) {
	if a == nil {
		return true, nil
	}
	w := strings.ToLower(word)

	// Check for exact match
	for _, dictWord := range a.dictionary {
		if dictWord == w {
			return true, nil
		}
	}

	// Find suggestions using third-party library
	type candidate struct {
		word string
		dist int
	}
	var candidates []candidate
	for _, dictWord := range a.dictionary {
		// Use the third-party library here
		dist := levenshtein.ComputeDistance(w, dictWord)
		if dist <= 2 {
			candidates = append(candidates, candidate{word: dictWord, dist: dist})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].dist == candidates[j].dist {
			return candidates[i].word < candidates[j].word
		}
		return candidates[i].dist < candidates[j].dist
	})

	limit := 3
	if len(candidates) < limit {
		limit = len(candidates)
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, candidates[i].word)
	}

	return false, result
}
