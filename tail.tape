# tail.tape - VHS script that renders the fleet-tail demo (assets/tail.gif).
# Regenerate from the repo root:  vhs tail.tape
# Same isolation as cassette.tape: --demo only, throwaway COOLDECK_CONFIG_DIR,
# no Coolify instance and no token anywhere in the recording.
# This one exists to show the single thing the web UI cannot do: several
# applications' runtime logs interleaved in one buffer.
# VHS: https://github.com/charmbracelet/vhs

Output assets/tail.gif

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
Type `export PS1="> " COOLDECK_CONFIG_DIR=/tmp/cooldeck-tail-config`
Enter
Type `rm -rf /tmp/cooldeck-tail-config && mkdir -p /tmp/cooldeck-tail-config`
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

# --- Launch ---
Sleep 800ms
Type "cooldeck --demo --theme catppuccin"
Sleep 400ms
Enter
Sleep 3000ms

# --- space marks an application, j moves on: three services, three marks ---
Type " "
Sleep 900ms
Type "j"
Sleep 500ms
Type " "
Sleep 900ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type " "
Sleep 1400ms

# --- t: their runtime logs, interleaved, each line named by its application ---
Type "t"
Sleep 6000ms

# --- search the combined buffer, not three separate ones ---
Type "/"
Sleep 400ms
Type "error"
Sleep 2600ms
Escape
Sleep 1800ms

# --- w wraps long lines, escape drops back to the fleet ---
Type "w"
Sleep 2000ms
Escape
Sleep 1600ms

# --- Hidden teardown ---
Hide
Type "q"
Sleep 1000ms
Type `rm -rf /tmp/cooldeck-tail-config`
Enter
Sleep 500ms
