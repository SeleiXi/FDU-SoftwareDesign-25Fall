package spellcheck

import (
	"sort"
	"strings"
	"unicode"
)

// Checker validates words and returns suggestions for corrections.
type Checker interface {
	Check(word string) (bool, []string)
}

// Service orchestrates spell checking across different document types.
type Service struct {
	checker Checker
}

// NewService constructs a spell check service.
func NewService(checker Checker) *Service {
	return &Service{checker: checker}
}

// TextIssue represents a finding in a plain text file.
type TextIssue struct {
	Line        int
	Column      int
	Word        string
	Suggestions []string
}

// XMLIssue represents a finding in an XML element.
type XMLIssue struct {
	ElementID   string
	Word        string
	Suggestions []string
}

// XMLText represents the text content extracted from an XML element.
type XMLText struct {
	ElementID string
	Text      string
}

// CheckLines evaluates each line of a text document.
func (s *Service) CheckLines(lines []string) []TextIssue {
	if s == nil || s.checker == nil {
		return nil
	}
	var issues []TextIssue
	for i, line := range lines {
		for _, pos := range extractWordPositions(line) {
			ok, suggestions := s.checker.Check(pos.word)
			if ok {
				continue
			}
			issues = append(issues, TextIssue{
				Line:        i + 1,
				Column:      pos.column,
				Word:        pos.word,
				Suggestions: suggestions,
			})
		}
	}
	return issues
}

// CheckXMLText evaluates the text content of XML elements.
func (s *Service) CheckXMLText(nodes []XMLText) []XMLIssue {
	if s == nil || s.checker == nil {
		return nil
	}
	var issues []XMLIssue
	for _, entry := range nodes {
		for _, word := range extractWords(entry.Text) {
			ok, suggestions := s.checker.Check(word)
			if ok {
				continue
			}
			issues = append(issues, XMLIssue{
				ElementID:   entry.ElementID,
				Word:        word,
				Suggestions: suggestions,
			})
		}
	}
	return issues
}

// SimpleChecker is a small dictionary-backed checker suitable for offline use.
type SimpleChecker struct {
	words map[string]struct{}
}

// NewSimpleChecker builds the default simple checker.
func NewSimpleChecker() *SimpleChecker {
	checker := &SimpleChecker{words: map[string]struct{}{}}
	for _, w := range defaultDictionary {
		checker.words[w] = struct{}{}
	}
	return checker
}

// Check validates a word against the dictionary, returning candidate suggestions.
func (c *SimpleChecker) Check(word string) (bool, []string) {
	if c == nil {
		return true, nil
	}
	w := strings.ToLower(word)
	if _, ok := c.words[w]; ok {
		return true, nil
	}
	candidates := collectSuggestions(w, c.words)
	return false, candidates
}

var defaultDictionary = []string{
	"a", "an", "and", "api", "append", "author", "book", "bookstore", "child", "command",
	"config", "content", "data", "delete", "editor", "element", "english", "file", "harry",
	"hello", "italian", "language", "list", "load", "log", "minute", "minutes", "node",
	"occurred", "parent", "please", "potter", "price", "receive", "redo", "root", "rowling",
	"save", "spell", "text", "title", "tree", "undo", "updates", "with", "world", "xml",
}

type wordPosition struct {
	word   string
	column int
}

func extractWordPositions(line string) []wordPosition {
	var result []wordPosition
	var builder strings.Builder
	column := 1
	startColumn := 1
	for _, r := range line {
		if unicode.IsLetter(r) {
			if builder.Len() == 0 {
				startColumn = column
			}
			builder.WriteRune(r)
		} else {
			if builder.Len() > 0 {
				result = append(result, wordPosition{word: builder.String(), column: startColumn})
				builder.Reset()
			}
		}
		column++
	}
	if builder.Len() > 0 {
		result = append(result, wordPosition{word: builder.String(), column: startColumn})
	}
	return result
}

func extractWords(text string) []string {
	tokens := extractWordPositions(text)
	words := make([]string, len(tokens))
	for i, t := range tokens {
		words[i] = t.word
	}
	return words
}

func collectSuggestions(word string, dictionary map[string]struct{}) []string {
	type candidate struct {
		word string
		dist int
	}
	var candidates []candidate
	for dictWord := range dictionary {
		dist := levenshtein(word, dictWord)
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
	return result
}

func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ra := range ar {
		curr[0] = i + 1
		for j, rb := range br {
			cost := 1
			if ra == rb {
				cost = 0
			}
			ins := curr[j] + 1
			del := prev[j+1] + 1
			sub := prev[j] + cost
			curr[j+1] = min(ins, min(del, sub))
		}
		copy(prev, curr)
	}
	return prev[len(br)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
