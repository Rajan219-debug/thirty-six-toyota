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

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

// ChatMessage is a single chat message inside a Conversation (Discussion
// feature). Named ChatMessage, not Message — models.Message is already the
// generic {message: "..."} API response wrapper used everywhere.
type ChatMessage struct {
	// The unique, numeric id of this message.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this message."`
	// The id of the conversation this message belongs to. Taken from the URL, not the body.
	ConversationID int64 `xorm:"bigint not null index" json:"conversation_id" param:"conversation" readOnly:"true" doc:"The id of the conversation this message belongs to. Taken from the URL, not the body."`
	// The message text.
	Text string `xorm:"longtext not null" json:"text" valid:"runelength(1|8000)" minLength:"1" maxLength:"8000" doc:"The message text."`

	AuthorID int64 `xorm:"bigint not null" json:"-"`
	// The user who sent this message.
	Author *user.User `xorm:"-" json:"author" readOnly:"true" doc:"The user who sent this message."`

	// A timestamp when this message was sent. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this message was sent. You cannot change this value."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for chat messages
func (*ChatMessage) TableName() string {
	return "chat_messages"
}

// Create sends a new message into m.ConversationID.
func (m *ChatMessage) Create(s *xorm.Session, a web.Auth) (err error) {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return
	}

	m.ID = 0
	m.Author = u
	m.AuthorID = u.ID

	if _, err = s.Insert(m); err != nil {
		return
	}

	// The author has, by definition, seen their own message — without this
	// their own conversation list would show it as unread to themselves.
	if err = markConversationRead(s, m.ConversationID, u.ID); err != nil {
		return
	}

	// Bump the conversation so it sorts to the top of the list.
	_, err = s.ID(m.ConversationID).Cols("updated").Update(&Conversation{Updated: time.Now()})
	return
}

// ReadAll returns every message in m.ConversationID, oldest first.
func (m *ChatMessage) ReadAll(s *xorm.Session, a web.Auth, _ string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	is, err := isConversationMember(s, m.ConversationID, a.GetID())
	if err != nil {
		return nil, 0, 0, err
	}
	if !is {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	// Fetching the message list is how a user "opens" a conversation — mark
	// it read now rather than requiring a separate endpoint the frontend
	// would have to remember to call.
	if err = markConversationRead(s, m.ConversationID, a.GetID()); err != nil {
		return nil, 0, 0, err
	}

	messages := []*ChatMessage{}
	limit, start := getLimitFromPageIndex(page, perPage)
	query := s.Where("conversation_id = ?", m.ConversationID).Asc("id")
	if limit > 0 {
		query = query.Limit(limit, start)
	}
	if err = query.Find(&messages); err != nil {
		return nil, 0, 0, err
	}

	if err = addAuthorsToMessages(s, messages); err != nil {
		return nil, 0, 0, err
	}

	numberOfTotalItems, err = s.Where("conversation_id = ?", m.ConversationID).Count(&ChatMessage{})
	return messages, len(messages), numberOfTotalItems, err
}

func getLastMessage(s *xorm.Session, conversationID int64) (*ChatMessage, error) {
	msg := &ChatMessage{}
	exists, err := s.Where("conversation_id = ?", conversationID).Desc("id").Get(msg)
	if err != nil || !exists {
		return nil, err
	}
	if err = addAuthorsToMessages(s, []*ChatMessage{msg}); err != nil {
		return nil, err
	}
	return msg, nil
}

func addAuthorsToMessages(s *xorm.Session, messages []*ChatMessage) error {
	if len(messages) == 0 {
		return nil
	}

	authorIDs := make([]int64, 0, len(messages))
	for _, msg := range messages {
		authorIDs = append(authorIDs, msg.AuthorID)
	}

	users, err := getUsersOrLinkSharesFromIDs(s, authorIDs)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		if author, has := users[msg.AuthorID]; has {
			msg.Author = author
		}
	}
	return nil
}
