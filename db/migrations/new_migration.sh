#!/usr/bin/env bash
set -e

if [[ -z $NAME ]]; then
    echo "NAME must be set, run like \`make migration NAME='new name here'\`"
    exit 1
fi

DATE=$(date +"%Y%m%d%H%M00")
SANITIZED_NAME=$(echo $NAME | sed 's/ /_/g')
FILENAME="db/migrations/${DATE}_${SANITIZED_NAME}.go"

cp db/migrations/x_migration_template.go.template $FILENAME
sed -i"" "s/DATE/${DATE}/g" $FILENAME
sed -i"" "s/NAME/${NAME}/g" $FILENAME
