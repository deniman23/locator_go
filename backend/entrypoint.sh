#!/usr/bin/env bash
set -e

until psql "$DATABASE_URL" -c '\q'; do
  echo "⏳ waiting for postgres…"
  sleep 2
done

goose -dir /migrations postgres "$DATABASE_URL" up
exec su-exec postgres /app/locator
