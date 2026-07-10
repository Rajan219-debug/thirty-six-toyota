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

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// DashboardEfficiencyRow is one employee's completion efficiency: what share of
// the tasks they closed were closed on time (not completed_late).
type DashboardEfficiencyRow struct {
	UserID     int64   `json:"user_id" doc:"The employee this row scores."`
	Username   string  `json:"username" doc:"The employee's username."`
	Name       string  `json:"name" doc:"The employee's full name (falls back to the username)."`
	Completed  int64   `json:"completed" doc:"Total tasks the employee closed (on time + late)."`
	OnTime     int64   `json:"on_time" doc:"How many of those closed on time."`
	Efficiency float64 `json:"efficiency" doc:"On-time percentage (on_time / completed * 100)."`
}

// DashboardAllocationRow is the share of currently-open tasks that sit in one
// project (= department, in the Thirty Six Toyota model).
type DashboardAllocationRow struct {
	ProjectID int64   `json:"project_id" doc:"The project / department."`
	Name      string  `json:"name" doc:"The project's title."`
	OpenCount int64   `json:"open_count" doc:"Open tasks currently in this project."`
	Percent   float64 `json:"percent" doc:"Share of all open tasks that are in this project."`
}

// DashboardTaskRow is one attention-worthy task (overdue or awaiting
// confirmation) shown in the dashboard's "Overdue & Pending" table.
type DashboardTaskRow struct {
	ID       int64  `json:"id" doc:"The task id."`
	Title    string `json:"title" doc:"The task title."`
	Project  string `json:"project" doc:"The task's project / department name."`
	Assignee string `json:"assignee" doc:"Assigned employee name(s), comma-joined."`
	Status   string `json:"status" doc:"Computed lifecycle status (overdue / pending_confirmation)."`
}

// DashboardStats is the manager dashboard's headline numbers, all scoped to the
// projects the caller can read (so a team lead sees their department, the owner
// sees the whole dealership — same permission-driven scoping as the reports).
type DashboardStats struct {
	TotalOpen           int64                     `json:"total_open" doc:"Open tasks (pending, ongoing, overdue or awaiting confirmation)."`
	Overdue             int64                     `json:"overdue" doc:"Open tasks past their due date."`
	PendingConfirmation int64                     `json:"pending_confirmation" doc:"Tasks whose overdue closure is awaiting approval."`
	Completed           int64                     `json:"completed" doc:"Closed tasks (on time or late)."`
	CompletionRate      float64                   `json:"completion_rate" doc:"Closed / total tasks, as a percentage."`
	Leaderboard         []*DashboardEfficiencyRow `json:"leaderboard" doc:"Per-employee on-time efficiency, most efficient first."`
	Allocation          []*DashboardAllocationRow `json:"allocation" doc:"Open-task share per project / department, largest first."`
	AttentionTasks      []*DashboardTaskRow       `json:"attention_tasks" doc:"Overdue and awaiting-confirmation tasks needing a manager's action."`
}

// GetDashboardStats computes the manager dashboard aggregates from every task in
// an accessible project. Status is derived through CalculateTaskStatus so these
// numbers can never drift from the task lifecycle rules used elsewhere.
func GetDashboardStats(s *xorm.Session, a web.Auth) (*DashboardStats, error) {
	stats := &DashboardStats{
		Leaderboard:    []*DashboardEfficiencyRow{},
		Allocation:     []*DashboardAllocationRow{},
		AttentionTasks: []*DashboardTaskRow{},
	}

	if _, isShare := a.(*LinkSharing); isShare {
		return stats, nil
	}

	var tasks []*Task
	if err := s.Where(accessibleProjectIDsSubquery(a, "project_id")).Find(&tasks); err != nil {
		return nil, err
	}

	var total int64
	openByProject := map[int64]int64{}
	projectOrder := []int64{}
	// isLate maps a done task's id to whether it closed late, for the efficiency tally.
	doneTaskLate := map[int64]bool{}
	// Overdue-first, then pending-confirmation — the tasks a manager must act on.
	var attention []*Task

	for _, t := range tasks {
		total++
		switch CalculateTaskStatus(t) {
		case TaskStatusOverdue:
			stats.Overdue++
			stats.TotalOpen++
			addOpen(openByProject, &projectOrder, t.ProjectID)
			attention = append(attention, t)
		case TaskStatusPendingConfirmation:
			stats.PendingConfirmation++
			stats.TotalOpen++
			addOpen(openByProject, &projectOrder, t.ProjectID)
			attention = append(attention, t)
		case TaskStatusPending, TaskStatusOngoing:
			stats.TotalOpen++
			addOpen(openByProject, &projectOrder, t.ProjectID)
		case TaskStatusCompleted:
			stats.Completed++
			doneTaskLate[t.ID] = false
		case TaskStatusCompletedLate:
			stats.Completed++
			doneTaskLate[t.ID] = true
		}
	}

	if total > 0 {
		stats.CompletionRate = round1(float64(stats.Completed) / float64(total) * 100)
	}

	if err := fillAllocation(s, stats, openByProject, projectOrder); err != nil {
		return nil, err
	}
	if err := fillLeaderboard(s, stats, doneTaskLate); err != nil {
		return nil, err
	}
	if err := fillAttention(s, stats, attention); err != nil {
		return nil, err
	}

	return stats, nil
}

// fillAttention resolves the overdue / pending-confirmation tasks into display
// rows (project title + assignee names), overdue first, capped for the panel.
func fillAttention(s *xorm.Session, stats *DashboardStats, attention []*Task) error {
	if len(attention) == 0 {
		return nil
	}
	sort.SliceStable(attention, func(i, j int) bool {
		// overdue before pending_confirmation
		return CalculateTaskStatus(attention[i]) == TaskStatusOverdue &&
			CalculateTaskStatus(attention[j]) != TaskStatusOverdue
	})
	if len(attention) > 15 {
		attention = attention[:15]
	}

	projectIDs := map[int64]bool{}
	taskIDs := make([]int64, 0, len(attention))
	for _, t := range attention {
		projectIDs[t.ProjectID] = true
		taskIDs = append(taskIDs, t.ID)
	}
	pIDs := make([]int64, 0, len(projectIDs))
	for id := range projectIDs {
		pIDs = append(pIDs, id)
	}
	projects, err := GetProjectsMapByIDs(s, pIDs)
	if err != nil {
		return err
	}

	// task -> assignee names
	assignments := []*TaskAssginee{}
	if err := s.In("task_id", taskIDs).Find(&assignments); err != nil {
		return err
	}
	userIDs := []int64{}
	for _, as := range assignments {
		userIDs = append(userIDs, as.UserID)
	}
	users, err := user.GetUsersByIDs(s, userIDs)
	if err != nil {
		return err
	}
	assigneeNames := map[int64][]string{}
	for _, as := range assignments {
		if u, ok := users[as.UserID]; ok {
			assigneeNames[as.TaskID] = append(assigneeNames[as.TaskID], u.GetName())
		}
	}

	for _, t := range attention {
		row := &DashboardTaskRow{ID: t.ID, Title: t.Title, Status: CalculateTaskStatus(t)}
		if p, ok := projects[t.ProjectID]; ok {
			row.Project = p.Title
		}
		if names := assigneeNames[t.ID]; len(names) > 0 {
			row.Assignee = joinNames(names)
		}
		stats.AttentionTasks = append(stats.AttentionTasks, row)
	}
	return nil
}

func joinNames(names []string) string {
	out := names[0]
	for _, n := range names[1:] {
		out += ", " + n
	}
	return out
}

func addOpen(openByProject map[int64]int64, order *[]int64, projectID int64) {
	if _, seen := openByProject[projectID]; !seen {
		*order = append(*order, projectID)
	}
	openByProject[projectID]++
}

// fillAllocation turns the per-project open counts into named, percentage rows.
func fillAllocation(s *xorm.Session, stats *DashboardStats, openByProject map[int64]int64, order []int64) error {
	if len(order) == 0 {
		return nil
	}
	projects, err := GetProjectsMapByIDs(s, order)
	if err != nil {
		return err
	}
	for _, id := range order {
		row := &DashboardAllocationRow{ProjectID: id, OpenCount: openByProject[id]}
		if p, ok := projects[id]; ok {
			row.Name = p.Title
		}
		if stats.TotalOpen > 0 {
			row.Percent = round1(float64(openByProject[id]) / float64(stats.TotalOpen) * 100)
		}
		stats.Allocation = append(stats.Allocation, row)
	}
	sort.Slice(stats.Allocation, func(i, j int) bool {
		return stats.Allocation[i].OpenCount > stats.Allocation[j].OpenCount
	})
	return nil
}

// fillLeaderboard tallies each assignee's on-time efficiency over the closed tasks.
func fillLeaderboard(s *xorm.Session, stats *DashboardStats, doneTaskLate map[int64]bool) error {
	if len(doneTaskLate) == 0 {
		return nil
	}
	doneIDs := make([]int64, 0, len(doneTaskLate))
	for id := range doneTaskLate {
		doneIDs = append(doneIDs, id)
	}

	assignments := []*TaskAssginee{}
	if err := s.In("task_id", doneIDs).Find(&assignments); err != nil {
		return err
	}

	rowByUser := map[int64]*DashboardEfficiencyRow{}
	userOrder := []int64{}
	for _, as := range assignments {
		row := rowByUser[as.UserID]
		if row == nil {
			row = &DashboardEfficiencyRow{UserID: as.UserID}
			rowByUser[as.UserID] = row
			userOrder = append(userOrder, as.UserID)
		}
		row.Completed++
		if !doneTaskLate[as.TaskID] {
			row.OnTime++
		}
	}

	users, err := user.GetUsersByIDs(s, userOrder)
	if err != nil {
		return err
	}
	for _, id := range userOrder {
		row := rowByUser[id]
		if u, ok := users[id]; ok {
			row.Username = u.Username
			row.Name = u.GetName()
		}
		if row.Completed > 0 {
			row.Efficiency = round1(float64(row.OnTime) / float64(row.Completed) * 100)
		}
		stats.Leaderboard = append(stats.Leaderboard, row)
	}
	sort.Slice(stats.Leaderboard, func(i, j int) bool {
		if stats.Leaderboard[i].Efficiency != stats.Leaderboard[j].Efficiency {
			return stats.Leaderboard[i].Efficiency > stats.Leaderboard[j].Efficiency
		}
		return stats.Leaderboard[i].Name < stats.Leaderboard[j].Name
	})
	return nil
}

// round1 rounds to one decimal place — enough precision for a percentage badge.
func round1(f float64) float64 {
	return float64(int64(f*10+0.5)) / 10
}
