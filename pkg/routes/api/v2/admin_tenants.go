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
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

type tenantsListBody struct {
	Body []*models.Tenant
}

type tenantBody struct {
	Body *models.Tenant
}

// createTenantBody accepts only the client-settable fields, so id/created/updated
// can't be spoofed via the request body.
type createTenantBody struct {
	Name string `json:"name" minLength:"1" maxLength:"250" doc:"Display name of the outlet, e.g. Faridabad."`
	Slug string `json:"slug" minLength:"1" maxLength:"100" doc:"Short, unique slug for the outlet, e.g. faridabad."`
}

type setOutletBody struct {
	TenantID int64 `json:"tenant_id" doc:"The outlet (tenant) id to assign the user to."`
	IsHQ     bool  `json:"is_hq" doc:"Whether the user is an HQ / group manager who reads across all outlets."`
}

type outletAssignment struct {
	UserID   int64 `json:"user_id"`
	TenantID int64 `json:"tenant_id"`
	IsHQ     bool  `json:"is_hq"`
}

type outletAssignmentBody struct {
	Body *outletAssignment
}

// Permissions are enforced by the gateV2AdminRoutes path middleware, not per-handler.
func RegisterAdminTenantRoutes(api huma.API) {
	tags := []string{"admin"}

	Register(api, huma.Operation{
		OperationID: "admin-tenants-list",
		Summary:     "List outlets",
		Description: "Returns every outlet (tenant). Restricted to instance admins.",
		Method:      http.MethodGet,
		Path:        "/admin/tenants",
		Tags:        tags,
	}, adminTenantsList)

	Register(api, huma.Operation{
		OperationID: "admin-tenants-create",
		Summary:     "Create an outlet",
		Description: "Creates a new outlet (tenant), e.g. Faridabad or Palwal. Restricted to instance admins.",
		Method:      http.MethodPost,
		Path:        "/admin/tenants",
		Tags:        tags,
	}, adminTenantsCreate)

	Register(api, huma.Operation{
		OperationID: "admin-tenants-set-default",
		Summary:     "Set the default outlet",
		Description: "Makes this outlet the one new users are created in. Doing so is also required before the previous default can be deleted. Restricted to instance admins.",
		Method:      http.MethodPatch,
		Path:        "/admin/tenants/{id}/default",
		Tags:        tags,
	}, adminTenantsSetDefault)

	Register(api, huma.Operation{
		OperationID: "admin-tenants-delete",
		Summary:     "Delete an outlet",
		Description: "Deletes an outlet. Refuses if it is the default outlet or still has users or data assigned to it. Restricted to instance admins.",
		Method:      http.MethodDelete,
		Path:        "/admin/tenants/{id}",
		Tags:        tags,
	}, adminTenantsDelete)

	Register(api, huma.Operation{
		OperationID: "admin-users-set-outlet",
		Summary:     "Assign a user to an outlet",
		Description: "Sets which outlet a user belongs to and whether they are an HQ / group manager (reads across all outlets). Restricted to instance admins.",
		Method:      http.MethodPatch,
		Path:        "/admin/users/{id}/outlet",
		Tags:        tags,
	}, adminUsersSetOutlet)
}

func init() { AddRouteRegistrar(RegisterAdminTenantRoutes) }

func adminTenantsList(_ context.Context, _ *struct{}) (*tenantsListBody, error) {
	// Instance-admin action: legitimately cross-tenant (creating/assigning/deleting
	// outlets touches more than the acting admin's own outlet), so this needs
	// system-wide visibility rather than a session pinned to one tenant.
	s := db.NewSystemSession()
	defer s.Close()
	tenants, err := models.GetAllTenants(s)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &tenantsListBody{Body: tenants}, nil
}

func adminTenantsCreate(_ context.Context, in *struct{ Body createTenantBody }) (*tenantBody, error) {
	// Instance-admin action: legitimately cross-tenant (creating/assigning/deleting
	// outlets touches more than the acting admin's own outlet), so this needs
	// system-wide visibility rather than a session pinned to one tenant.
	s := db.NewSystemSession()
	defer s.Close()

	exists, err := s.Where("slug = ?", in.Body.Slug).Exist(&models.Tenant{})
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if exists {
		_ = s.Rollback()
		return nil, huma.Error409Conflict("an outlet with this slug already exists")
	}

	t := models.Tenant{Name: in.Body.Name, Slug: in.Body.Slug}
	if err := models.CreateTenant(s, &t); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &tenantBody{Body: &t}, nil
}

func adminTenantsSetDefault(_ context.Context, in *struct {
	ID int64 `path:"id" doc:"The numeric ID of the outlet to make default."`
}) (*struct{}, error) {
	// Instance-admin action: legitimately cross-tenant (creating/assigning/deleting
	// outlets touches more than the acting admin's own outlet), so this needs
	// system-wide visibility rather than a session pinned to one tenant.
	s := db.NewSystemSession()
	defer s.Close()
	if err := models.SetDefaultOutlet(s, in.ID); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &struct{}{}, nil
}

func adminTenantsDelete(_ context.Context, in *struct {
	ID int64 `path:"id" doc:"The numeric ID of the outlet to delete."`
}) (*struct{}, error) {
	// Instance-admin action: legitimately cross-tenant (creating/assigning/deleting
	// outlets touches more than the acting admin's own outlet), so this needs
	// system-wide visibility rather than a session pinned to one tenant.
	s := db.NewSystemSession()
	defer s.Close()
	if err := models.DeleteTenant(s, in.ID); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &struct{}{}, nil
}

func adminUsersSetOutlet(_ context.Context, in *struct {
	ID   int64 `path:"id" doc:"The numeric ID of the user."`
	Body setOutletBody
}) (*outletAssignmentBody, error) {
	// Instance-admin action: legitimately cross-tenant (creating/assigning/deleting
	// outlets touches more than the acting admin's own outlet), so this needs
	// system-wide visibility rather than a session pinned to one tenant.
	s := db.NewSystemSession()
	defer s.Close()
	if err := models.SetUserOutlet(s, in.ID, in.Body.TenantID, in.Body.IsHQ); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &outletAssignmentBody{Body: &outletAssignment{UserID: in.ID, TenantID: in.Body.TenantID, IsHQ: in.Body.IsHQ}}, nil
}
