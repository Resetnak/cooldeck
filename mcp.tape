# mcp.tape - VHS script that renders the MCP demo (assets/mcp.gif).
# Regenerate from the repo root:  vhs mcp.tape
# Needs: vhs, bash, go and jq - the hidden setup rebuilds bin/cooldeck, so the
# recording always shows the working tree rather than a stale binary. jq only
# pretty-prints the responses; the protocol traffic itself is unedited.
# Everything runs against `--demo`: no Coolify instance, no network, no token.
# COOLDECK_CONFIG_DIR points at a throwaway directory under /tmp, so your real
# config and keyring entries are never read or written.
# The JSON-RPC frames typed on screen are what an MCP client sends - a real
# handshake against the real server, not a mock.
# VHS: https://github.com/charmbracelet/vhs

Output assets/mcp.gif

Require bash
Require go
Require jq

Set Shell "bash"
Set FontSize 15
Set Width 1500
Set Height 700
Set Padding 16
Set Framerate 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

# --- Hidden setup: rebuild, isolate config, define the JSON-RPC helpers ---
Hide
Type `export PS1="> " COOLDECK_CONFIG_DIR=/tmp/cooldeck-mcp-config`
Enter
Type `rm -rf /tmp/cooldeck-mcp-config && mkdir -p /tmp/cooldeck-mcp-config`
Enter
Type `go build -o bin/cooldeck ./cmd/cooldeck || { echo BUILD FAILED; sleep 5; }`
Enter
Sleep 20s
Type `export PATH=$PWD/bin:$PATH`
Enter
# hello: the MCP handshake every client performs before anything else.
Type `hello='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"agent","version":"1"}}}\n{"jsonrpc":"2.0","method":"notifications/initialized"}'`
Enter
# mcp: pipe the handshake plus one request into the server, keep stdin open
# long enough for the reply, and hand back the last frame.
Type `mcp() { { printf "%b\n" "$hello" "$1"; sleep 1; } | cooldeck mcp $2 --demo 2>/dev/null | tail -1; }`
Enter
Type `call() { mcp "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"$1\",\"arguments\":$2}}"; }`
Enter
Type "clear"
Enter
Sleep 500ms
Show

# --- The contract: read-only unless you say otherwise ---
Sleep 800ms
Type "cooldeck mcp --help | head -12"
Sleep 300ms
Enter
Sleep 4000ms
Type "clear"
Enter

# --- What the agent discovers: a real handshake, a real tools/list ---
Type `mcp '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | jq -r '.result.tools[].name'`
Sleep 400ms
Enter
Sleep 3800ms

# --- The fleet, as the agent sees it ---
Type `call list_applications '{}' | jq -r '.result.structuredContent.Applications[]`
Type `  | "\(.Status.State)\t\(.Name)"' | head -6`
Sleep 400ms
Enter
Sleep 4000ms

# --- The payoff: it reads the logs of the degraded one and finds the cause ---
Type `call get_runtime_logs '{"application_uuid":"demo0044pjawa","lines":6}'`
Type `  | jq -r '.result.structuredContent.Lines[].Text'`
Sleep 400ms
Enter
Sleep 4500ms
Type "clear"
Enter

# --- Deploy, restart, start and stop appear only when you opt in ---
Type `mcp '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' --allow-mutations`
Type ` | jq -r '.result.tools[].name' | grep -Ev '^(get|list)_'`
Sleep 400ms
Enter
Sleep 4000ms

# --- Hidden teardown ---
Hide
Type `rm -rf /tmp/cooldeck-mcp-config`
Enter
Sleep 500ms
