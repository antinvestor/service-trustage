// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
limit:
  max_read_depth: 10

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
  location: file:///home/ory/namespaces/trustage.ts

`

	oplNamespaces = `import { Namespace, Context } from "@ory/keto-namespace-types"

class profile_user implements Namespace {}

class tenancy_access implements Namespace {
  related: {
    member: profile_user[]
    service: profile_user[]
  }
}

class service_trustage implements Namespace {
  related: {
    owner: profile_user[]
    admin: profile_user[]
    member: profile_user[]
    service: (profile_user | tenancy_access)[]
    granted_event_ingest: (profile_user | service_trustage)[]
    granted_workflow_manage: (profile_user | service_trustage)[]
    granted_workflow_view: (profile_user | service_trustage)[]
    granted_instance_view: (profile_user | service_trustage)[]
    granted_instance_retry: (profile_user | service_trustage)[]
    granted_execution_view: (profile_user | service_trustage)[]
    granted_execution_retry: (profile_user | service_trustage)[]
    granted_form_definition_manage: (profile_user | service_trustage)[]
    granted_form_definition_view: (profile_user | service_trustage)[]
    granted_form_submit: (profile_user | service_trustage)[]
    granted_submission_view: (profile_user | service_trustage)[]
    granted_submission_update: (profile_user | service_trustage)[]
    granted_submission_delete: (profile_user | service_trustage)[]
    granted_queue_manage: (profile_user | service_trustage)[]
    granted_queue_view: (profile_user | service_trustage)[]
    granted_item_enqueue: (profile_user | service_trustage)[]
    granted_queue_item_view: (profile_user | service_trustage)[]
    granted_counter_manage: (profile_user | service_trustage)[]
    granted_stats_view: (profile_user | service_trustage)[]
  }

  permits = {
    event_ingest: (ctx: Context): boolean =>
      this.related.granted_event_ingest.includes(ctx.subject),
    workflow_manage: (ctx: Context): boolean =>
      this.related.granted_workflow_manage.includes(ctx.subject),
    workflow_view: (ctx: Context): boolean =>
      this.related.granted_workflow_view.includes(ctx.subject),
    instance_view: (ctx: Context): boolean =>
      this.related.granted_instance_view.includes(ctx.subject),
    instance_retry: (ctx: Context): boolean =>
      this.related.granted_instance_retry.includes(ctx.subject),
    execution_view: (ctx: Context): boolean =>
      this.related.granted_execution_view.includes(ctx.subject),
    execution_retry: (ctx: Context): boolean =>
      this.related.granted_execution_retry.includes(ctx.subject),
    form_definition_manage: (ctx: Context): boolean =>
      this.related.granted_form_definition_manage.includes(ctx.subject),
    form_definition_view: (ctx: Context): boolean =>
      this.related.granted_form_definition_view.includes(ctx.subject),
    form_submit: (ctx: Context): boolean =>
      this.related.granted_form_submit.includes(ctx.subject),
    submission_view: (ctx: Context): boolean =>
      this.related.granted_submission_view.includes(ctx.subject),
    submission_update: (ctx: Context): boolean =>
      this.related.granted_submission_update.includes(ctx.subject),
    submission_delete: (ctx: Context): boolean =>
      this.related.granted_submission_delete.includes(ctx.subject),
    queue_manage: (ctx: Context): boolean =>
      this.related.granted_queue_manage.includes(ctx.subject),
    queue_view: (ctx: Context): boolean =>
      this.related.granted_queue_view.includes(ctx.subject),
    item_enqueue: (ctx: Context): boolean =>
      this.related.granted_item_enqueue.includes(ctx.subject),
    queue_item_view: (ctx: Context): boolean =>
      this.related.granted_queue_item_view.includes(ctx.subject),
    counter_manage: (ctx: Context): boolean =>
      this.related.granted_counter_manage.includes(ctx.subject),
    stats_view: (ctx: Context): boolean =>
      this.related.granted_stats_view.includes(ctx.subject),
  }
}

export { profile_user, tenancy_access, service_trustage }
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
