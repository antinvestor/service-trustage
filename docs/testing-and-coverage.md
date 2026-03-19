# Testing And Coverage

Trustage has two different coverage views, and they answer different questions.

## Raw Whole-Repo Coverage

Use this when you want the literal Go toolchain view of the entire repository:

```bash
make coverage-raw
```

This includes everything under `./...`, including:

- generated code under `gen/`
- proto source packages under `proto/`
- `cmd` and `config` packages
- packages whose behavior is exercised indirectly from external integration test packages

That number is useful for reference, but it is not a good quality gate for Trustage.

## Handwritten App Coverage

Use this when you want the quality signal for the actual handwritten application code:

```bash
make coverage-app
```

This target:

- runs the integration-heavy test suites in `apps/default/tests`, `apps/formstore/tests`, and `apps/queue/tests`
- includes focused package-local unit tests for authz, business, handlers, connectors, DSL, and utility packages
- uses `-coverpkg` so coverage from external integration packages is attributed back to the handwritten packages they exercise
- excludes generated/proto artifacts and similar non-gate packages from the coverage set

The profile is written to `coverage_handwritten.out`.

## Coverage Gate

To enforce a threshold on handwritten application coverage:

```bash
make coverage-check
```

The default threshold is `85%`.

To override it:

```bash
COVERAGE_MIN=80 make coverage-check
```

## Why The Numbers Differ

`go test ./... -coverprofile=coverage.out` can understate meaningful Trustage coverage because many of the important end-to-end tests live in separate integration packages. Without `-coverpkg`, those tests do not fully credit the business, repository, scheduler, and handler packages they exercise.

That is why Trustage keeps both:

- `coverage-raw` for literal whole-repo reporting
- `coverage-app` and `coverage-check` for the actual engineering quality gate
