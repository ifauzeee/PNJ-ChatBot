#!/bin/bash

# PNJ Anonymous Bot - Database Backup Script
# This script creates a timestamped backup of the database.

set -e

BACKUP_DIR="/app/backups"
mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
echo "🚀 Starting backup at $TIMESTAMP..."

if [ "$DB_TYPE" = "postgres" ]; then
    FILENAME="pnj_backup_pg_$TIMESTAMP.sql.gz"
    echo "📦 Backing up PostgreSQL database: $DB_NAME..."
    # Map DB_PASSWORD to PGPASSWORD for pg_dump
    export PGPASSWORD="$DB_PASSWORD"
    pg_dump -h "$DB_HOST" -U "$DB_USER" "$DB_NAME" | gzip > "$BACKUP_DIR/$FILENAME"
else
    FILENAME="pnj_backup_sqlite_$TIMESTAMP.db"
    echo "📦 Backing up SQLite database: $DB_PATH..."
    if [ -f "$DB_PATH" ]; then
        cp "$DB_PATH" "$BACKUP_DIR/$FILENAME"
    else
        echo "⚠️  SQLite database file not found at $DB_PATH"
        exit 1
    fi
fi

# Keep only the last 30 backups
echo "🧹 Cleaning up old backups (keeping last 30)..."
ls -tp "$BACKUP_DIR"/pnj_backup_* | grep -v '/$' | tail -n +31 | xargs -I {} rm -- {} || true

echo "✅ Backup completed: $FILENAME"
