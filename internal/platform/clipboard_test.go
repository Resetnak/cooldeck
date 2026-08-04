package platform

import (
	"errors"
	"testing"
)

func TestWriteClipboardRejectsEmpty(t *testing.T) {
	if err := WriteClipboard(""); err == nil {
		t.Fatal("empty clipboard write should fail")
	}
}

func TestWriteClipboardUsesWriter(t *testing.T) {
	orig := writeClipboard
	t.Cleanup(func() { writeClipboard = orig })

	var got string
	writeClipboard = func(text string) error {
		got = text
		return nil
	}
	if err := WriteClipboard("hello"); err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("wrote %q", got)
	}

	writeClipboard = func(string) error { return errors.New("boom") }
	if err := WriteClipboard("x"); err == nil {
		t.Fatal("expected writer error")
	}
}
