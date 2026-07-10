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
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixtures (pkg/db/fixtures/time_entries.yml): entry 1 = user1/task1 (1h),
// 2 = user1/project1 (1h), 3 = user3/project3 (1h), 4 = user1/task1 running
// (no end). user1 (testuser1) can read all four — see TestHumaTimeEntry ReadAll.

type timesheetRow struct {
	UserID       int64 `json:"user_id"`
	EntryCount   int64 `json:"entry_count"`
	TotalSeconds int64 `json:"total_seconds"`
}
type timesheetResp struct {
	TotalSeconds int64          `json:"total_seconds"`
	Rows         []timesheetRow `json:"rows"`
}

func TestHumaReports(t *testing.T) {
	t.Run("timesheet aggregates completed time per employee and excludes running timers", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/reports/timesheet", "", humaTokenFor(t, &testuser1), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var report timesheetResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))

		byUser := map[int64]timesheetRow{}
		for _, r := range report.Rows {
			byUser[r.UserID] = r
		}
		// user1: entries 1 + 2 = 7200s over 2 entries; the running timer (4) is excluded.
		assert.Equal(t, int64(7200), byUser[1].TotalSeconds)
		assert.Equal(t, int64(2), byUser[1].EntryCount)
		// user3: entry 3 = 3600s.
		assert.Equal(t, int64(3600), byUser[3].TotalSeconds)
		assert.Equal(t, int64(10800), report.TotalSeconds)
	})

	t.Run("project filter scopes the report", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/reports/timesheet?project_id=1", "", humaTokenFor(t, &testuser1), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var report timesheetResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
		// Project 3's entry (user3) must not appear when scoped to project 1.
		for _, r := range report.Rows {
			assert.Equal(t, int64(1), r.UserID, "only user1 has time on project 1")
		}
		assert.Greater(t, report.TotalSeconds, int64(0))
	})

	t.Run("timesheet CSV download", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/reports/timesheet.csv", "", humaTokenFor(t, &testuser1), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
		assert.True(t, strings.HasPrefix(rec.Body.String(), "user_id,username,name,entries"), rec.Body.String())
	})

	t.Run("late-completion report serves", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/reports/late-completions", "", humaTokenFor(t, &testuser1), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var report struct {
			Total int64 `json:"total"`
			Rows  []struct {
				UserID    int64 `json:"user_id"`
				LateCount int64 `json:"late_count"`
			} `json:"rows"`
			Weeks []struct {
				WeekStart string `json:"week_start"`
				Count     int64  `json:"count"`
			} `json:"weeks"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
		// No fixture task is completed_late, so the tally is empty — but the shape must hold.
		assert.Equal(t, int64(0), report.Total)
		assert.Empty(t, report.Rows)
		assert.Empty(t, report.Weeks)
	})

	t.Run("department estimates compare estimated vs tracked effort per department", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		token := humaTokenFor(t, &testuser1)

		// Estimate the done fixture task 2 (project 1) at 2h via the task API.
		// (PUT, not PATCH: AutoPatch's GET→PUT round trip trips over the
		// subscription sub-object's schema — a pre-existing upstream quirk.)
		rec := humaRequest(t, e, http.MethodPut, "/api/v2/tasks/2", `{"title":"task #2 done","done":true,"estimated_seconds":7200}`, token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		// ...and track 1h of actual work on it.
		rec = humaRequest(t, e, http.MethodPost, "/api/v2/time-entries",
			`{"task_id":2,"start_time":"2020-01-01T09:00:00Z","end_time":"2020-01-01T10:00:00Z"}`, token, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = humaRequest(t, e, http.MethodGet, "/api/v2/reports/department-estimates", "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var report struct {
			TotalEstimated int64 `json:"total_estimated"`
			TotalActual    int64 `json:"total_actual"`
			Rows           []struct {
				ProjectID        int64 `json:"project_id"`
				TaskCount        int64 `json:"task_count"`
				EstimatedSeconds int64 `json:"estimated_seconds"`
				ActualSeconds    int64 `json:"actual_seconds"`
			} `json:"rows"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
		// Only project 1 has a done task carrying an estimate or tracked time;
		// the fixture entry on the not-done task 1 must not leak in, and
		// departments with nothing to compare are dropped.
		require.Len(t, report.Rows, 1, rec.Body.String())
		assert.Equal(t, int64(1), report.Rows[0].ProjectID)
		assert.Equal(t, int64(7200), report.Rows[0].EstimatedSeconds)
		assert.Equal(t, int64(3600), report.Rows[0].ActualSeconds)
		assert.Equal(t, int64(7200), report.TotalEstimated)
		assert.Equal(t, int64(3600), report.TotalActual)
	})

	t.Run("department estimates CSV download", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/reports/department-estimates.csv", "", humaTokenFor(t, &testuser1), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
		assert.True(t, strings.HasPrefix(rec.Body.String(), "project_id,department,tasks"), rec.Body.String())
	})
}
