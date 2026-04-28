# Service-specific configuration
SERVICE_NAME := trustage
APP_DIRS     := apps/default apps/formstore apps/queue

# Bootstrap: download shared Makefile.common if missing
ifeq (,$(wildcard .tmp/Makefile.common))
  $(shell mkdir -p .tmp && curl -sSfL https://raw.githubusercontent.com/antinvestor/common/main/Makefile.common -o .tmp/Makefile.common)
endif

include .tmp/Makefile.common

format: ## Format Go files (used by pre-commit hook)
	gofmt -w .

# Dart proto modules — each gets its own buf.gen.dart.<module>.yaml.
DART_MODULES := event runtime signal workflow

.PHONY: proto-generate-dart
proto-generate-dart: $(BIN)/buf ## Regenerate the per-module dart SDKs
	@if [ ! -d "$(PROTO_DIR)" ]; then exit 0; fi
	@for m in $(DART_MODULES); do rm -rf sdk/dart/$$m/lib/src/v1; done
	@for m in $(DART_MODULES); do \
		echo "==> dart $$m"; \
		(cd $(PROTO_DIR) && buf generate --template buf.gen.dart.$$m.yaml $$m); \
	done

proto-generate: proto-generate-dart
