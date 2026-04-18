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

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

// FormDefinitionHandler handles form definition HTTP endpoints.
type FormDefinitionHandler struct {
	biz business.FormStoreBusiness
}

// NewFormDefinitionHandler creates a new FormDefinitionHandler.
func NewFormDefinitionHandler(biz business.FormStoreBusiness) *FormDefinitionHandler {
	return &FormDefinitionHandler{biz: biz}
}

// Create handles POST /api/v1/form-definitions.
func (h *FormDefinitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	var req struct {
		FormID      string          `json:"form_id"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		JSONSchema  json.RawMessage `json:"json_schema"`
		Active      *bool           `json:"active"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if req.FormID == "" || req.Name == "" {
		http.Error(w, "form_id and name are required", http.StatusBadRequest)
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	def := &models.FormDefinition{
		FormID:      req.FormID,
		Name:        req.Name,
		Description: req.Description,
		Active:      active,
	}

	if req.JSONSchema != nil {
		def.JSONSchema = string(req.JSONSchema)
	}

	if err := h.biz.CreateDefinition(ctx, def); err != nil {
		log.WithError(err).Error("failed to create form definition", "form_id", req.FormID)
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(definitionToJSON(def))
}

// List handles GET /api/v1/form-definitions.
func (h *FormDefinitionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	activeOnly := r.URL.Query().Get("active") == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	defs, err := h.biz.ListDefinitions(ctx, activeOnly, limit, offset)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	items := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		items = append(items, definitionToJSON(d))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// Get handles GET /api/v1/form-definitions/{id}.
func (h *FormDefinitionHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	def, err := h.biz.GetDefinition(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(definitionToJSON(def))
}

// Update handles PUT /api/v1/form-definitions/{id}.
func (h *FormDefinitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := util.Log(ctx)

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	def, err := h.biz.GetDefinition(ctx, id)
	if err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	var req struct {
		Name        *string         `json:"name"`
		Description *string         `json:"description"`
		JSONSchema  json.RawMessage `json:"json_schema"`
		Active      *bool           `json:"active"`
	}

	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		def.Name = *req.Name
	}

	if req.Description != nil {
		def.Description = *req.Description
	}

	if req.JSONSchema != nil {
		def.JSONSchema = string(req.JSONSchema)
	}

	if req.Active != nil {
		def.Active = *req.Active
	}

	if updateErr := h.biz.UpdateDefinition(ctx, def); updateErr != nil {
		log.WithError(updateErr).Error("failed to update form definition", "definition_id", id)
		status, msg := httpStatusForError(updateErr)
		http.Error(w, msg, status)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(definitionToJSON(def))
}

// Delete handles DELETE /api/v1/form-definitions/{id}.
func (h *FormDefinitionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !requireAuth(ctx, w) {
		return
	}

	id := r.PathValue("id")

	if err := h.biz.DeleteDefinition(ctx, id); err != nil {
		status, msg := httpStatusForError(err)
		http.Error(w, msg, status)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func definitionToJSON(def *models.FormDefinition) map[string]any {
	result := map[string]any{
		"id":          def.ID,
		"form_id":     def.FormID,
		"name":        def.Name,
		"description": def.Description,
		"active":      def.Active,
		"created_at":  def.CreatedAt,
		"modified_at": def.ModifiedAt,
	}

	if def.JSONSchema != "" {
		result["json_schema"] = json.RawMessage(def.JSONSchema)
	}

	return result
}
