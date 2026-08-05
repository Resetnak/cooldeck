# social.tape - renders assets/social-preview.png, the GitHub social preview.
# Regenerate from the repo root:  vhs social.tape
# 1280x640 is GitHub's expected size; the frame shows the fleet tail, which is
# the one screen no Coolify web UI can produce. Demo data only.
# VHS: https://github.com/charmbracelet/vhs

Output "/tmp/cooldeck-social.gif"

Require bash
Require go

Set Shell "bash"
Set FontSize 20
Set Width 1280
Set Height 640
Set Padding 20
Set Theme "Catppuccin Mocha"

Hide
Type `export PS1="> " COOLDECK_CONFIG_DIR=/tmp/cooldeck-social-config`
Enter
Type `rm -rf /tmp/cooldeck-social-config && mkdir -p /tmp/cooldeck-social-config`
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
Type "cooldeck --demo --theme catppuccin"
Enter
Sleep 3s

# Mark three services and tail them, then hold the frame.
Type " "
Sleep 300ms
Type "j"
Type " "
Sleep 300ms
Type "j"
Type "j"
Type " "
Sleep 300ms
Type "t"
Sleep 8s
Screenshot "assets/social-preview.png"

Hide
Type "q"
Sleep 500ms
Type `rm -rf /tmp/cooldeck-social-config`
Enter
Sleep 500ms
