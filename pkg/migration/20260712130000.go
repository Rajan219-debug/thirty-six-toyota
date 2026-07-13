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

package migration

import (
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

type conversation20260712130000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	Title       string    `xorm:"varchar(250) null"`
	IsGroup     bool      `xorm:"bool not null default false"`
	CreatedByID int64     `xorm:"bigint not null"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (conversation20260712130000) TableName() string {
	return "conversations"
}

type conversationMember20260712130000 struct {
	ID             int64     `xorm:"bigint autoincr not null unique pk"`
	ConversationID int64     `xorm:"bigint not null index"`
	UserID         int64     `xorm:"bigint not null index"`
	Created        time.Time `xorm:"created not null"`
}

func (conversationMember20260712130000) TableName() string {
	return "conversation_members"
}

type chatMessage20260712130000 struct {
	ID             int64     `xorm:"bigint autoincr not null unique pk"`
	ConversationID int64     `xorm:"bigint not null index"`
	Text           string    `xorm:"longtext not null"`
	AuthorID       int64     `xorm:"bigint not null"`
	Created        time.Time `xorm:"created not null"`
}

func (chatMessage20260712130000) TableName() string {
	return "chat_messages"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260712130000",
		Description: "Add conversations, conversation_members and chat_messages tables for the Discussion feature",
		Migrate: func(tx *xorm.Engine) error {
			if err := tx.Sync(conversation20260712130000{}); err != nil {
				return err
			}
			if err := tx.Sync(conversationMember20260712130000{}); err != nil {
				return err
			}
			return tx.Sync(chatMessage20260712130000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			if err := tx.DropTables(chatMessage20260712130000{}); err != nil {
				return err
			}
			if err := tx.DropTables(conversationMember20260712130000{}); err != nil {
				return err
			}
			return tx.DropTables(conversation20260712130000{})
		},
	})
}
