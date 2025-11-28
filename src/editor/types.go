package editor

// Type enumerates supported editor kinds.
type Type string

const (
	// TypeText represents the plain text editor.
	TypeText Type = "text"
	// TypeXML represents the XML editor.
	TypeXML Type = "xml"
)

// Editor exposes the common behaviour shared by all editors.
type Editor interface {
	Path() string
	Name() string
	Type() Type
	IsModified() bool
	SetModified(bool)
	Content() (string, error)
	Undo() error
	Redo() error
}

// TextDocument offers plain text editing commands.
type TextDocument interface {
	Editor
	Lines() []string
	SetLines([]string)
	Append(string) error
	Insert(line, col int, text string) error
	Delete(line, col, length int) error
	Replace(line, col, length int, text string) error
	Show(start, end int) ([]string, error)
}

// XMLTreeEditor describes XML specific operations.
type XMLTreeEditor interface {
	Editor
	InsertBefore(tag, newID, targetID string, text *string) error
	AppendChild(tag, newID, parentID string, text *string) error
	EditID(oldID, newID string) error
	EditText(elementID string, text string) error
	DeleteElement(elementID string) error
	TreeString() string
	TextNodes() []XMLTextNode
	RootAttributes() map[string]string
}

// XMLTextNode describes an XML element with text content for spell checking.
type XMLTextNode struct {
	ElementID string
	Text      string
}
