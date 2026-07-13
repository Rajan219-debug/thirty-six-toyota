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

type folder20260712120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	Title       string    `xorm:"varchar(250) not null"`
	CreatedByID int64     `xorm:"bigint not null"`
	Created     time.Time `xorm:"created not null"`
	Updated     time.Time `xorm:"updated not null"`
}

func (folder20260712120000) TableName() string {
	return "folders"
}

type document20260712120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	FolderID    int64     `xorm:"bigint not null index"`
	FileID      int64     `xorm:"bigint not null"`
	CreatedByID int64     `xorm:"bigint not null"`
	Created     time.Time `xorm:"created not null"`
}

func (document20260712120000) TableName() string {
	return "documents"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260712120000",
		Description: "Add folders and documents tables for the Documents feature",
		Migrate: func(tx *xorm.Engine) error {
			if err := tx.Sync(folder20260712120000{}); err != nil {
				return err
			}
			return tx.Sync(document20260712120000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			if err := tx.DropTables(document20260712120000{}); err != nil {
				return err
			}
			return tx.DropTables(folder20260712120000{})
		},
	})
}
