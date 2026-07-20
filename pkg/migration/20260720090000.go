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
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID: "20260720090000",
		Description: "Backfill NULL tasks.priority to 0 and default the column to 0 " +
			"(Postgres/MySQL) — a task created without an explicit priority stored NULL " +
			"instead of 0 because the column had no default, which sorted those tasks to " +
			"the wrong end of an ascending sort_by=priority list instead of alongside the " +
			"other unprioritized tasks",
		Migrate: func(tx *xorm.Engine) error {
			if _, err := tx.Exec("UPDATE tasks SET priority = 0 WHERE priority IS NULL"); err != nil {
				return err
			}

			// SQLite has no ALTER COLUMN ... SET DEFAULT; the backfill above already
			// fixes existing rows, and this fork's production database is Postgres-only
			// (see test.yml), so a SQLite default is not required for the live app.
			if tx.Dialect().URI().DBType == schemas.SQLITE {
				return nil
			}

			_, err := tx.Exec("ALTER TABLE tasks ALTER COLUMN priority SET DEFAULT 0")
			return err
		},
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
