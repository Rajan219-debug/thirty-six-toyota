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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/humaecho5"
	webfiles "code.vikunja.io/api/pkg/web/files"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

// --- Folders (plain CRUD, mirrors labels.go) ---

type folderListBody struct {
	Body Paginated[*models.Folder]
}

type folderReadBody struct {
	models.Folder
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The maximum permission the requesting user has on this folder (0=read, 2=admin for the owner)."`
}

func RegisterFolderRoutes(api huma.API) {
	tags := []string{"documents"}

	Register(api, huma.Operation{
		OperationID: "folders-list",
		Summary:     "List folders",
		Description: "Returns every folder in the shared company drive — every authenticated user can see every folder.",
		Method:      http.MethodGet, Path: "/folders", Tags: tags,
	}, foldersList)

	Register(api, huma.Operation{
		OperationID: "folders-read",
		Summary:     "Get a folder",
		Description: "Returns a single folder.",
		Method:      http.MethodGet, Path: "/folders/{id}", Tags: tags,
	}, foldersRead)

	Register(api, huma.Operation{
		OperationID: "folders-create",
		Summary:     "Create a folder",
		Description: "Creates a folder; the authenticated user becomes its owner.",
		Method:      http.MethodPost, Path: "/folders", Tags: tags,
	}, foldersCreate)

	Register(api, huma.Operation{
		OperationID: "folders-update",
		Summary:     "Rename a folder",
		Description: "Renames a folder. Only the owner may update it.",
		Method:      http.MethodPut, Path: "/folders/{id}", Tags: tags,
	}, foldersUpdate)

	Register(api, huma.Operation{
		OperationID: "folders-delete",
		Summary:     "Delete a folder",
		Description: "Deletes a folder and every document inside it. Only the owner may delete it.",
		Method:      http.MethodDelete, Path: "/folders/{id}", Tags: tags,
	}, foldersDelete)
}

func init() { AddRouteRegistrar(RegisterFolderRoutes) }

func foldersList(ctx context.Context, in *struct {
	ListParams
}) (*folderListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.Folder{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.Folder)
	if !ok {
		return nil, fmt.Errorf("folders.ReadAll returned unexpected type %T", result)
	}
	return &folderListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

func foldersRead(ctx context.Context, in *struct {
	ID int64 `path:"id"`
	conditional.Params
}) (*singleReadBody[folderReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	folder := &models.Folder{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, folder, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &folderReadBody{Folder: *folder, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, folder.Updated, maxPermission)
}

func foldersCreate(ctx context.Context, in *struct {
	Body models.Folder
}) (*singleBody[models.Folder], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Folder]{Body: &in.Body}, nil
}

func foldersUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id"`
	Body folderReadBody
}) (*singleBody[models.Folder], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	folder := &in.Body.Folder
	folder.ID = in.ID
	if err := handler.DoUpdate(ctx, folder, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.Folder]{Body: folder}, nil
}

func foldersDelete(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.Folder{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

// --- Documents (list is plain CRUD; upload/download/delete are custom, mirroring task_attachments.go) ---

type documentListBody struct {
	Body Paginated[*models.Document]
}

type documentUploadError struct {
	Message string `json:"message" doc:"A human-readable description of why this file failed."`
}

type documentUploadResult struct {
	Success []*models.Document    `json:"success"`
	Errors  []documentUploadError `json:"errors"`
}

type documentUploadInput struct {
	FolderID int64 `path:"folder" doc:"The id of the folder to upload the files into."`
	RawBody  huma.MultipartFormFiles[struct {
		Files []huma.FormFile `form:"files" required:"true" doc:"One or more files to upload into the folder. Send multiple parts under the same \"files\" field to upload several at once."`
	}]
}

type documentUploadBody struct {
	Body *documentUploadResult
}

func RegisterDocumentRoutes(api huma.API) {
	tags := []string{"documents"}

	Register(api, huma.Operation{
		OperationID: "documents-list",
		Summary:     "List a folder's documents",
		Description: "Returns the document metadata for one folder, paginated. The file bytes are not included; fetch them from the download endpoint.",
		Method:      http.MethodGet, Path: "/folders/{folder}/documents", Tags: tags,
	}, documentsList)

	Register(api, huma.Operation{
		OperationID: "documents-upload",
		Summary:     "Upload documents",
		Description: "Uploads one or more files into a folder via multipart/form-data under the \"files\" field. Each file is processed independently: a file that fails is reported in the errors list while the others still succeed, so the request returns 201 even on a partial upload.",
		Method:      http.MethodPost, Path: "/folders/{folder}/documents", Tags: tags,
	}, documentsUpload)

	Register(api, huma.Operation{
		OperationID: "documents-download",
		Summary:     "Download a document",
		Description: "Returns the raw bytes of one document. The Content-Type header carries the file's real mime type.",
		Method:      http.MethodGet, Path: "/documents/{document}/download", Tags: tags,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "The document file bytes. The Content-Type header carries the file's mime type.",
				Content: map[string]*huma.MediaType{
					"application/octet-stream": {
						Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"},
					},
				},
			},
		},
	}, documentsDownload)

	Register(api, huma.Operation{
		OperationID: "documents-delete",
		Summary:     "Delete a document",
		Description: "Deletes one document and its underlying file. Only the uploader may delete it.",
		Method:      http.MethodDelete, Path: "/documents/{document}", Tags: tags,
	}, documentsDelete)
}

func init() { AddRouteRegistrar(RegisterDocumentRoutes) }

func documentsList(ctx context.Context, in *struct {
	FolderID int64 `path:"folder" doc:"The id of the folder whose documents to list."`
	ListParams
}) (*documentListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &models.Document{FolderID: in.FolderID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*models.Document)
	if !ok {
		return nil, fmt.Errorf("documents.ReadAll returned unexpected type %T", result)
	}
	return &documentListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

// documentsUpload owns auth, the session and the permission check because there
// is no handler.Do* for multipart uploads (see the api-v2-routes skill's
// "Non-CRUDable / custom routes" section).
func documentsUpload(ctx context.Context, in *documentUploadInput) (*documentUploadBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewBoundSession(a.GetID())
	defer s.Close()

	formFiles := in.RawBody.Data().Files
	uploads := make([]*models.DocumentToUpload, 0, len(formFiles))
	for _, file := range formFiles {
		uploads = append(uploads, &models.DocumentToUpload{Reader: file, Filename: file.Filename, Size: uint64(file.Size)})
	}

	success, failures, err := models.UploadDocuments(s, a, in.FolderID, uploads)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	errs := make([]documentUploadError, 0, len(failures))
	for _, e := range failures {
		errs = append(errs, documentUploadError{Message: e.Error()})
	}
	return &documentUploadBody{Body: &documentUploadResult{Success: success, Errors: errs}}, nil
}

// documentsDownload owns auth, the session and the permission check; there is no
// handler.Do* for a file body.
func documentsDownload(ctx context.Context, in *struct {
	DocumentID int64 `path:"document" doc:"The id of the document to download."`
}) (*huma.StreamResponse, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewBoundSession(a.GetID())
	defer s.Close()

	doc, err := models.LoadDocumentForDownload(s, a, in.DocumentID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	if err := s.Commit(); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		c := humaecho5.Unwrap(hctx)
		webfiles.WriteFileDownload((*c).Response(), (*c).Request(), doc.File)
	}}, nil
}

func documentsDelete(ctx context.Context, in *struct {
	DocumentID int64 `path:"document" doc:"The id of the document to delete."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &models.Document{ID: in.DocumentID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
