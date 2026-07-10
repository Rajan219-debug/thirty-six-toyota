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

type dashboardStatsBody struct {
	Body *models.DashboardStats
}

// RegisterDashboardRoutes wires the manager-dashboard aggregate stats onto the
// Huma API. The numbers are scoped to the projects the caller can read.
func RegisterDashboardRoutes(api huma.API) {
	Register(api, huma.Operation{
		OperationID: "dashboard-stats",
		Summary:     "Manager dashboard stats",
		Description: "Team-wide task counts (open, overdue, awaiting confirmation), completion rate, per-employee on-time efficiency and per-department open-task allocation, scoped to the projects the caller can read.",
		Method:      http.MethodGet,
		Path:        "/dashboard/stats",
		Tags:        []string{"dashboard"},
	}, dashboardStats)
}

func init() { AddRouteRegistrar(RegisterDashboardRoutes) }

func dashboardStats(ctx context.Context, _ *struct{}) (*dashboardStatsBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()
	stats, err := models.GetDashboardStats(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &dashboardStatsBody{Body: stats}, nil
}
