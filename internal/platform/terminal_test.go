package platform

import (
	"strings"
	"testing"
)

func TestContainerListCommandBuildsSSHInvocation(t *testing.T) {
	cmd, err := ContainerListCommand(t.Context(), "root@203.0.113.7", "x0kkgoscw0o8k4gochws8g4s")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(cmd.Args, " ")
	if cmd.Args[0] != "ssh" || !strings.Contains(joined, "root@203.0.113.7") {
		t.Fatalf("unexpected command: %q", joined)
	}
	if !strings.Contains(joined, "docker ps") || !strings.Contains(joined, "x0kkgoscw0o8k4gochws8g4s") {
		t.Fatalf("missing docker ps filter: %q", joined)
	}
	if strings.Contains(joined, "-t") {
		t.Fatalf("listing must not allocate a TTY: %q", joined)
	}
}

func TestTerminalCommandBuildsSSHInvocation(t *testing.T) {
	cmd, err := TerminalCommand("root@203.0.113.7", "web-x0kkgoscw0o8k4gochws8g4s")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(cmd.Args, " ")
	if cmd.Args[0] != "ssh" || !strings.Contains(joined, "-t") {
		t.Fatalf("missing ssh -t: %q", joined)
	}
	if !strings.Contains(joined, "docker exec -it web-x0kkgoscw0o8k4gochws8g4s") {
		t.Fatalf("missing docker exec: %q", joined)
	}
}

func TestParseContainerListDropsJunk(t *testing.T) {
	out := "web-abc123\n\nWarning: Permanently added 'host'\ndb-abc123 extra\npg-abc123\n"
	got := ParseContainerList(out)
	want := []string{"web-abc123", "pg-abc123"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ParseContainerList = %v, want %v", got, want)
	}
}

func TestTerminalCommandsRejectBadInput(t *testing.T) {
	hosts := []string{"", "-oProxyCommand=evil", "root@host extra"}
	for _, host := range hosts {
		if _, err := ContainerListCommand(t.Context(), host, "abc123"); err == nil {
			t.Errorf("ContainerListCommand accepted host %q", host)
		}
		if _, err := TerminalCommand(host, "abc123"); err == nil {
			t.Errorf("TerminalCommand accepted host %q", host)
		}
	}
	uuids := []string{"", "abc; rm -rf /", "$(reboot)", "abc'def"}
	for _, uuid := range uuids {
		if _, err := ContainerListCommand(t.Context(), "root@host", uuid); err == nil {
			t.Errorf("ContainerListCommand accepted uuid %q", uuid)
		}
	}
	containers := []string{"", "-leading-dash", "name with space", "a;b", "$(reboot)"}
	for _, c := range containers {
		if _, err := TerminalCommand("root@host", c); err == nil {
			t.Errorf("TerminalCommand accepted container %q", c)
		}
	}
}
