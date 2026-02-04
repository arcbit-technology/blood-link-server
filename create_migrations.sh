#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <migration_name>"
    exit 1
fi

timestamp=$(date +"%Y%m%d%H%M%S")
migration_name=$1

cd dao/migrations
touch "${timestamp}_${migration_name}.up.sql"

touch "${timestamp}_${migration_name}.down.sql"

echo "Created migration files:"
echo "${timestamp}_${migration_name}.up.sql"
echo "${timestamp}_${migration_name}.down.sql"
