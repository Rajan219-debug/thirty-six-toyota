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

func TestHumaConversations(t *testing.T) {
	type conversationResp struct {
		ID       int64 `json:"id"`
		IsGroup  bool  `json:"is_group"`
		IsUnread bool  `json:"is_unread"`
	}
	type conversationList struct {
		Items []conversationResp `json:"items"`
		Total int64              `json:"total"`
	}
	type messageResp struct {
		ID   int64  `json:"id"`
		Text string `json:"text"`
	}
	type messageList struct {
		Items []messageResp `json:"items"`
		Total int64         `json:"total"`
	}

	t.Run("both members see the conversation, non-members do not", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		user1 := humaTokenFor(t, &testuser1)
		user2 := humaTokenFor(t, &testuser2)
		outsider := humaTokenFor(t, &testuser10)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/conversations", `{"member_ids":[2]}`, user1, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created conversationResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		assert.NotZero(t, created.ID)
		assert.False(t, created.IsGroup, "a two-person chat is 1:1, not a group")

		// user2 (a member) sees it
		rec = humaRequest(t, e, http.MethodGet, "/api/v2/conversations", "", user2, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var list conversationList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list.Items, 1)
		assert.Equal(t, created.ID, list.Items[0].ID)

		// outsider (not a member) does not
		rec = humaRequest(t, e, http.MethodGet, "/api/v2/conversations", "", outsider, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var outsiderList conversationList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &outsiderList))
		assert.Empty(t, outsiderList.Items)

		// outsider cannot read it directly
		rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/conversations/%d", created.ID), "", outsider, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("members can send and read messages; outsiders cannot", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		user1 := humaTokenFor(t, &testuser1)
		user2 := humaTokenFor(t, &testuser2)
		outsider := humaTokenFor(t, &testuser10)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/conversations", `{"member_ids":[2]}`, user1, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created conversationResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

		// user1 sends a message
		rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/conversations/%d/messages", created.ID), `{"text":"Morning huddle at 9"}`, user1, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		// user2 (a member) reads it
		rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/conversations/%d/messages", created.ID), "", user2, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var msgs messageList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &msgs))
		require.Len(t, msgs.Items, 1)
		assert.Equal(t, "Morning huddle at 9", msgs.Items[0].Text)

		// outsider cannot read messages
		rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/conversations/%d/messages", created.ID), "", outsider, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

		// outsider cannot send a message
		rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/conversations/%d/messages", created.ID), `{"text":"let me in"}`, outsider, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("unread flips true for the recipient, never for the sender, and clears on read", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		user1 := humaTokenFor(t, &testuser1)
		user2 := humaTokenFor(t, &testuser2)

		findConversation := func(t *testing.T, token string, id int64) conversationResp {
			t.Helper()
			rec := humaRequest(t, e, http.MethodGet, "/api/v2/conversations", "", token, "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var list conversationList
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
			for _, c := range list.Items {
				if c.ID == id {
					return c
				}
			}
			require.Fail(t, "conversation not found in list", "id=%d", id)
			return conversationResp{}
		}

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/conversations", `{"member_ids":[2]}`, user1, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var created conversationResp
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

		// user1 sends the first message.
		rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/conversations/%d/messages", created.ID), `{"text":"anyone around?"}`, user1, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		// The sender never sees their own message as unread to themselves.
		assert.False(t, findConversation(t, user1, created.ID).IsUnread, "sender should not see their own message as unread")

		// The recipient hasn't read it yet.
		assert.True(t, findConversation(t, user2, created.ID).IsUnread, "recipient should see an unread message")

		// Fetching messages is how user2 "opens" the conversation.
		rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/conversations/%d/messages", created.ID), "", user2, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.False(t, findConversation(t, user2, created.ID).IsUnread, "reading the messages should clear the unread flag")
	})

	t.Run("a conversation needs at least one other member", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)
		user1 := humaTokenFor(t, &testuser1)

		rec := humaRequest(t, e, http.MethodPost, "/api/v2/conversations", `{"member_ids":[]}`, user1, "")
		assert.GreaterOrEqual(t, rec.Code, 400, rec.Body.String())
	})

	t.Run("requires authentication", func(t *testing.T) {
		e, err := setupTestEnv()
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, "/api/v2/conversations", "", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})
}
