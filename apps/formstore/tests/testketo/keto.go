package testketo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// ImageName is the Ory Keto image used for test containers.
	ImageName = "oryd/keto:latest"

	ketoConfiguration = `
version: v0.14.0

dsn: memory

serve:
  read:
    host: 0.0.0.0
    port: 4466
  write:
    host: 0.0.0.0
    port: 4467

log:
  level: debug
  format: text

namespaces:
  location: file:///home/ory/namespaces

`

	oplNamespaces = `// Keto Namespace Configuration for Trustage
// Using Ory Permission Language (OPL) - TypeScript-like DSL

import { Namespace, Context } from "@ory/keto-namespace-types"

// trustage_profile namespace represents users/actors
class profile implements Namespace {
  related: {
    self: profile[]
  }
}

// trustage_tenant namespace represents a tenant boundary
class trustage_tenant implements Namespace {
  related: {
    owner: profile[]
    admin: profile[]
    member: profile[]
  }

  permits = {
    ingest_event: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    manage_workflow: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject),

    view_workflow: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    view_instance: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    retry_instance: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject),

    view_execution: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject) ||
      this.related.member.includes(ctx.subject),

    retry_execution: (ctx: Context): boolean =>
      this.related.owner.includes(ctx.subject) ||
      this.related.admin.includes(ctx.subject),
  }
}

export { profile, trustage_tenant }
`

	namespaceFile = "/home/ory/namespaces/trustage.ts"
)

type dependancy struct {
	*definition.DefaultImpl
}

// NewWithOpts creates a new Keto test resource with OPL namespace support.
func NewWithOpts(
	containerOpts ...definition.ContainerOption,
) definition.TestResource {
	opts := definition.ContainerOpts{
		ImageName:      ImageName,
		Ports:          []string{"4467/tcp", "4466/tcp"},
		NetworkAliases: []string{"keto", "auth-keto"},
	}
	opts.Setup(containerOpts...)

	return &dependancy{
		DefaultImpl: definition.NewDefaultImpl(opts, "http"),
	}
}

func (d *dependancy) migrateContainer(
	ctx context.Context,
	ntwk *testcontainers.DockerNetwork,
	databaseURL string,
) error {
	containerRequest := testcontainers.ContainerRequest{
		Image: d.Name(),
		Cmd:   []string{"migrate", "up", "--yes"},
		Env: map[string]string{
			"LOG_LEVEL": "debug",
			"DSN":       databaseURL,
		},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            strings.NewReader(ketoConfiguration),
				ContainerFilePath: "/home/ory/keto.yml",
				FileMode:          definition.ContainerFileMode,
			},
			{
				Reader:            strings.NewReader(oplNamespaces),
				ContainerFilePath: namespaceFile,
				FileMode:          definition.ContainerFileMode,
			},
		},
		WaitingFor: wait.ForExit(),
	}

	d.Configure(ctx, ntwk, &containerRequest)

	ketoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: containerRequest,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start keto migration container: %w", err)
	}

	if err = ketoContainer.Terminate(ctx); err != nil {
		return fmt.Errorf("failed to terminate keto migration container: %w", err)
	}
	return nil
}

func (d *dependancy) Setup(ctx context.Context, ntwk *testcontainers.DockerNetwork) error {
	if len(d.Opts().Dependencies) == 0 || !d.Opts().Dependencies[0].GetDS(ctx).IsDB() {
		return errors.New("no database dependency was supplied")
	}

	ketoDB, _, err := testpostgres.CreateDatabase(ctx, d.Opts().Dependencies[0].GetInternalDS(ctx), "keto")
	if err != nil {
		return fmt.Errorf("failed to create keto database: %w", err)
	}

	databaseURL := ketoDB.String()

	if err = d.migrateContainer(ctx, ntwk, databaseURL); err != nil {
		return err
	}

	containerRequest := testcontainers.ContainerRequest{
		Image: d.Name(),
		Cmd:   []string{"serve", "--config", "/home/ory/keto.yml"},
		Env: d.Opts().Env(map[string]string{
			"LOG_LEVEL":                 "debug",
			"LOG_LEAK_SENSITIVE_VALUES": "true",
			"DSN":                       databaseURL,
		}),
		Files: []testcontainers.ContainerFile{
			{
				Reader:            strings.NewReader(ketoConfiguration),
				ContainerFilePath: "/home/ory/keto.yml",
				FileMode:          definition.ContainerFileMode,
			},
			{
				Reader:            strings.NewReader(oplNamespaces),
				ContainerFilePath: namespaceFile,
				FileMode:          definition.ContainerFileMode,
			},
		},
		WaitingFor: wait.ForHTTP("/health/ready").WithPort(d.DefaultPort),
	}

	d.Configure(ctx, ntwk, &containerRequest)

	ketoContainer, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: containerRequest,
			Started:          true,
		})
	if err != nil {
		return fmt.Errorf("failed to start keto serve container: %w", err)
	}

	d.SetContainer(ketoContainer)
	return nil
}
