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
DART_RUNTIME_WKTS := any api duration empty field_mask source_context struct timestamp type wrappers

.PHONY: proto-generate-dart
proto-generate-dart: $(BIN)/buf ## Regenerate the per-module dart SDKs
	@if [ ! -d "$(PROTO_DIR)" ]; then exit 0; fi
	@# lib/src is entirely generated: wipe it so outputs the plugins no longer
	@# emit (e.g. empty .pbserver.dart files) do not linger.
	@for m in $(DART_MODULES); do rm -rf sdk/dart/$$m/lib/src; done
	@for m in $(DART_MODULES); do \
		echo "==> dart $$m"; \
		(cd $(PROTO_DIR) && buf generate --template buf.gen.dart.$$m.yaml $$m); \
	done
	@# protoc_plugin 25 imports the well-known types from package:protobuf, so
	@# the copies include_wkt emits are unreferenced. descriptor.proto is not
	@# shipped by the runtime and stays (buf/validate and gnostic import it).
	@for m in $(DART_MODULES); do \
		for wkt in $(DART_RUNTIME_WKTS); do \
			rm -f sdk/dart/$$m/lib/src/google/protobuf/$$wkt.pb*.dart; \
		done; \
	done
	@# connect-dart output is not dart-formatted; keep `dart format` checks green.
	@if command -v dart >/dev/null 2>&1; then \
		for m in $(DART_MODULES); do dart format sdk/dart/$$m/lib >/dev/null; done; \
	fi

proto-generate: proto-generate-dart
