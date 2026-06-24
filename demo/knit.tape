# knit demo — render with: vhs demo/knit.tape
# All commands are offline & deterministic (no token, no network): they exercise the
# read-only gate, the dry-run plan+hash, and the machine-readable contract.
Output demo/knit.gif

Require knit
Require jq

Set Shell "bash"
Set FontSize 21
Set Width 1240
Set Height 760
Set Padding 28
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set MarginFill "#1b1b2a"
Set Margin 18
Set BorderRadius 12

Hide
Type "export NO_COLOR= PS1='\033[38;5;212m❯\033[0m '" Enter
Type "clear" Enter
Show

Sleep 600ms
Type "# Threads is read-only by default — publishing is blocked in the binary" Sleep 400ms Enter
Type "knit post create --text 'gm threads' --json" Sleep 300ms Enter
Sleep 2s

Type "# Preview the exact plan + a hash — nothing is published" Sleep 400ms Enter
Type "knit post create --text 'gm threads' --allow-mutations --dry-run --json" Sleep 300ms Enter
Sleep 2.5s

Type "# The whole contract is machine-readable" Sleep 400ms Enter
Type "knit schema | jq '{exit_codes, safety}'" Sleep 300ms Enter
Sleep 2.5s

Type "# Read-only by default. Gated in the binary. Built for agents." Sleep 500ms Enter
Sleep 2s
