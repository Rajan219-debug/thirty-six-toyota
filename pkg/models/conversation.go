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

// Conversation is a chat between two or more users (Discussion feature).
// ponytail: membership is fixed at creation — no add/remove-member endpoint.
// Add one if group chats need to grow after creation.
type Conversation struct {
	// The unique, numeric id of this conversation.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"conversation" readOnly:"true" doc:"The unique, numeric id of this conversation."`
	// The conversation's title. Only used for group chats — 1:1 chats are titled after the other member on the frontend.
	Title string `xorm:"varchar(250) null" json:"title" maxLength:"250" doc:"The conversation's title. Only used for group chats — 1:1 chats are titled after the other member on the frontend."`
	// Whether this is a group chat (3+ members) or a 1:1 direct message.
	IsGroup bool `xorm:"bool not null default false" json:"is_group" doc:"Whether this is a group chat (3+ members) or a 1:1 direct message."`

	// The ids of the other members to start this conversation with. Write-only — used on create.
	MemberIDs []int64 `xorm:"-" json:"member_ids" maxItems:"50" doc:"The ids of the other members to start this conversation with. Write-only — used on create. Every id must be an existing user; at most 50 members."`
	// The full member list of this conversation, including the creator.
	Members []*user.User `xorm:"-" json:"members" readOnly:"true" doc:"The full member list of this conversation, including the creator."`
	// The most recent message in this conversation, if any.
	LastMessage *ChatMessage `xorm:"-" json:"last_message" readOnly:"true" doc:"The most recent message in this conversation, if any."`
	// Whether the caller has unread messages in this conversation — true if
	// the last message came in after they last fetched the message list.
	IsUnread bool `xorm:"-" json:"is_unread" readOnly:"true" doc:"Whether the caller has unread messages in this conversation."`

	CreatedByID int64 `xorm:"bigint not null" json:"-"`
	// The user who started this conversation.
	CreatedBy *user.User `xorm:"-" json:"created_by" readOnly:"true" doc:"The user who started this conversation."`

	// A timestamp when this conversation was created. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this conversation was created. You cannot change this value."`
	// A timestamp when this conversation was last updated (e.g. a new message). You cannot change this value.
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when this conversation was last updated (e.g. a new message). You cannot change this value."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for conversations
func (*Conversation) TableName() string {
	return "conversations"
}

// Create creates a new conversation with the caller plus every id in MemberIDs as members.
func (c *Conversation) Create(s *xorm.Session, a web.Auth) (err error) {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return
	}

	memberIDs := uniqueInt64s(c.MemberIDs)
	if len(memberIDs) == 0 {
		return ErrConversationNeedsMembers{}
	}
	// Huma enforces maxItems on the schema; re-check here so the v1-style
	// pipeline (and any future caller) can't bypass it and mass-insert rows.
	if len(memberIDs) > 50 {
		return ErrConversationNeedsMembers{}
	}

	// Reject ids that aren't real users instead of silently creating ghost
	// memberships that hydrateConversations would filter out on display.
	members, err := user.GetUsersByIDs(s, memberIDs)
	if err != nil {
		return err
	}
	for _, id := range memberIDs {
		if _, is := members[id]; !is {
			return ErrConversationMemberDoesNotExist{UserID: id}
		}
	}

	c.ID = 0
	c.CreatedBy = u
	c.CreatedByID = u.ID
	if len(memberIDs) > 1 {
		c.IsGroup = true
	}

	if _, err = s.Insert(c); err != nil {
		return
	}

	allMemberIDs := append([]int64{u.ID}, memberIDs...)
	for _, memberID := range allMemberIDs {
		member := &ConversationMember{ConversationID: c.ID, UserID: memberID}
		if _, err = s.Insert(member); err != nil {
			return
		}
	}

	return c.ReadOne(s, a)
}

// ReadOne returns a single conversation the caller is a member of.
func (c *Conversation) ReadOne(s *xorm.Session, a web.Auth) (err error) {
	exists, err := s.Where("id = ?", c.ID).Get(c)
	if err != nil {
		return
	}
	if !exists {
		return ErrConversationDoesNotExist{ConversationID: c.ID}
	}

	if err = hydrateConversations(s, []*Conversation{c}); err != nil {
		return
	}
	lastReadAt, err := getLastReadAt(s, c.ID, a.GetID())
	if err != nil {
		return
	}
	c.IsUnread = conversationIsUnread(c.LastMessage, lastReadAt)
	return
}

// ReadAll returns every conversation the caller is a member of, newest activity first.
func (c *Conversation) ReadAll(s *xorm.Session, a web.Auth, _ string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	if _, is := a.(*LinkSharing); is {
		return []*Conversation{}, 0, 0, nil
	}

	memberships := []*ConversationMember{}
	err = s.Where("user_id = ?", a.GetID()).Find(&memberships)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(memberships) == 0 {
		return []*Conversation{}, 0, 0, nil
	}
	conversationIDs := make([]int64, 0, len(memberships))
	lastReadAtByConversation := make(map[int64]time.Time, len(memberships))
	for _, m := range memberships {
		conversationIDs = append(conversationIDs, m.ConversationID)
		lastReadAtByConversation[m.ConversationID] = m.LastReadAt
	}

	conversations := []*Conversation{}
	limit, start := getLimitFromPageIndex(page, perPage)
	query := s.In("id", conversationIDs).Desc("updated")
	if limit > 0 {
		query = query.Limit(limit, start)
	}
	if err = query.Find(&conversations); err != nil {
		return nil, 0, 0, err
	}

	if err = hydrateConversations(s, conversations); err != nil {
		return nil, 0, 0, err
	}
	for _, conv := range conversations {
		conv.IsUnread = conversationIsUnread(conv.LastMessage, lastReadAtByConversation[conv.ID])
	}

	numberOfTotalItems, err = s.In("id", conversationIDs).Count(&Conversation{})
	return conversations, len(conversations), numberOfTotalItems, err
}

// hydrateConversations fills in Members, CreatedBy and LastMessage for a
// batch of conversations in a fixed small number of queries, regardless of
// how many conversations are passed — not the three-per-conversation
// (members, creator, last message) round trips this used to do in a loop,
// which made listing conversations scale with query count, not row count.
func hydrateConversations(s *xorm.Session, conversations []*Conversation) error {
	if len(conversations) == 0 {
		return nil
	}

	conversationIDs := make([]int64, len(conversations))
	for i, conv := range conversations {
		conversationIDs[i] = conv.ID
	}

	memberRows := []*ConversationMember{}
	if err := s.In("conversation_id", conversationIDs).Find(&memberRows); err != nil {
		return err
	}

	memberIDsByConversation := make(map[int64][]int64, len(conversations))
	allReferencedIDs := make([]int64, 0, len(memberRows)+len(conversations))
	for _, m := range memberRows {
		memberIDsByConversation[m.ConversationID] = append(memberIDsByConversation[m.ConversationID], m.UserID)
		allReferencedIDs = append(allReferencedIDs, m.UserID)
	}
	for _, conv := range conversations {
		allReferencedIDs = append(allReferencedIDs, conv.CreatedByID)
	}

	users, err := getUsersOrLinkSharesFromIDs(s, uniqueInt64s(allReferencedIDs))
	if err != nil {
		return err
	}

	lastMessages, err := getLastMessagesForConversations(s, conversationIDs)
	if err != nil {
		return err
	}

	for _, conv := range conversations {
		memberIDs := memberIDsByConversation[conv.ID]
		conv.Members = make([]*user.User, 0, len(memberIDs))
		for _, id := range memberIDs {
			if u, has := users[id]; has {
				conv.Members = append(conv.Members, u)
			}
		}
		conv.CreatedBy = users[conv.CreatedByID]
		conv.LastMessage = lastMessages[conv.ID]
	}
	return nil
}

// conversationIsUnread reports whether the last message came in after the
// caller last read the conversation. No message yet is never unread; never
// having read (the zero time) means any message counts as unread.
func conversationIsUnread(lastMessage *ChatMessage, lastReadAt time.Time) bool {
	return lastMessage != nil && lastMessage.Created.After(lastReadAt)
}

func uniqueInt64s(in []int64) []int64 {
	seen := make(map[int64]bool, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if v == 0 || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
