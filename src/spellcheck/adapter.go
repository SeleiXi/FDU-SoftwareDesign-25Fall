package spellcheck

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// LanguageToolAdapter implements Checker using the LanguageTool API directly.
// We use direct HTTP calls because the github.com/bas24/languagetool library
// has issues parsing responses for certain words (e.g., "helo").
type LanguageToolAdapter struct {
	fastDict map[string]struct{}
}

// NewLanguageToolAdapter creates a new adapter with the default dictionary for acceleration.
func NewLanguageToolAdapter() *LanguageToolAdapter {
	dict := make(map[string]struct{})
	for _, w := range defaultDictionary {
		dict[strings.ToLower(w)] = struct{}{}
	}
	return &LanguageToolAdapter{fastDict: dict}
}

type ltResponse struct {
	Matches []struct {
		Replacements []struct {
			Value string `json:"value"`
		} `json:"replacements"`
	} `json:"matches"`
}

// Check validates a word.
// 1. Fast path: Check local dictionary.
// 2. Slow path: Call LanguageTool API.
func (a *LanguageToolAdapter) Check(word string) (bool, []string) {
	if a == nil {
		return true, nil
	}
	w := strings.ToLower(word)

	// 1. Acceleration: Check local dictionary
	if _, ok := a.fastDict[w]; ok {
		return true, nil
	}

	// 2. Remote Check: Use LanguageTool API directly
	resp, err := http.PostForm("https://api.languagetool.org/v2/check",
		url.Values{"text": {word}, "language": {"en-US"}})
	if err != nil {
		// Fallback: if API fails, assume correct to avoid blocking user.
		return true, nil
	}
	defer resp.Body.Close()

	var res ltResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return true, nil
	}

	if len(res.Matches) > 0 {
		// Found issues
		var suggestions []string
		// We only care about the first match since we sent a single word
		for _, repl := range res.Matches[0].Replacements {
			suggestions = append(suggestions, repl.Value)
		}
		return false, suggestions
	}

	return true, nil
}
