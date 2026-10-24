package workflows

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"connectrpc.com/connect/v2"
	"github.com/pitabwire/util"

	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
)

// WorkflowClient is the subset of workflowv1connect.WorkflowServiceClient
// the loader needs; that client satisfies it.
type WorkflowClient interface {
	ListWorkflows(
		context.Context, *workflowv1.ListWorkflowsRequest,
	) (*workflowv1.ListWorkflowsResponse, error)
	CreateWorkflow(
		context.Context, *workflowv1.CreateWorkflowRequest,
	) (*workflowv1.CreateWorkflowResponse, error)
	ActivateWorkflow(
		context.Context, *workflowv1.ActivateWorkflowRequest,
	) (*workflowv1.ActivateWorkflowResponse, error)
}

const (
	syncCreated   = "created"
	syncUnchanged = "unchanged"
	syncSkipped   = "skipped"
)

// SyncFromDir registers every *.json workflow document directly inside dir.
// A missing directory is a no-op. See SyncFromFS.
func SyncFromDir(ctx context.Context, client WorkflowClient, dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		util.Log(ctx).WithField("dir", dir).Debug("workflows: directory does not exist, skipping")
		return nil
	}

	return SyncFromFS(ctx, client, os.DirFS(dir))
}

// SyncFromFS registers every *.json workflow document at the root of fsys
// (for example a go:embed'ed directory), in name order.
//
// Each document goes through CreateWorkflow, which is idempotent: an
// identical live version is returned as is, and when that version is ACTIVE
// the server re-asserts its trigger bindings and signal routes. A version
// that is not yet ACTIVE is then activated, which switches the workflow's
// trigger bindings to it.
func SyncFromFS(ctx context.Context, client WorkflowClient, fsys fs.FS) error {
	log := util.Log(ctx)

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("workflows: read dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	var created, unchanged, skipped int
	for _, name := range names {
		raw, readErr := fs.ReadFile(fsys, name)
		if readErr != nil {
			return fmt.Errorf("workflows: read %s: %w", name, readErr)
		}

		action, syncErr := syncOne(ctx, client, name, raw)
		if syncErr != nil {
			return fmt.Errorf("workflows: sync %s: %w", name, syncErr)
		}

		switch action {
		case syncCreated:
			created++
			log.WithField("file", name).Info("workflows: created and activated")
		case syncUnchanged:
			unchanged++
			log.WithField("file", name).Debug("workflows: already active, routing re-asserted")
		case syncSkipped:
			skipped++
			log.WithField("file", name).Warn("workflows: already exists, skipped")
		}
	}

	log.WithField("synced", len(names)).
		WithField("created", created).
		WithField("unchanged", unchanged).
		WithField("skipped", skipped).
		Info("workflows: sync complete")
	return nil
}

func syncOne(ctx context.Context, client WorkflowClient, label string, raw []byte) (string, error) {
	dslStruct, name, err := parseDSL(raw, label)
	if err != nil {
		return "", err
	}

	createResp, err := client.CreateWorkflow(ctx, &workflowv1.CreateWorkflowRequest{
		Dsl: dslStruct,
	})
	if err != nil {
		if isAlreadyExists(err) {
			return syncSkipped, nil
		}
		return "", fmt.Errorf("create workflow %s: %w", name, err)
	}

	wf := createResp.GetWorkflow()
	if wf.GetStatus() == workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE {
		return syncUnchanged, nil
	}

	_, err = client.ActivateWorkflow(ctx, &workflowv1.ActivateWorkflowRequest{
		Id: wf.GetId(),
	})
	if err != nil {
		return "", fmt.Errorf("activate workflow %s (id=%s): %w", name, wf.GetId(), err)
	}

	return syncCreated, nil
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	if connect.CodeOf(err) == connect.CodeAlreadyExists {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint")
}
