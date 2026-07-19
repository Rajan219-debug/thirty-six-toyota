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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConversationIsUnread(t *testing.T) {
	now := time.Now()

	t.Run("no messages yet is never unread", func(t *testing.T) {
		assert.False(t, conversationIsUnread(nil, time.Time{}))
		assert.False(t, conversationIsUnread(nil, now))
	})

	t.Run("never read (zero time) with a message is unread", func(t *testing.T) {
		msg := &ChatMessage{Created: now}
		assert.True(t, conversationIsUnread(msg, time.Time{}))
	})

	t.Run("message sent after the last read is unread", func(t *testing.T) {
		msg := &ChatMessage{Created: now}
		lastReadAt := now.Add(-time.Minute)
		assert.True(t, conversationIsUnread(msg, lastReadAt))
	})

	t.Run("message sent before the last read is not unread", func(t *testing.T) {
		msg := &ChatMessage{Created: now.Add(-time.Minute)}
		lastReadAt := now
		assert.False(t, conversationIsUnread(msg, lastReadAt))
	})

	t.Run("read at the exact same instant is not unread", func(t *testing.T) {
		msg := &ChatMessage{Created: now}
		assert.False(t, conversationIsUnread(msg, now))
	})
}
