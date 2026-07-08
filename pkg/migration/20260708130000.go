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

type taskAccountability20260708130000 struct {
	Urgency             int64     `xorm:"bigint null default 0"`
	Importance          int64     `xorm:"bigint null default 0"`
	CompletionClaimedAt time.Time `xorm:"datetime null 'completion_claimed_at'"`
	ConfirmedByID       int64     `xorm:"bigint null 'confirmed_by_id'"`
	RejectionReason     string    `xorm:"text null 'rejection_reason'"`
}

func (taskAccountability20260708130000) TableName() string {
	return "tasks"
}

type projectFallbackApprover20260708130000 struct {
	FallbackApproverID int64 `xorm:"bigint null 'fallback_approver_id'"`
}

func (projectFallbackApprover20260708130000) TableName() string {
	return "projects"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20260708130000",
		Description: "Add task accountability fields (urgency, importance, closure confirmation) and project fallback approver",
		Migrate: func(tx *xorm.Engine) error {
			if err := tx.Sync(taskAccountability20260708130000{}); err != nil {
				return err
			}
			return tx.Sync(projectFallbackApprover20260708130000{})
		},
		Rollback: func(tx *xorm.Engine) error {
			return nil
		},
	})
}
