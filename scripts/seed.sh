#!/bin/sh
# Наполняет БД тестовыми данными для ручной проверки/демонстрации.
# Требует уже поднятый docker compose (kitchen-db, venue-db).
set -e

[ -f .env ] && . ./.env

docker compose exec -T kitchen-db psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    < scripts/seed/kitchen.sql

docker compose exec -T venue-db psql -v ON_ERROR_STOP=1 -U "$VENUE_POSTGRES_USER" -d "$VENUE_POSTGRES_DB" \
    < scripts/seed/venue.sql

echo "Тестовые данные загружены."
