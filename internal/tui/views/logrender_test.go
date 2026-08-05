package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/domain"
)

func TestLogBlockDoesNotPrintTheLevelTwice(t *testing.T) {
	th := tailTheme(t)
	content := logContent{
		title: "logs",
		lines: []domain.LogLine{
			domain.ParseLogLine("2026-08-05T14:30:16Z INFO GET /healthz 200 in 2ms"),
			domain.ParseLogLine("2026-08-05T14:30:37Z ERROR upstream dependency timed out"),
			// Only a leading level is the badge; one in the middle is content.
			domain.ParseLogLine("2026-08-05T14:30:44Z GET /error 500 in 3ms"),
		},
	}

	body, _, _ := renderLogBlock(th, 120, 12, content, 0)
	plain := ansi.Strip(body)

	if strings.Contains(plain, "INFO  INFO") || strings.Contains(plain, "ERROR ERROR") {
		t.Fatalf("level rendered twice:\n%s", plain)
	}
	if !strings.Contains(plain, "GET /error 500") {
		t.Fatalf("a level word inside the message was eaten:\n%s", plain)
	}
}
