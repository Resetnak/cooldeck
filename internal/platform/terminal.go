package platform

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// applicationUUIDPattern matches Coolify resource UUIDs. The UUID is
// interpolated into a remote shell script, so anything outside this strict
// alphabet must be rejected, not escaped: the value originates from API data
// and is therefore untrusted input.
var applicationUUIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// containerNamePattern matches Docker container names. The names come back
// from docker ps on the server and are interpolated into a second remote
// command, so they pass through the same strict gate as the UUID.
var containerNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// validateSSHHost vets the configured destination. The host is its own argv
// element so the local shell never parses it, but a leading dash would be read
// as an ssh option and embedded whitespace would smuggle extra arguments past
// that guarantee.
func validateSSHHost(sshHost string) (string, error) {
	sshHost = strings.TrimSpace(sshHost)
	if sshHost == "" {
		return "", fmt.Errorf("no ssh_host configured for this instance")
	}
	if strings.HasPrefix(sshHost, "-") || strings.ContainsAny(sshHost, " \t\n") {
		return "", fmt.Errorf("ssh_host %q is not a plain user@host destination", sshHost)
	}
	return sshHost, nil
}

// ContainerListCommand builds the non-interactive SSH call that lists the
// running containers belonging to an application. Coolify's REST API has no
// exec or container endpoint, so the terminal reaches the server the same way
// Coolify itself does: over SSH. Containers are matched by name prefix,
// because Coolify names application containers "<uuid>-<suffix>" (and
// "<service>-<uuid>" for compose services).
func ContainerListCommand(ctx context.Context, sshHost, appUUID string) (*exec.Cmd, error) {
	host, err := validateSSHHost(sshHost)
	if err != nil {
		return nil, err
	}
	if !applicationUUIDPattern.MatchString(appUUID) {
		return nil, fmt.Errorf("application UUID %q contains characters that cannot go into a remote command", appUUID)
	}
	script := fmt.Sprintf(`docker ps --filter name=%s --format '{{.Names}}'`, appUUID)
	return exec.CommandContext(ctx, "ssh", "--", host, script), nil
}

// ParseContainerList reduces docker ps output to valid container names. A line
// that does not look like a container name (MOTD noise, warnings, anything a
// compromised server might inject) is dropped rather than offered to the user,
// because whatever is picked here goes into the next remote command.
func ParseContainerList(output string) []string {
	names := []string{}
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !containerNamePattern.MatchString(line) {
			continue
		}
		names = append(names, line)
	}
	return names
}

// TerminalCommand builds the interactive SSH session that drops the user into
// a shell inside one specific container. bash is preferred, falling back to
// sh for slim images.
func TerminalCommand(sshHost, container string) (*exec.Cmd, error) {
	host, err := validateSSHHost(sshHost)
	if err != nil {
		return nil, err
	}
	if !containerNamePattern.MatchString(container) {
		return nil, fmt.Errorf("container name %q contains characters that cannot go into a remote command", container)
	}
	script := fmt.Sprintf(
		`exec docker exec -it %s sh -c 'command -v bash >/dev/null 2>&1 && exec bash || exec sh'`,
		container)
	return exec.Command("ssh", "-t", "--", host, script), nil
}
