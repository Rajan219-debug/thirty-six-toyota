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

type note20260713120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	Title       string    `xorm:"varchar(250) not null"`
	Body        string    `xorm:"longtext null"`
	Color       string    `xorm:"varchar(20) not null"`
	Pinned      bool      `xorm:"not null"`
	CreatedByID int64     `xorm:"bigint not null index"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (note20260713120000) TableName() string {
	return "notes"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260713120000",
		Description: "Add notes table for the Notes feature",
		Migrate: func(tx *xorm.Engine) error {
			return tx.Sync(note20260713120000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			return tx.DropTables(note20260713120000{})
		},
	})
}
