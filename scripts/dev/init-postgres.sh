#!/bin/bash
set -e

# Create temporal database if it doesn't exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE temporal;
    GRANT ALL PRIVILEGES ON DATABASE temporal TO $POSTGRES_USER;
EOSQL

echo "PostgreSQL databases initialized successfully"