# VHS script: 10-second walkthrough of `pickmem pick`.
# Run with: vhs demo/pick.tape → demo/pick.gif
#
# Prereqs (VHS handles most, but pickmem must be on PATH):
#   go install ./cmd/pickmem  (or ln -s the built binary)
#   brew install charmbracelet/tap/vhs
#
# The tape seeds a fresh vault so the recording is deterministic.

Output demo/pick.gif

Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 520
Set Theme "Nord"
Set PlaybackSpeed 1.0

# Set up a demo vault in a temp dir so the GIF doesn't depend on the
# viewer's own vault.
Hide
Type "export VAULT=$(mktemp -d) && export XDG_CONFIG_HOME=$(mktemp -d)"
Enter
Type "pickmem init $VAULT --template personal >/dev/null"
Enter
Type `pickmem add --label "salary" --group financial --tags money,recurring --body "monthly base \$8k" >/dev/null`
Enter
Type `pickmem add --label "rent + utilities" --group financial --body "\$3200 rent, \$180 utilities" >/dev/null`
Enter
Type `pickmem add --label "kickoff notes" --group "work/Client-Acme" --body "kickoff Aug 1" >/dev/null`
Enter
Type `pickmem add --label "gift ideas for sister" --group relationships --tags gifts --body "plants, enamel pins" >/dev/null`
Enter
Type `pickmem add --label "solar research" --group home --body "3 quotes, compare inverters" >/dev/null`
Enter
Type "clear"
Enter
Show

# The walkthrough proper.
Type "pickmem pick"
Enter
Sleep 1s

# Move down, toggle a couple of items.
Type "j"
Sleep 300ms
Type " "
Sleep 400ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type " "
Sleep 500ms

# Filter for the sister gift note.
Type "/"
Sleep 300ms
Type "gift"
Sleep 700ms
Escape
Sleep 400ms

# Toggle it.
Type " "
Sleep 500ms

# Save as a lens.
Type "s"
Sleep 400ms
Type "Weekend"
Sleep 400ms
Enter
Sleep 700ms

# Confirm.
Enter
Sleep 1s
