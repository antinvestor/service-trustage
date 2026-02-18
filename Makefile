.PHONY: tests format vet docker-setup docker-stop lint build

tests:
	go test ./... -v -race -coverprofile=coverage.out

format:
	find . -name '*.go' -exec goimports -w {} +
	golangci-lint run --fix

vet:
	go vet ./...

lint:
	golangci-lint run --timeout=5m

docker-setup:
	docker compose up -d --remove-orphans --force-recreate

docker-stop:
	docker compose down

build:
	CGO_ENABLED=0 go build -trimpath -o bin/trustage ./apps/default/cmd/main.go
