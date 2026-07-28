.PHONY: dev build test up down logs

dev:
	docker compose up --build

build:
	cd frontend && npm run build
	cd backend && go build ./...

test:
	cd frontend && npm run lint
	cd backend && go test ./...

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

