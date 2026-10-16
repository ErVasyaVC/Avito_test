#!/bin/sh
# Полностью очищает все таблицы kitchen-db/venue-db (TRUNCATE ... CASCADE).
# Схема и миграции не трогаются — только данные. Требует уже поднятый docker compose.
set -e

[ -f .env ] && . ./.env

docker compose exec -T kitchen-db psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -c "TRUNCATE order_status_history, order_items, orders, menu_items, venue_categories, venues, categories CASCADE;"

docker compose exec -T venue-db psql -v ON_ERROR_STOP=1 -U "$VENUE_POSTGRES_USER" -d "$VENUE_POSTGRES_DB" \
    -c "TRUNCATE order_items, orders, menu_items CASCADE;"

echo "Данные очищены."
