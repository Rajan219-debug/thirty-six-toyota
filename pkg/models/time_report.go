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

package models

import (
	"sort"
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// TimesheetReportRow is one employee's tracked-time total within the report window.
type TimesheetReportRow struct {
	UserID       int64  `json:"user_id" doc:"The employee this row aggregates."`
	Username     string `json:"username" doc:"The employee's username."`
	Name         string `json:"name" doc:"The employee's full name (falls back to the username)."`
	EntryCount   int64  `json:"entry_count" doc:"How many completed time entries were summed for this employee."`
	TotalSeconds int64  `json:"total_seconds" doc:"Total tracked seconds across those entries."`
}

// TimesheetReport aggregates completed time entries per employee, scoped to what
// the requester can read (which is what gives the individual / team / dealership
// levels: a team lead admin on the department's projects sees their team, the
// owner sees everyone). Running timers (no end time) are excluded — they have no
// finished duration yet.
type TimesheetReport struct {
	Start        *time.Time            `json:"start" doc:"Start of the window, if one was given."`
	End          *time.Time            `json:"end" doc:"End of the window, if one was given."`
	Rows         []*TimesheetReportRow `json:"rows" doc:"One row per employee with tracked time, sorted by name."`
	TotalSeconds int64                 `json:"total_seconds" doc:"Grand total tracked seconds across all rows."`
}

// entryDuration is the finished duration of a time entry in whole seconds; a
// running entry (nil end) or a negative interval counts as zero. Pulled out so
// the summing rule has a pure, unit-testable home.
func entryDuration(startTime time.Time, endTime *time.Time) int64 {
	if endTime == nil {
		return 0
	}
	secs := int64(endTime.Sub(startTime).Seconds())
	if secs < 0 {
		return 0
	}
	return secs
}

// GetTimesheetReport aggregates tracked time per employee over the readable set
// of time entries. userID and projectID (both optional, 0 = ignore) narrow the
// report to one employee or one project/department; start/end (optional) bound
// it by entry start time.
func GetTimesheetReport(s *xorm.Session, a web.Auth, start, end *time.Time, userID, projectID int64) (*TimesheetReport, error) {
	report := &TimesheetReport{Start: start, End: end, Rows: []*TimesheetReportRow{}}

	// Link shares have no time-tracking access (mirrors TimeEntry.ReadAll).
	if _, isShare := a.(*LinkSharing); isShare {
		return report, nil
	}

	cond := readableTimeEntriesCond(a).
		And(builder.NotNull{"end_time"}) // only finished entries contribute a duration
	if userID > 0 {
		cond = cond.And(builder.Eq{"user_id": userID})
	}
	if projectID > 0 {
		cond = cond.And(entriesForProjectCond(builder.Eq{"project_id": projectID}))
	}
	if start != nil {
		cond = cond.And(builder.Gte{"start_time": *start})
	}
	if end != nil {
		cond = cond.And(builder.Lte{"start_time": *end})
	}

	entries := []*TimeEntry{}
	if err := s.Where(cond).OrderBy("start_time ASC").Find(&entries); err != nil {
		return nil, err
	}

	// Aggregate in Go rather than SQL: end_time - start_time arithmetic differs
	// per DB engine, and a dealership's entry volume is tiny.
	rowByUser := map[int64]*TimesheetReportRow{}
	userOrder := []int64{}
	for _, e := range entries {
		row := rowByUser[e.UserID]
		if row == nil {
			row = &TimesheetReportRow{UserID: e.UserID}
			rowByUser[e.UserID] = row
			userOrder = append(userOrder, e.UserID)
		}
		secs := entryDuration(e.StartTime, e.EndTime)
		row.EntryCount++
		row.TotalSeconds += secs
		report.TotalSeconds += secs
	}

	if err := fillReportUsers(s, userOrder, rowByUser); err != nil {
		return nil, err
	}

	report.Rows = sortedTimesheetRows(userOrder, rowByUser)
	return report, nil
}

// fillReportUsers stamps username/name onto each row from a single batch user load.
func fillReportUsers(s *xorm.Session, userIDs []int64, rows map[int64]*TimesheetReportRow) error {
	if len(userIDs) == 0 {
		return nil
	}
	users, err := user.GetUsersByIDs(s, userIDs)
	if err != nil {
		return err
	}
	for id, row := range rows {
		if u, ok := users[id]; ok {
			row.Username = u.Username
			row.Name = u.GetName()
		}
	}
	return nil
}

func sortedTimesheetRows(order []int64, rows map[int64]*TimesheetReportRow) []*TimesheetReportRow {
	out := make([]*TimesheetReportRow, 0, len(order))
	for _, id := range order {
		out = append(out, rows[id])
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].UserID < out[j].UserID
	})
	return out
}

// LateCompletionReportRow is one employee's count of tasks that closed late
// (overdue, then approved as completed_late).
type LateCompletionReportRow struct {
	UserID    int64  `json:"user_id" doc:"The employee this row counts late completions for."`
	Username  string `json:"username" doc:"The employee's username."`
	Name      string `json:"name" doc:"The employee's full name (falls back to the username)."`
	LateCount int64  `json:"late_count" doc:"How many of the employee's assigned tasks closed late in the window."`
}

// LateCompletionReport is the per-employee late-completion tally backing the
// §3.5 "late-completion count tracked per employee" performance metric.
type LateCompletionReport struct {
	Start *time.Time                 `json:"start" doc:"Start of the window, if one was given."`
	End   *time.Time                 `json:"end" doc:"End of the window, if one was given."`
	Rows  []*LateCompletionReportRow `json:"rows" doc:"One row per employee with late completions, most first."`
	Total int64                      `json:"total" doc:"Total late completions across all rows."`
}

// GetLateCompletionReport tallies completed_late tasks per assignee, scoped to
// the projects the requester can access. start/end (optional) bound it by the
// task's completion time.
func GetLateCompletionReport(s *xorm.Session, a web.Auth, start, end *time.Time) (*LateCompletionReport, error) {
	report := &LateCompletionReport{Start: start, End: end, Rows: []*LateCompletionReportRow{}}

	if _, isShare := a.(*LinkSharing); isShare {
		return report, nil
	}

	// Candidate tasks: done + confirmed closure, in an accessible project. The
	// completed_late decision itself is left to CalculateTaskStatus so the report
	// can never drift from the lifecycle rule.
	cond := accessibleProjectIDsSubquery(a, "project_id").
		And(builder.Eq{"done": true}).
		And(builder.Neq{"confirmed_by_id": 0})
	if start != nil {
		cond = cond.And(builder.Gte{"done_at": *start})
	}
	if end != nil {
		cond = cond.And(builder.Lte{"done_at": *end})
	}

	var tasks []*Task
	if err := s.Where(cond).Find(&tasks); err != nil {
		return nil, err
	}

	lateTaskIDs := []int64{}
	for _, t := range tasks {
		if CalculateTaskStatus(t) == TaskStatusCompletedLate {
			lateTaskIDs = append(lateTaskIDs, t.ID)
		}
	}
	if len(lateTaskIDs) == 0 {
		return report, nil
	}

	assignments := []*TaskAssginee{}
	if err := s.In("task_id", lateTaskIDs).Find(&assignments); err != nil {
		return nil, err
	}

	countByUser := map[int64]int64{}
	userOrder := []int64{}
	for _, as := range assignments {
		if _, seen := countByUser[as.UserID]; !seen {
			userOrder = append(userOrder, as.UserID)
		}
		countByUser[as.UserID]++
		report.Total++
	}

	users, err := user.GetUsersByIDs(s, userOrder)
	if err != nil {
		return nil, err
	}
	for _, id := range userOrder {
		row := &LateCompletionReportRow{UserID: id, LateCount: countByUser[id]}
		if u, ok := users[id]; ok {
			row.Username = u.Username
			row.Name = u.GetName()
		}
		report.Rows = append(report.Rows, row)
	}

	// Most late completions first — this report is a performance signal.
	sort.Slice(report.Rows, func(i, j int) bool {
		if report.Rows[i].LateCount != report.Rows[j].LateCount {
			return report.Rows[i].LateCount > report.Rows[j].LateCount
		}
		return report.Rows[i].Name < report.Rows[j].Name
	})
	return report, nil
}
