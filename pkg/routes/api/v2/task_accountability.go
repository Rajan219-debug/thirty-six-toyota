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
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/danielgtaylor/huma/v2"
	"xorm.io/xorm"
)

// RegisterTaskAccountabilityRoutes wires the overdue-closure confirmation
// workflow and the prioritized ("Do Now / Do Soon / Do Later") task view onto
// the Huma API. These are custom actions, not plain CRUD, so each handler
// checks permissions explicitly and manages its own session — see the
// api-v2-routes skill's "Non-CRUDable / custom routes" section.
func RegisterTaskAccountabilityRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "tasks-inform-assigner",
		Summary:     "Inform the approver about a pending overdue closure",
		Description: "Called by an assignee after their attempt to mark an overdue task done was blocked. Notifies the task's creator (and the project's fallback approver, if set) that a closure approval is needed. Requires the task to already have a pending closure claim — set automatically when an assignee tries to close an overdue task.",
		Method:      http.MethodPost,
		Path:        "/tasks/{id}/inform-assigner",
		Tags:        tags,
	}, tasksInformAssigner)

	Register(api, huma.Operation{
		OperationID: "tasks-approve-closure",
		Summary:     "Approve an overdue task's closure",
		Description: "Only the task's creator, the project's fallback approver, or a project admin may call this. Marks the task done and records who approved it; the task's status becomes completed_late.",
		Method:      http.MethodPost,
		Path:        "/tasks/{id}/approve-closure",
		Tags:        tags,
	}, tasksApproveClosure)

	Register(api, huma.Operation{
		OperationID: "tasks-reject-closure",
		Summary:     "Reject an overdue task's closure",
		Description: "Only the task's creator, the project's fallback approver, or a project admin may call this. The task stays overdue and undone; the assignees are notified with the given reason.",
		Method:      http.MethodPost,
		Path:        "/tasks/{id}/reject-closure",
		Tags:        tags,
	}, tasksRejectClosure)

	Register(api, huma.Operation{
		OperationID: "tasks-dispute-rejection",
		Summary:     "Dispute a closure rejection",
		Description: "Called by an assignee after a closure rejection they disagree with. Escalates to the project owner as a notification; does not change the task's state.",
		Method:      http.MethodPost,
		Path:        "/tasks/{id}/dispute-rejection",
		Tags:        tags,
	}, tasksDisputeRejection)

	Register(api, huma.Operation{
		OperationID: "tasks-prioritized",
		Summary:     "List the authenticated user's open tasks bucketed by priority",
		Description: "Returns every not-done task assigned to the caller, split into do_now, do_soon, and do_later based on the computed priority score (urgency, importance, and deadline proximity). No pagination; intended for a personal \"what should I work on\" view.",
		Method:      http.MethodGet,
		Path:        "/tasks/prioritized",
		Tags:        tags,
	}, tasksPrioritized)
}

func init() { AddRouteRegistrar(RegisterTaskAccountabilityRoutes) }

// resolveApprovers returns the task creator plus the project's fallback
// approver (if set), so the "informed" notification reaches whoever can
// actually approve the closure even if one of them is unavailable.
func resolveApprovers(s *xorm.Session, t *models.Task) ([]*user.User, error) {
	var approvers []*user.User

	if t.CreatedByID != 0 {
		creator, err := user.GetUserByID(s, t.CreatedByID)
		if err != nil {
			return nil, err
		}
		approvers = append(approvers, creator)
	}

	p, err := models.GetProjectSimpleByID(s, t.ProjectID)
	if err != nil {
		return nil, err
	}
	if p.FallbackApproverID != 0 && p.FallbackApproverID != t.CreatedByID {
		fallback, err := user.GetUserByID(s, p.FallbackApproverID)
		if err != nil {
			return nil, err
		}
		approvers = append(approvers, fallback)
	}

	return approvers, nil
}

// getTaskAssigneeUsers returns the users assigned to a task.
func getTaskAssigneeUsers(s *xorm.Session, taskID int64) ([]*user.User, error) {
	var rows []*models.TaskAssigneeWithUser
	err := s.Table("task_assignees").
		Select("task_id, users.*").
		Where("task_assignees.task_id = ?", taskID).
		Join("INNER", "users", "task_assignees.user_id = users.id").
		Find(&rows)
	if err != nil {
		return nil, err
	}
	assignees := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		u := row.User
		assignees = append(assignees, &u)
	}
	return assignees, nil
}

func tasksInformAssigner(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*singleBody[models.Task], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewBoundSession(a.GetID())
	defer s.Close()

	t := &models.Task{ID: in.ID}
	can, _, err := t.CanRead(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}

	isAssignee, err := models.TaskHasAssignee(s, t.ID, a.GetID())
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !isAssignee {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("only an assignee can inform the approver about a pending closure")
	}

	if t.CompletionClaimedAt.IsZero() {
		_ = s.Rollback()
		return nil, huma.Error409Conflict("this task has no pending overdue-closure claim")
	}

	doer, err := models.GetUserOrLinkShareUser(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	approvers, err := resolveApprovers(s, t)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	events.DispatchOnCommit(s, &models.TaskClosureRequestedEvent{
		Task:      t,
		Doer:      doer,
		Approvers: approvers,
	})

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)

	return &singleBody[models.Task]{Body: t}, nil
}

func tasksApproveClosure(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*singleBody[models.Task], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewBoundSession(a.GetID())
	defer s.Close()

	t := &models.Task{ID: in.ID}
	can, _, err := t.CanRead(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}

	canApprove, err := models.UserCanApproveOverdueClosure(s, t, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !canApprove {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("only the task's creator, the project's fallback approver, or a project admin may approve an overdue closure")
	}

	if t.CompletionClaimedAt.IsZero() {
		_ = s.Rollback()
		return nil, huma.Error409Conflict("this task has no pending overdue-closure claim")
	}

	doer, err := models.GetUserOrLinkShareUser(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	assignees, err := getTaskAssigneeUsers(s, t.ID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	t.Done = true
	t.DoneAt = time.Now()
	t.ConfirmedByID = a.GetID()
	t.CompletionClaimedAt = time.Time{}
	t.RejectionReason = ""
	if _, err := s.ID(t.ID).Cols("done", "done_at", "confirmed_by_id", "completion_claimed_at", "rejection_reason").Update(t); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	events.DispatchOnCommit(s, &models.TaskClosureApprovedEvent{
		Task:      t,
		Doer:      doer,
		Assignees: assignees,
	})

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)

	return &singleBody[models.Task]{Body: t}, nil
}

type rejectClosureBody struct {
	// Why the closure is being rejected. Shown to the assignees.
	Reason string `json:"reason" minLength:"1" doc:"Why the closure is being rejected. Shown to the assignees."`
}

func tasksRejectClosure(ctx context.Context, in *struct {
	ID   int64 `path:"id"`
	Body rejectClosureBody
}) (*singleBody[models.Task], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewBoundSession(a.GetID())
	defer s.Close()

	t := &models.Task{ID: in.ID}
	can, _, err := t.CanRead(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}

	canApprove, err := models.UserCanApproveOverdueClosure(s, t, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !canApprove {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("only the task's creator, the project's fallback approver, or a project admin may reject an overdue closure")
	}

	if t.CompletionClaimedAt.IsZero() {
		_ = s.Rollback()
		return nil, huma.Error409Conflict("this task has no pending overdue-closure claim")
	}

	doer, err := models.GetUserOrLinkShareUser(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	assignees, err := getTaskAssigneeUsers(s, t.ID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	t.CompletionClaimedAt = time.Time{}
	t.RejectionReason = in.Body.Reason
	if _, err := s.ID(t.ID).Cols("completion_claimed_at", "rejection_reason").Update(t); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	events.DispatchOnCommit(s, &models.TaskClosureRejectedEvent{
		Task:      t,
		Doer:      doer,
		Reason:    in.Body.Reason,
		Assignees: assignees,
	})

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)

	return &singleBody[models.Task]{Body: t}, nil
}

func tasksDisputeRejection(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*singleBody[models.Task], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewBoundSession(a.GetID())
	defer s.Close()

	t := &models.Task{ID: in.ID}
	can, _, err := t.CanRead(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}

	isAssignee, err := models.TaskHasAssignee(s, t.ID, a.GetID())
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !isAssignee {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("only an assignee can dispute a closure rejection")
	}

	if t.RejectionReason == "" {
		_ = s.Rollback()
		return nil, huma.Error409Conflict("this task has no rejection to dispute")
	}

	doer, err := models.GetUserOrLinkShareUser(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	p, err := models.GetProjectSimpleByID(s, t.ProjectID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	admin, err := user.GetUserByID(s, p.OwnerID)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	events.DispatchOnCommit(s, &models.TaskClosureDisputedEvent{
		Task:  t,
		Doer:  doer,
		Admin: admin,
	})

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)

	return &singleBody[models.Task]{Body: t}, nil
}

type prioritizedTasksBody struct {
	Body struct {
		DoNow   []*models.Task `json:"do_now"`
		DoSoon  []*models.Task `json:"do_soon"`
		DoLater []*models.Task `json:"do_later"`
	}
}

func tasksPrioritized(ctx context.Context, _ *struct{}) (*prioritizedTasksBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewBoundSession(a.GetID())
	defer s.Close()

	var assigneeRows []*models.TaskAssginee
	if err := s.Where("user_id = ?", a.GetID()).Find(&assigneeRows); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	taskIDs := make([]int64, 0, len(assigneeRows))
	for _, row := range assigneeRows {
		taskIDs = append(taskIDs, row.TaskID)
	}

	var tasks []*models.Task
	if len(taskIDs) > 0 {
		if err := s.In("id", taskIDs).Where("done = ?", false).Find(&tasks); err != nil {
			_ = s.Rollback()
			return nil, translateDomainError(err)
		}
	}

	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	body := &prioritizedTasksBody{}
	body.Body.DoNow = []*models.Task{}
	body.Body.DoSoon = []*models.Task{}
	body.Body.DoLater = []*models.Task{}
	for _, t := range tasks {
		t.Status = models.CalculateTaskStatus(t)
		t.PriorityScore, t.PriorityBucket = models.CalculatePriorityScore(t.Urgency, t.Importance, t.DueDate)
		switch t.PriorityBucket {
		case models.PriorityBucketDoNow:
			body.Body.DoNow = append(body.Body.DoNow, t)
		case models.PriorityBucketDoSoon:
			body.Body.DoSoon = append(body.Body.DoSoon, t)
		default:
			body.Body.DoLater = append(body.Body.DoLater, t)
		}
	}

	return body, nil
}
