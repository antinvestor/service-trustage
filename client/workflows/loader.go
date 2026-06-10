package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/pitabwire/util"

	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
)

type WorkflowClient interface {
	ListWorkflows(
		context.Context,
		*connect.Request[workflowv1.ListWorkflowsRequest],
	) (*connect.Response[workflowv1.ListWorkflowsResponse], error)
	CreateWorkflow(
		context.Context,
		*connect.Request[workflowv1.CreateWorkflowRequest],
	) (*connect.Response[workflowv1.CreateWorkflowResponse], error)
	ActivateWorkflow(
		context.Context,
		*connect.Request[workflowv1.ActivateWorkflowRequest],
	) (*connect.Response[workflowv1.ActivateWorkflowResponse], error)
}

func SyncFromDir(ctx context.Context, client WorkflowClient, dir string) error {
	log := util.Log(ctx)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.WithField("dir", dir).Debug("workflows: directory does not exist, skipping")
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("workflows: read dir %s: %w", dir, err)
	}

	var synced, skipped, created int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())

		action, syncErr := syncOne(ctx, client, path)
		if syncErr != nil {
			return fmt.Errorf("workflows: sync %s: %w", entry.Name(), syncErr)
		}

		switch action {
		case "created":
			created++
			log.WithField("file", entry.Name()).Info("workflows: created and activated")
		case "skipped":
			skipped++
			log.WithField("file", entry.Name()).Debug("workflows: already exists, skipped")
		}
		synced++
	}

	log.WithField("synced", synced).
		WithField("created", created).
		WithField("skipped", skipped).
		Info("workflows: sync complete")
	return nil
}

func syncOne(ctx context.Context, client WorkflowClient, path string) (string, error) {
	dslStruct, name, err := parseDSLFile(path)
	if err != nil {
		return "", err
	}

	hash := dslHash(dslStruct)

	listResp, err := client.ListWorkflows(ctx, connect.NewRequest(&workflowv1.ListWorkflowsRequest{
		Name: name,
	}))
	if err != nil {
		return "", fmt.Errorf("list workflows: %w", err)
	}

	for _, existing := range listResp.Msg.GetItems() {
		if existing.GetInputSchemaHash() == hash &&
			existing.GetStatus() == workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE {
			return "skipped", nil
		}
	}

	createResp, err := client.CreateWorkflow(ctx, connect.NewRequest(&workflowv1.CreateWorkflowRequest{
		Dsl: dslStruct,
	}))
	if err != nil {
		if isAlreadyExists(err) {
			return "skipped", nil
		}
		return "", fmt.Errorf("create workflow %s: %w", name, err)
	}

	wfID := createResp.Msg.GetWorkflow().GetId()

	_, err = client.ActivateWorkflow(ctx, connect.NewRequest(&workflowv1.ActivateWorkflowRequest{
		Id: wfID,
	}))
	if err != nil {
		return "", fmt.Errorf("activate workflow %s (id=%s): %w", name, wfID, err)
	}

	return "created", nil
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint")
}
