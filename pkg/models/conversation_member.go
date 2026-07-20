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
	"time"

	"xorm.io/xorm"
)

// ConversationMember is a join row between a Conversation and a User.
// Internal only — not exposed as its own CRUD resource, see Conversation.Create.
type ConversationMember struct {
	ID             int64 `xorm:"bigint autoincr not null unique pk"`
	ConversationID int64 `xorm:"bigint not null index"`
	UserID         int64 `xorm:"bigint not null index"`
	// When this member last fetched the conversation's messages. Zero value
	// means never — the whole conversation is unread. Compared against the
	// last message's Created time to compute Conversation.IsUnread; no
	// separate per-message unread table needed since membership here is
	// fixed and small (see Conversation's ponytail note).
	LastReadAt time.Time `xorm:"datetime null"`
	Created    time.Time `xorm:"created not null"`
}

// TableName returns the table name for conversation members
func (*ConversationMember) TableName() string {
	return "conversation_members"
}

// isConversationMember reports whether userID is a member of conversationID.
func isConversationMember(s *xorm.Session, conversationID, userID int64) (bool, error) {
	return s.Where("conversation_id = ? AND user_id = ?", conversationID, userID).Exist(&ConversationMember{})
}

// markConversationRead stamps the given user's membership row with the
// current time, so Conversation.IsUnread goes false until the next message.
func markConversationRead(s *xorm.Session, conversationID, userID int64) error {
	_, err := s.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Cols("last_read_at").
		Update(&ConversationMember{LastReadAt: time.Now()})
	return err
}

// getLastReadAt returns when userID last read conversationID, or the zero
// time if they never have (or aren't a member — callers already gate that).
func getLastReadAt(s *xorm.Session, conversationID, userID int64) (time.Time, error) {
	var member ConversationMember
	has, err := s.Where("conversation_id = ? AND user_id = ?", conversationID, userID).Get(&member)
	if err != nil || !has {
		return time.Time{}, err
	}
	return member.LastReadAt, nil
}
