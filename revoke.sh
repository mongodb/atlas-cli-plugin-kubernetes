#!/bin/bash

GH_APP_ID=885509
GH_APP_KEY=~/app.pem

for PR_URL in "$@"
do
    GH_OWNER=$(echo "$PR_URL" | cut -d "/" -f 4)
    GH_INSTALLATION_ID=$(gh token installations --app-id "$GH_APP_ID" --key "$GH_APP_KEY" 2>/dev/null  | jq --arg owner "$GH_OWNER" '.[] | select(.account.login == $owner) | .id')
    GH_TOKEN=$(gh token generate --app-id "$GH_APP_ID" --key "$GH_APP_KEY" --installation-id "$GH_INSTALLATION_ID" -d 10 -t)
    GH_TOKEN=$GH_TOKEN gh pr close "$PR_URL" -d -c "testing"
    gh token revoke --token "$GH_TOKEN"
done
