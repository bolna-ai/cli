# VHS tape for bolna-cli's README demo GIF.
# Install VHS (https://github.com/charmbracelet/vhs), build `bolna` onto your
# PATH, log in with a real account, then run:
#
#   vhs demo/demo.tape
#
# Output lands at demo/demo.gif — this file is a script, never run at build
# time, and is not part of the shipped binary.
#
# Deliberately scoped to demo/demo-agent.json (a disposable, clearly-fake
# agent it creates and deletes) rather than `bolna agents list` or the bare
# `bolna` dashboard — both of those show every real agent on the account and
# the header always shows the real wallet balance. This keeps the recording
# safe to publish regardless of whose account it's recorded against.

Output demo/demo.gif

Set FontSize 16
Set Width 1200
Set Height 720
Set Theme "nord"
Set Shell zsh
Set TypingSpeed 40ms

Type "bolna doctor"
Enter
Sleep 3s

Type "bash demo/record.sh"
Enter
Sleep 15s
