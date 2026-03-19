.PHONY: tests coverage-raw coverage-app coverage-check format vet docker-setup docker-stop lint build baml-gen

tests:
	go test ./... -v -race

coverage-raw:
	./scripts/coverage.sh raw

coverage-app:
	./scripts/coverage.sh app

coverage-check:
	./scripts/coverage.sh check

format:
	find . -name '*.go' -not -path './.git/*' -exec sed -i '/^import (/,/^)/{/^$$/d}' {} +
	find . -name '*.go' -not -path './.git/*' -exec goimports -w {} +
	golangci-lint run --fix -c .golangci.yaml

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
