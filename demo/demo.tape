# VHS tape for bolna-cli's README demo GIF.
# Install VHS (https://github.com/charmbracelet/vhs), build `bolna` onto your
# PATH, log in with a real account, then run:
#
#   vhs demo/demo.tape
#
# Output lands at demo/demo.gif — this file is a script, never run at build
# time, and is not part of the shipped binary.
#
# The CLI-lifecycle portion (create/view/update/delete) is deliberately
# scoped to demo/demo-agent.json — a disposable, clearly-fake agent it
# creates and deletes — rather than `bolna agents list`, which shows every
# real agent on the account. This keeps the recording safe to publish
# regardless of whose account it's recorded against.
#
# The splash screen (bare `bolna`) opens the recording. It auto-transitions
# into the real dashboard (real agent names, real wallet balance in the
# header) once its spring animation settles, so we quit with `q` while it's
# still showing rather than letting that happen — `q` during the splash is
# specifically wired to quit immediately instead of just dismissing the
# splash (see dashboard.go's handleKey), so this never renders the real
# dashboard even for one frame.

Output demo/demo.gif

Set FontSize 16
Set Width 1200
Set Height 720
Set Theme "nord"
Set Shell zsh
Set TypingSpeed 40ms

Type "bolna"
Enter
Sleep 1.9s
Type "q"
Sleep 500ms

Type "clear"
Enter
Sleep 500ms

Type "bolna doctor"
Enter
Sleep 3s

Type "bash demo/record.sh"
Enter
Sleep 15s
