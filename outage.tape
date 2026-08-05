# outage.tape - VHS script that renders assets/outage.gif, the clip about what
# CoolDeck does when Coolify stops answering.
# Regenerate from the repo root:  vhs outage.tape
#
# Same throwaway setup as cassette.tape: `--demo` data, an isolated
# COOLDECK_CONFIG_DIR under /tmp, no network and no token. The outage itself is
# simulated by the demo service's F2 switch, so the clip is reproducible - the
# point being that a failed refresh keeps the last good snapshot on screen and
# adds a stale banner instead of wiping the list.
# VHS: https://github.com/charmbracelet/vhs

Output assets/outage.gif

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

# --- Hidden setup ---
Hide
Type `export PS1="> " COOLDECK_CONFIG_DIR=/tmp/cooldeck-outage-config`
Enter
Type `rm -rf /tmp/cooldeck-outage-config && mkdir -p /tmp/cooldeck-outage-config`
Enter
Type `go build -o bin/cooldeck ./cmd/cooldeck || { echo BUILD FAILED; sleep 5; }`
Enter
Sleep 20s
Type `export PATH=$PWD/bin:$PATH && clear`
Enter
Sleep 500ms
Type "cooldeck --demo --theme catppuccin"
Enter
Sleep 4s
Show

# --- ':' - every action also has a name, for the days you forget the key ---
Sleep 800ms
Type ":"
Sleep 700ms
Type "log"
Sleep 2200ms
Escape
Sleep 1200ms

# --- F2 simulates the outage, R forces a refresh into it. VHS 0.11 has no F2
# --- keyword, so the xterm sequence for it is typed by hand, with no delay
# --- between the bytes or the terminal would flush a bare Escape instead.
Escape@0ms
Type@0ms "[12~"
Sleep 800ms
Type "R"
Sleep 3600ms

# --- The list is still there. Stale, banner-flagged, and honest about it ---
Type "j"
Sleep 1000ms
Type "j"
Sleep 1800ms

# --- F2 again and the next refresh heals it; no restart, no lost place ---
Escape@0ms
Type@0ms "[12~"
Sleep 600ms
Type "R"
Sleep 3600ms

# --- Hidden teardown ---
Hide
Type "q"
Sleep 1000ms
Type `rm -rf /tmp/cooldeck-outage-config`
Enter
Sleep 500ms
