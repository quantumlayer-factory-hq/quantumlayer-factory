#!/bin/bash
set -e

# Create factory database for our application (temporal user already exists)
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE factory OWNER temporal;
    GRANT ALL PRIVILEGES ON DATABASE factory TO temporal;
EOSQL

echo "PostgreSQL databases initialized successfully"