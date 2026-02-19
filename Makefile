.PHONY: tests format vet docker-setup docker-stop lint build baml-gen

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
	go build -trimpath -o bin/trustage ./apps/default/cmd/main.go

baml-gen:
	baml-cli generate
