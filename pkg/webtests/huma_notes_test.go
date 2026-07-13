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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHumaNotes(t *testing.T) {
	type noteResp struct {
		ID     int64  `json:"id"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Color  string `json:"color"`
		Pinned bool   `json:"pinned"`
	}
	type noteList struct {
		Items []noteResp `json:"items"`
		Total int64      `json:"total"`
	}

	t.Run("create then list round-trips for the owner", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		token := humaTokenFor(t, &testuser1)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/notes", `{"title":"Standup","body":"Discuss Q3","color":"mint","pinned":false}`, token, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created noteResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		assert.NotZero(t, created.ID)
		assert.Equal(t, "Standup", created.Title)

		rec = humaRequest(t, e, http.MethodGet, "/api/v2/notes", "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var list noteList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list.Items, 1)
		assert.Equal(t, created.ID, list.Items[0].ID)
		assert.Equal(t, "Standup", list.Items[0].Title)
	})

	t.Run("notes are private to their owner", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		owner := humaTokenFor(t, &testuser1)
		other := humaTokenFor(t, &testuser2)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/notes", `{"title":"Private","body":"secret","color":"amber","pinned":false}`, owner, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created noteResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

		// user2 does not see user1's notes in their own list
		rec = humaRequest(t, e, http.MethodGet, "/api/v2/notes", "", other, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var list noteList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Empty(t, list.Items)

		// user2 cannot read user1's note directly
		rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/notes/%d", created.ID), "", other, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

		// user2 cannot delete user1's note
		rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/notes/%d", created.ID), "", other, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("requires authentication", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/notes", "", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})
}
