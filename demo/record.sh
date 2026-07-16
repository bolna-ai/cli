#!/bin/bash
# Helper for demo/demo.tape: runs the create/view/update/delete lifecycle
# against the disposable demo/demo-agent.json agent, with real pauses for
# each API call rather than relying on VHS's typing-speed timing for long
# one-liners (which was fragile — see git history for why this replaced
# inline `Type` commands).
set -e

echo '$ bolna agents create --file demo/demo-agent.json'
OUT=$(bolna agents create --file demo/demo-agent.json)
echo "$OUT"
ID=$(echo "$OUT" | grep -oE "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")
sleep 1.5

echo
echo "\$ bolna agents view $ID"
bolna agents view "$ID"
sleep 1.5

echo
echo "\$ bolna agents update $ID --name \"Aria (Updated)\" --welcome \"...\" --yes"
bolna agents update "$ID" --name "Aria (Updated)" --welcome "Thanks for calling Acme Corp, this is Aria!" --yes
sleep 1.5

echo
echo "\$ bolna agents delete $ID --yes"
bolna agents delete "$ID" --yes
