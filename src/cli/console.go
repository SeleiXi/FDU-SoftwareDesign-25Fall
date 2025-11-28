package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Console wraps standard IO for prompting.
type Console struct {
	reader *bufio.Reader
	writer io.Writer
}

// NewConsole constructs a console facade.
func NewConsole(in io.Reader, out io.Writer) *Console {
	return &Console{
		reader: bufio.NewReader(in),
		writer: out,
	}
}

// ReadLine reads a line without newline characters.
func (c *Console) ReadLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// Print writes raw text.
func (c *Console) Print(text string) {
	fmt.Fprint(c.writer, text)
}

// Println writes a line with newline.
func (c *Console) Println(text string) {
	fmt.Fprintln(c.writer, text)
}

// ConfirmSave prompts user for saving decision.
func (c *Console) ConfirmSave(path string) (bool, error) {
	for {
		c.Print(fmt.Sprintf("文件已修改，是否保存? (y/n) [%s]: ", path))
		answer, err := c.ReadLine()
		if err != nil {
			return false, err
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		switch answer {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			c.Println("请输入 y 或 n")
		}
	}
}
