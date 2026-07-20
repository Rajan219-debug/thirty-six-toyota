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
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/humaecho5"
	"code.vikunja.io/api/pkg/web"

	"github.com/danielgtaylor/huma/v2"
)

// reportWindow is the shared date-range query for the timesheet and
// late-completion reports. Both bounds are optional; an omitted (zero) time
// means "unbounded on that side".
type reportWindow struct {
	Start time.Time `query:"start" doc:"Only include activity at or after this time (RFC3339). Omit for no lower bound."`
	End   time.Time `query:"end" doc:"Only include activity at or before this time (RFC3339). Omit for no upper bound."`
}

// nilIfZero turns a zero time into nil so the model layer can treat it as "unset".
func nilIfZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type timesheetReportBody struct {
	Body *models.TimesheetReport
}

type lateCompletionReportBody struct {
	Body *models.LateCompletionReport
}

type departmentEstimateReportBody struct {
	Body *models.DepartmentEstimateReport
}

// RegisterReportRoutes wires the timesheet and late-completion reports (§3.7 /
// §3.5 of the Thirty Six Toyota spec) onto the Huma API. Each is offered as JSON
// and as a CSV download. Reports are scoped to what the caller can read, which
// is what yields the individual / team / dealership-wide levels without a
// separate role model.
func RegisterReportRoutes(api huma.API) {
	tags := []string{"reports"}

	Register(api, huma.Operation{
		OperationID: "reports-timesheet",
		Summary:     "Timesheet report",
		Description: "Aggregates completed tracked time per employee, over the time entries the caller can read. Filter by user_id (one employee), project_id (one project/department) and a start/end window. Running timers are excluded.",
		Method:      http.MethodGet,
		Path:        "/reports/timesheet",
		Tags:        tags,
	}, reportsTimesheet)

	Register(api, huma.Operation{
		OperationID: "reports-timesheet-csv",
		Summary:     "Timesheet report as CSV",
		Description: "The same timesheet report as a downloadable CSV (employee, username, entries, total hours).",
		Method:      http.MethodGet,
		Path:        "/reports/timesheet.csv",
		Tags:        tags,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "The timesheet report as CSV.",
				Content: map[string]*huma.MediaType{
					"text/csv": {Schema: &huma.Schema{Type: huma.TypeString}},
				},
			},
		},
	}, reportsTimesheetCSV)

	Register(api, huma.Operation{
		OperationID: "reports-late-completions",
		Summary:     "Late-completion report",
		Description: "Counts tasks that closed late (overdue, then approved as completed_late) per assigned employee, over the projects the caller can read. Optional start/end window filters by completion time.",
		Method:      http.MethodGet,
		Path:        "/reports/late-completions",
		Tags:        tags,
	}, reportsLateCompletions)

	Register(api, huma.Operation{
		OperationID: "reports-department-estimates",
		Summary:     "Department estimated-vs-actual report",
		Description: "Compares estimated against tracked effort per project / department, over the tasks completed in the window (both sums cover the same task set). Scoped to the projects the caller can read.",
		Method:      http.MethodGet,
		Path:        "/reports/department-estimates",
		Tags:        tags,
	}, reportsDepartmentEstimates)

	Register(api, huma.Operation{
		OperationID: "reports-department-estimates-csv",
		Summary:     "Department estimated-vs-actual report as CSV",
		Description: "The same department estimates report as a downloadable CSV (department, tasks, estimated hours, actual hours).",
		Method:      http.MethodGet,
		Path:        "/reports/department-estimates.csv",
		Tags:        tags,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "The department estimates report as CSV.",
				Content: map[string]*huma.MediaType{
					"text/csv": {Schema: &huma.Schema{Type: huma.TypeString}},
				},
			},
		},
	}, reportsDepartmentEstimatesCSV)

	Register(api, huma.Operation{
		OperationID: "reports-late-completions-csv",
		Summary:     "Late-completion report as CSV",
		Description: "The same late-completion report as a downloadable CSV (employee, username, late count).",
		Method:      http.MethodGet,
		Path:        "/reports/late-completions.csv",
		Tags:        tags,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "The late-completion report as CSV.",
				Content: map[string]*huma.MediaType{
					"text/csv": {Schema: &huma.Schema{Type: huma.TypeString}},
				},
			},
		},
	}, reportsLateCompletionsCSV)
}

func init() { AddRouteRegistrar(RegisterReportRoutes) }

// buildTimesheetReport runs the shared session/permission plumbing for both the
// JSON and CSV timesheet handlers.
func buildTimesheetReport(a web.Auth, w reportWindow, userID, projectID int64) (*models.TimesheetReport, error) {
	s := db.NewBoundSession(a.GetID())
	defer s.Close()
	report, err := models.GetTimesheetReport(s, a, nilIfZero(w.Start), nilIfZero(w.End), userID, projectID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return report, nil
}

func reportsTimesheet(ctx context.Context, in *struct {
	reportWindow
	UserID    int64 `query:"user_id" doc:"Limit the report to a single employee."`
	ProjectID int64 `query:"project_id" doc:"Limit the report to a single project / department."`
}) (*timesheetReportBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	report, err := buildTimesheetReport(a, in.reportWindow, in.UserID, in.ProjectID)
	if err != nil {
		return nil, err
	}
	return &timesheetReportBody{Body: report}, nil
}

func reportsTimesheetCSV(ctx context.Context, in *struct {
	reportWindow
	UserID    int64 `query:"user_id" doc:"Limit the report to a single employee."`
	ProjectID int64 `query:"project_id" doc:"Limit the report to a single project / department."`
}) (*huma.StreamResponse, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	report, err := buildTimesheetReport(a, in.reportWindow, in.UserID, in.ProjectID)
	if err != nil {
		return nil, err
	}

	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		c := humaecho5.Unwrap(hctx)
		resp := (*c).Response()
		resp.Header().Set("Content-Type", "text/csv")
		resp.Header().Set("Content-Disposition", `attachment; filename="timesheet-report.csv"`)
		cw := csv.NewWriter(resp)
		_ = cw.Write([]string{"user_id", "username", "name", "entries", "total_seconds", "total_hours"})
		for _, r := range report.Rows {
			_ = cw.Write([]string{
				strconv.FormatInt(r.UserID, 10),
				csvSafe(r.Username),
				csvSafe(r.Name),
				strconv.FormatInt(r.EntryCount, 10),
				strconv.FormatInt(r.TotalSeconds, 10),
				secondsToHours(r.TotalSeconds),
			})
		}
		_ = cw.Write([]string{"", "", "TOTAL", "", strconv.FormatInt(report.TotalSeconds, 10), secondsToHours(report.TotalSeconds)})
		cw.Flush()
	}}, nil
}

func reportsLateCompletions(ctx context.Context, in *struct {
	reportWindow
}) (*lateCompletionReportBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewBoundSession(a.GetID())
	defer s.Close()
	report, err := models.GetLateCompletionReport(s, a, nilIfZero(in.Start), nilIfZero(in.End))
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &lateCompletionReportBody{Body: report}, nil
}

func reportsLateCompletionsCSV(ctx context.Context, in *struct {
	reportWindow
}) (*huma.StreamResponse, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewBoundSession(a.GetID())
	defer s.Close()
	report, err := models.GetLateCompletionReport(s, a, nilIfZero(in.Start), nilIfZero(in.End))
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		c := humaecho5.Unwrap(hctx)
		resp := (*c).Response()
		resp.Header().Set("Content-Type", "text/csv")
		resp.Header().Set("Content-Disposition", `attachment; filename="late-completion-report.csv"`)
		cw := csv.NewWriter(resp)
		_ = cw.Write([]string{"user_id", "username", "name", "late_count"})
		for _, r := range report.Rows {
			_ = cw.Write([]string{
				strconv.FormatInt(r.UserID, 10),
				csvSafe(r.Username),
				csvSafe(r.Name),
				strconv.FormatInt(r.LateCount, 10),
			})
		}
		cw.Flush()
	}}, nil
}

// buildDepartmentEstimateReport runs the shared session/permission plumbing for
// both the JSON and CSV department-estimate handlers.
func buildDepartmentEstimateReport(a web.Auth, w reportWindow) (*models.DepartmentEstimateReport, error) {
	s := db.NewBoundSession(a.GetID())
	defer s.Close()
	report, err := models.GetDepartmentEstimateReport(s, a, nilIfZero(w.Start), nilIfZero(w.End))
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return report, nil
}

func reportsDepartmentEstimates(ctx context.Context, in *struct {
	reportWindow
}) (*departmentEstimateReportBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	report, err := buildDepartmentEstimateReport(a, in.reportWindow)
	if err != nil {
		return nil, err
	}
	return &departmentEstimateReportBody{Body: report}, nil
}

func reportsDepartmentEstimatesCSV(ctx context.Context, in *struct {
	reportWindow
}) (*huma.StreamResponse, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	report, err := buildDepartmentEstimateReport(a, in.reportWindow)
	if err != nil {
		return nil, err
	}

	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		c := humaecho5.Unwrap(hctx)
		resp := (*c).Response()
		resp.Header().Set("Content-Type", "text/csv")
		resp.Header().Set("Content-Disposition", `attachment; filename="department-estimates-report.csv"`)
		cw := csv.NewWriter(resp)
		_ = cw.Write([]string{"project_id", "department", "tasks", "estimated_seconds", "estimated_hours", "actual_seconds", "actual_hours"})
		for _, r := range report.Rows {
			_ = cw.Write([]string{
				strconv.FormatInt(r.ProjectID, 10),
				csvSafe(r.ProjectTitle),
				strconv.FormatInt(r.TaskCount, 10),
				strconv.FormatInt(r.EstimatedSeconds, 10),
				secondsToHours(r.EstimatedSeconds),
				strconv.FormatInt(r.ActualSeconds, 10),
				secondsToHours(r.ActualSeconds),
			})
		}
		_ = cw.Write([]string{"", "TOTAL", "", strconv.FormatInt(report.TotalEstimated, 10), secondsToHours(report.TotalEstimated), strconv.FormatInt(report.TotalActual, 10), secondsToHours(report.TotalActual)})
		cw.Flush()
	}}, nil
}

// csvSafe neutralises spreadsheet formula (CSV) injection. A user-controlled
// cell (username, display name, project/department title) that begins with
// =, +, -, @ or a control char is executed as a formula when the CSV is opened
// in Excel/Sheets. Prefixing with a single quote forces it to render as literal
// text. encoding/csv only escapes for CSV structure, not for this.
func csvSafe(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	}
	return v
}

// secondsToHours renders whole seconds as decimal hours to two places, for the
// human-facing CSV column (the raw seconds stay in their own column).
func secondsToHours(seconds int64) string {
	return strconv.FormatFloat(float64(seconds)/3600, 'f', 2, 64)
}
