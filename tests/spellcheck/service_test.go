package spellcheck_test

import (
	"testing"

	"softwaredesign/src/spellcheck"
)

func TestSpellCheckLines(t *testing.T) {
	service := spellcheck.NewService(spellcheck.NewThirdPartyAdapter())
	lines := []string{"Please recieve updates"}
	issues := service.CheckLines(lines)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Word != "recieve" {
		t.Fatalf("unexpected word: %s", issues[0].Word)
	}
	if len(issues[0].Suggestions) == 0 || issues[0].Suggestions[0] != "receive" {
		t.Fatalf("missing suggestion for recieve: %+v", issues[0].Suggestions)
	}
}

func TestSpellCheckXML(t *testing.T) {
	service := spellcheck.NewService(spellcheck.NewThirdPartyAdapter())
	entries := []spellcheck.XMLText{{ElementID: "title1", Text: "Rowlling"}}
	issues := service.CheckXMLText(entries)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].ElementID != "title1" {
		t.Fatalf("unexpected element id: %s", issues[0].ElementID)
	}
	if len(issues[0].Suggestions) == 0 {
		t.Fatalf("expected suggestions for misspelling")
	}
}

// MockChecker demonstrates testability by simulating a spell checker without real dependencies.
type MockChecker struct {
	mockResults map[string][]string
}

func (m *MockChecker) Check(word string) (bool, []string) {
	suggestions, ok := m.mockResults[word]
	if ok {
		return false, suggestions
	}
	return true, nil
}

func TestSpellCheckWithMock(t *testing.T) {
	mock := &MockChecker{
		mockResults: map[string][]string{
			"mistak": {"mistake"},
		},
	}
	service := spellcheck.NewService(mock)

	lines := []string{"This is a mistak."}
	issues := service.CheckLines(lines)

	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Word != "mistak" {
		t.Errorf("expected word 'mistak', got '%s'", issues[0].Word)
	}
	if issues[0].Suggestions[0] != "mistake" {
		t.Errorf("expected suggestion 'mistake', got '%s'", issues[0].Suggestions[0])
	}
}
