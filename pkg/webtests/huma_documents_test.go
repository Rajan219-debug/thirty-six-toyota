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

func TestHumaDocuments(t *testing.T) {
	type folderResp struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	type folderList struct {
		Items []folderResp `json:"items"`
		Total int64        `json:"total"`
	}

	t.Run("folders are a shared company drive: any user can see them", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		owner := humaTokenFor(t, &testuser1)
		other := humaTokenFor(t, &testuser2)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/folders", `{"title":"Recall Notices"}`, owner, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created folderResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		assert.NotZero(t, created.ID)

		// A different user sees the same folder — the drive is company-wide.
		rec = humaRequest(t, e, http.MethodGet, "/api/v2/folders", "", other, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var list folderList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list.Items, 1)
		assert.Equal(t, created.ID, list.Items[0].ID)
	})

	t.Run("only the folder owner may delete it", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		owner := humaTokenFor(t, &testuser1)
		other := humaTokenFor(t, &testuser2)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/folders", `{"title":"Owner only"}`, owner, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created folderResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

		// A non-owner cannot delete it.
		rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/folders/%d", created.ID), "", other, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

		// The owner can.
		rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/folders/%d", created.ID), "", owner, "")
		assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	})

	t.Run("a fresh folder has no documents", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		owner := humaTokenFor(t, &testuser1)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/folders", `{"title":"Empty"}`, owner, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created folderResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

		rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/folders/%d/documents", created.ID), "", owner, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var docs struct {
			Total int64 `json:"total"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &docs))
		assert.Equal(t, int64(0), docs.Total)
	})

	t.Run("requires authentication", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/folders", "", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})
}
