package platform

import (
	"fmt"

	"github.com/atotto/clipboard"
)

// writeClipboard is the OS clipboard writer. Tests replace it so they do not
// need a real clipboard.
var writeClipboard = clipboard.WriteAll

// WriteClipboard copies text to the system clipboard.
func WriteClipboard(text string) error {
	if text == "" {
		return fmt.Errorf("nothing to copy")
	}
	if err := writeClipboard(text); err != nil {
		return fmt.Errorf("write clipboard: %w", err)
	}
	return nil
}
