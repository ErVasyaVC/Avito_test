.PHONY: help up up-d down logs \
	lint lint-kitchen lint-venue \
	test test-kitchen test-venue testdb-up testdb-down \
	seed reset-db migrate-kitchen migrate-venue

GOLANGCI := docker run --rm -v "$(CURDIR):/repo" golangci/golangci-lint:latest golangci-lint run ./...
GO := docker run --rm --network host -v "$(CURDIR):/repo"

help:
	@echo "make up            - поднять весь стек (foreground, видно логи)"
	@echo "make up-d          - поднять весь стек в фоне"
	@echo "make down          - остановить и убрать контейнеры (данные сохраняются)"
	@echo "make logs          - логи всех сервисов"
	@echo "make lint          - golangci-lint по обоим сервисам"
	@echo "make test          - тесты по обоим сервисам"
	@echo "make seed          - наполнить dev-БД тестовыми данными"
	@echo "make reset-db      - очистить все таблицы dev-БД"

up:
	docker compose up --build

up-d:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

## --- линтеры ---

lint: lint-kitchen lint-venue

lint-kitchen:
	docker run --rm -v "$(CURDIR):/repo" -w /repo/services/kitchen-api golangci/golangci-lint:latest golangci-lint run ./...

lint-venue:
	docker run --rm -v "$(CURDIR):/repo" -w /repo/services/venue-service golangci/golangci-lint:latest golangci-lint run ./...

## --- тестовые БД ---

testdb-up:
	docker compose --profile test up -d kitchen-test-db venue-test-db
	@echo "ждём готовности..."
	@until docker compose exec -T kitchen-test-db pg_isready -U kitchen -d kitchen_test >/dev/null 2>&1; do sleep 1; done
	@until docker compose exec -T venue-test-db pg_isready -U venue -d venue_test >/dev/null 2>&1; do sleep 1; done
	docker run --rm --network host -v "$(CURDIR)/services/kitchen-api/migrations:/migrations" migrate/migrate:v4.17.1 \
		-path=/migrations -database "postgres://kitchen:kitchen@localhost:5442/kitchen_test?sslmode=disable" up
	docker run --rm --network host -v "$(CURDIR)/services/venue-service/migrations:/migrations" migrate/migrate:v4.17.1 \
		-path=/migrations -database "postgres://venue:venue@localhost:5443/venue_test?sslmode=disable" up

testdb-down:
	docker compose rm -sf kitchen-test-db venue-test-db

## --- тесты ---

test: testdb-up test-kitchen test-venue testdb-down

test-kitchen:
	docker run --rm --network host \
		-e TEST_DATABASE_URL="postgres://kitchen:kitchen@localhost:5442/kitchen_test?sslmode=disable" \
		-v "$(CURDIR)/services/kitchen-api:/app" -w /app golang:1.23-alpine \
		go test ./...

test-venue:
	docker run --rm --network host \
		-e TEST_DATABASE_URL="postgres://venue:venue@localhost:5443/venue_test?sslmode=disable" \
		-v "$(CURDIR)/services/venue-service:/app" -w /app golang:1.23-alpine \
		go test ./...

## --- данные dev-БД ---

seed:
	./scripts/seed.sh

reset-db:
	./scripts/reset-db.sh
