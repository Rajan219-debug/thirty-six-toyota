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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumaDashboardStats(t *testing.T) {
	type allocationRow struct {
		ProjectID int64   `json:"project_id"`
		OpenCount int64   `json:"open_count"`
		Percent   float64 `json:"percent"`
	}
	type taskRow struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	type statsResp struct {
		TotalOpen      int64           `json:"total_open"`
		Overdue        int64           `json:"overdue"`
		Completed      int64           `json:"completed"`
		CompletionRate float64         `json:"completion_rate"`
		Allocation     []allocationRow `json:"allocation"`
		AttentionTasks []taskRow       `json:"attention_tasks"`
	}

	t.Run("aggregates are scoped, consistent and overdue-first", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/dashboard/stats", "", humaTokenFor(t, &testuser1), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var stats statsResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stats))

		// Fixture task 36 has a 2018 due date and is readable by user1.
		assert.GreaterOrEqual(t, stats.Overdue, int64(1))
		assert.Positive(t, stats.TotalOpen)
		assert.GreaterOrEqual(t, stats.CompletionRate, float64(0))
		assert.LessOrEqual(t, stats.CompletionRate, float64(100))

		var allocated int64
		for _, a := range stats.Allocation {
			allocated += a.OpenCount
		}
		assert.Equal(t, stats.TotalOpen, allocated, "every open task must appear in exactly one allocation row")

		require.NotEmpty(t, stats.AttentionTasks)
		assert.Equal(t, "overdue", stats.AttentionTasks[0].Status, "overdue tasks sort before pending confirmations")
		assert.LessOrEqual(t, len(stats.AttentionTasks), 15)
	})

	t.Run("requires authentication", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/dashboard/stats", "", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})
}
