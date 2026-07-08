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

package webtests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskAccountabilityV2 covers the overdue-closure confirmation workflow
// (blocked self-closure, inform-assigner, approve, reject, dispute) and the
// prioritized task view. Like TestTaskDuplicateV2 it drives the Echo+Huma
// stack directly because these are action sub-paths.
//
// Fixture topology: project 33 is owned by user6; team 1 (user1 + user2) has
// write access to it. user1 creates the task, so user1 is the assigner and
// user2 is an assignee with write access who is neither the task's creator,
// the project's fallback approver, nor a project admin — exactly the actor
// the accountability flow is supposed to block.
func TestTaskAccountabilityV2(t *testing.T) {
	type taskResp struct {
		ID                  int64  `json:"id"`
		Done                bool   `json:"done"`
		Status              string `json:"status"`
		CompletionClaimedAt string `json:"completion_claimed_at"`
		RejectionReason     string `json:"rejection_reason"`
	}

	const zeroTime = "0001-01-01T00:00:00Z"
	const pastDue = "2020-01-01T10:00:00Z"

	parseTask := func(t *testing.T, body []byte) (parsed taskResp) {
		require.NoError(t, json.Unmarshal(body, &parsed), "body: %s", string(body))
		return
	}

	// setupOverdueTask creates a task with a past due date in project 33 as
	// user1 and assigns user2 to it.
	setupOverdueTask := func(t *testing.T) (e *echo.Echo, assignerToken, assigneeToken string, taskID int64) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		assignerToken = humaTokenFor(t, &testuser1)
		assigneeToken = humaTokenFor(t, &testuser2)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/projects/33/tasks",
			`{"title":"overdue accountability task","due_date":"`+pastDue+`"}`, assignerToken, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		taskID = parseTask(t, rec.Body.Bytes()).ID
		require.NotZero(t, taskID)

		rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/assignees", taskID),
			`{"user_id":2}`, assignerToken, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		return
	}

	// claimClosure attempts to mark the task done as the assignee and asserts
	// the closure was blocked into a pending claim instead. The body carries
	// the assignee list because a task update replaces it wholesale — the
	// frontend always sends the full task, so omitting it here would silently
	// unassign user2 mid-test.
	claimClosure := func(t *testing.T, e *echo.Echo, assigneeToken string, taskID int64) {
		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/tasks/%d", taskID),
			`{"title":"overdue accountability task","due_date":"`+pastDue+`","done":true,"assignees":[{"id":2}]}`, assigneeToken, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		task := parseTask(t, rec.Body.Bytes())
		assert.False(t, task.Done, "an assignee must not be able to self-close an overdue task")
		assert.NotEqual(t, zeroTime, task.CompletionClaimedAt, "the blocked closure must be recorded as a pending claim")
	}

	getStatus := func(t *testing.T, e *echo.Echo, token string, taskID int64) string {
		rec := humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/tasks/%d", taskID), ``, token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		return parseTask(t, rec.Body.Bytes()).Status
	}

	t.Run("assignee cannot self-close an overdue task", func(t *testing.T) {
		e, assignerToken, assigneeToken, taskID := setupOverdueTask(t)
		claimClosure(t, e, assigneeToken, taskID)
		assert.Equal(t, "pending_confirmation", getStatus(t, e, assignerToken, taskID))
	})

	t.Run("inform assigner then approve closes the task late", func(t *testing.T) {
		e, assignerToken, assigneeToken, taskID := setupOverdueTask(t)
		claimClosure(t, e, assigneeToken, taskID)

		// POST actions default to 201 via the v2 Register wrapper, like tasks-duplicate.
		rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/inform-assigner", taskID), ``, assigneeToken, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())

		rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/approve-closure", taskID), ``, assignerToken, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		assert.True(t, parseTask(t, rec.Body.Bytes()).Done)

		assert.Equal(t, "completed_late", getStatus(t, e, assignerToken, taskID))
	})

	t.Run("reject keeps the task overdue and dispute escalates", func(t *testing.T) {
		e, assignerToken, assigneeToken, taskID := setupOverdueTask(t)
		claimClosure(t, e, assigneeToken, taskID)

		rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/reject-closure", taskID),
			`{"reason":"the customer has not signed off yet"}`, assignerToken, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		task := parseTask(t, rec.Body.Bytes())
		assert.False(t, task.Done)
		assert.Equal(t, "the customer has not signed off yet", task.RejectionReason)

		assert.Equal(t, "overdue", getStatus(t, e, assignerToken, taskID))

		rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dispute-rejection", taskID), ``, assigneeToken, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("the creator can close an overdue task directly", func(t *testing.T) {
		e, assignerToken, _, taskID := setupOverdueTask(t)

		rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/tasks/%d", taskID),
			`{"title":"overdue accountability task","due_date":"`+pastDue+`","done":true}`, assignerToken, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.True(t, parseTask(t, rec.Body.Bytes()).Done, "an approver closing directly must not be blocked")

		assert.Equal(t, "completed_late", getStatus(t, e, assignerToken, taskID))
	})

	t.Run("a non-assignee cannot inform the assigner", func(t *testing.T) {
		e, assignerToken, assigneeToken, taskID := setupOverdueTask(t)
		claimClosure(t, e, assigneeToken, taskID)

		rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/inform-assigner", taskID), ``, assignerToken, "")
		require.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("informing without a pending claim conflicts", func(t *testing.T) {
		e, _, assigneeToken, taskID := setupOverdueTask(t)

		rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/inform-assigner", taskID), ``, assigneeToken, "")
		require.Equal(t, http.StatusConflict, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("an assignee cannot approve their own claim", func(t *testing.T) {
		e, _, assigneeToken, taskID := setupOverdueTask(t)
		claimClosure(t, e, assigneeToken, taskID)

		rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/approve-closure", taskID), ``, assigneeToken, "")
		require.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("approving without a pending claim conflicts", func(t *testing.T) {
		e, assignerToken, _, taskID := setupOverdueTask(t)

		rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/approve-closure", taskID), ``, assignerToken, "")
		require.Equal(t, http.StatusConflict, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("disputing without a rejection conflicts", func(t *testing.T) {
		e, _, assigneeToken, taskID := setupOverdueTask(t)
		claimClosure(t, e, assigneeToken, taskID)

		rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/tasks/%d/dispute-rejection", taskID), ``, assigneeToken, "")
		require.Equal(t, http.StatusConflict, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("prioritized buckets the caller's open tasks", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		// user2 is assigned to fixture tasks 30 (no due date), 35 (no due
		// date), and 36 (due 2018, overdue). With urgency and importance
		// unset, 36 scores 0.4*5 = 2.0 (do_soon); 30 and 35 score 0 (do_later).
		token := humaTokenFor(t, &testuser2)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/tasks/prioritized", ``, token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		var resp struct {
			DoNow   []taskResp `json:"do_now"`
			DoSoon  []taskResp `json:"do_soon"`
			DoLater []taskResp `json:"do_later"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

		ids := func(tasks []taskResp) (out []int64) {
			for _, task := range tasks {
				out = append(out, task.ID)
			}
			return
		}
		assert.Empty(t, resp.DoNow)
		assert.Contains(t, ids(resp.DoSoon), int64(36))
		assert.Contains(t, ids(resp.DoLater), int64(30))
		assert.Contains(t, ids(resp.DoLater), int64(35))
	})
}
