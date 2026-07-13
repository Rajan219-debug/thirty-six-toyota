// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package apiv2

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

// --- Notes (plain CRUD, private per-user; mirrors folders in documents.go) ---

type noteListBody struct {
	Body Paginated[*models.Note]
}

type noteReadBody struct {
	models.Note
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The maximum permission the requesting user has on this note (2=admin for the owner)."`
}

func RegisterNoteRoutes(api huma.API) {
	tags := []string{"notes"}

	Register(api, huma.Operation{
		OperationID: "notes-list",
		Summary:     "List notes",
		Description: "Returns every note owned by the authenticated user. Notes are private — a user only ever sees their own.",
		Method:      http.MethodGet, Path: "/notes", Tags: tags,
	}, notesList)

	Register(api, huma.Operation{
		OperationID: "notes-read",
		Summary:     "Get a note",
		Description: "Returns a single note. Only the owner may read it.",
		Method:      http.MethodGet, Path: "/notes/{id}", Tags: tags,
	}, notesRead)

	Register(api, huma.Operation{
		OperationID: "notes-create",
		Summary:     "Create a note",
		Description: "Creates a note owned by the authenticated user.",
		Method:      http.MethodPost, Path: "/notes", Tags: tags,
	}, notesCreate)

	Register(api, huma.Operation{
		OperationID: "notes-update",
		Summary:     "Update a note",
		Description: "Updates a note's title, body, color or pinned state. Only the owner may update it.",
		Method:      http.MethodPut, Path: "/notes/{id}", Tags: tags,
	}, notesUpdate)

	Register(api, huma.Operation{
		OperationID: "notes-delete",
		Summary:     "Delete a note",
		Description: "Deletes a note. Only the owner may delete it.",
		Method:      http.MethodDelete, Path: "/notes/{id}", Tags: tags,
	}, notesDelete)
}

func init() { AddRouteRegistrar(RegisterNoteRoutes) }

func notesList(ctx context.Context, in *struct {
	ListParams
}) (*noteListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.Note{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.Note)
	if !ok {
		return nil, fmt.Errorf("notes.ReadAll returned unexpected type %T", result)
	}
	return &noteListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

func notesRead(ctx context.Context, in *struct {
	ID int64 `path:"id"`
	conditional.Params
}) (*singleReadBody[noteReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	note := &models.Note{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, note, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &noteReadBody{Note: *note, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, note.Updated, maxPermission)
}

func notesCreate(ctx context.Context, in *struct {
	Body models.Note
}) (*singleBody[models.Note], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Note]{Body: &in.Body}, nil
}

func notesUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id"`
	Body noteReadBody
}) (*singleBody[models.Note], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	note := &in.Body.Note
	note.ID = in.ID
	if err := handler.DoUpdate(ctx, note, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Note]{Body: note}, nil
}

func notesDelete(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.Note{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
