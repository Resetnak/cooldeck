# cassette.tape - VHS script that renders the README demo (assets/demo.gif).
# Regenerate from the repo root:  vhs cassette.tape
# Needs: vhs, bash and go - the hidden setup below rebuilds bin/cooldeck, so the
# recording always shows the working tree rather than a stale binary.
# Everything runs against `--demo`: no Coolify instance, no network, no token.
# COOLDECK_CONFIG_DIR points at a throwaway directory under /tmp, so your real
# config and keyring entries are never read or written.
# The width is deliberate: 150+ columns is where the side preview panel earns
# its space (theme.PreviewMinWidth), which is the layout worth showing off.
# VHS: https://github.com/charmbracelet/vhs

Output assets/demo.gif

Require bash
Require go

Set Shell "bash"
Set FontSize 15
Set Width 1600
Set Height 640
Set Padding 16
Set Framerate 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

# --- Hidden setup: rebuild, isolate config, clear the screen ---
Hide
Type `export PS1="> " COOLDECK_CONFIG_DIR=/tmp/cooldeck-demo-config`
Enter
Type `rm -rf /tmp/cooldeck-demo-config && mkdir -p /tmp/cooldeck-demo-config`
Enter
Type `go build -o bin/cooldeck ./cmd/cooldeck || { echo BUILD FAILED; sleep 5; }`
Enter
Sleep 20s
Type `export PATH=$PWD/bin:$PATH`
Enter
Type "clear"
Enter
Sleep 500ms
Show

# --- Launch: the whole fleet on one screen ---
Sleep 800ms
Type "cooldeck --demo --theme catppuccin"
Sleep 400ms
Enter
Sleep 3500ms

# --- The side panel follows the selection; g jumps back to the degraded app ---
Type "j"
Sleep 900ms
Type "j"
Sleep 1100ms
Type "g"
Sleep 1400ms

# --- / narrows 12 apps down to one project; enter applies the filter ---
Type "/"
Sleep 500ms
Type "billing"
Sleep 1800ms
Enter
Sleep 1400ms

# --- enter opens the detail; l streams the runtime logs ---
Enter
Sleep 2400ms
Type "l"
Sleep 2800ms

# --- search the buffer for the thing that actually broke ---
Type "/"
Sleep 400ms
Type "upstream"
Sleep 2400ms
Escape
Sleep 600ms
Escape
Sleep 1000ms

# --- back to the list; d redeploys, but never without confirming ---
Escape
Sleep 1400ms
Type "d"
Sleep 2200ms
Enter
Sleep 2600ms

# --- 2 then a: only what is in flight, including what we just triggered ---
Type "2"
Sleep 2000ms
Type "a"
Sleep 3000ms

# --- ? : every binding, always one keystroke away ---
Type "?"
Sleep 2800ms
Escape
Sleep 800ms

# --- Hidden teardown ---
Hide
Type "q"
Sleep 1000ms
Type `rm -rf /tmp/cooldeck-demo-config`
Enter
Sleep 500ms
