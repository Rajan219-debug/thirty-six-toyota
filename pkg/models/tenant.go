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

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

// Tenant is an outlet / dealership location. Every tenant-scoped row references
// one via tenant_id; isolation between outlets is enforced by Postgres RLS
// (see pkg/db). The table is created and the default outlet seeded by
// db.EnsureTenantColumns; this model provides admin CRUD on top of it.
type Tenant struct {
	// The unique numeric id of this outlet.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique numeric id of this outlet."`
	// The display name of the outlet, e.g. "Faridabad".
	Name string `xorm:"varchar(250) not null" json:"name" valid:"length(1|250)" minLength:"1" maxLength:"250" doc:"The display name of the outlet."`
	// A short unique slug for the outlet, e.g. "faridabad".
	Slug string `xorm:"varchar(100) not null unique" json:"slug" valid:"length(1|100)" minLength:"1" maxLength:"100" doc:"A short, unique slug for the outlet."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true"`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true"`
}

// TableName is the outlet registry table.
func (Tenant) TableName() string {
	return "tenants"
}

// GetAllTenants returns every outlet, ordered by name.
func GetAllTenants(s *xorm.Session) (tenants []*Tenant, err error) {
	tenants = []*Tenant{}
	err = s.OrderBy("name").Find(&tenants)
	return
}

// CreateTenant inserts a new outlet.
func CreateTenant(s *xorm.Session, t *Tenant) error {
	_, err := s.Insert(t)
	return err
}

// SetUserOutlet assigns a user to an outlet and sets their HQ flag (HQ users
// read across all outlets — see the RLS policy). Written with raw SQL so the
// tenancy columns stay out of the User struct: the ORM never touches tenant_id,
// only RLS and this helper do. Validates that the outlet exists first.
func SetUserOutlet(s *xorm.Session, targetUserID, tenantID int64, isHQ bool) error {
	exists, err := s.ID(tenantID).Exist(&Tenant{})
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidData{Message: "outlet does not exist"}
	}
	res, err := s.Exec("UPDATE users SET tenant_id = ?, is_hq = ? WHERE id = ?", tenantID, isHQ, targetUserID)
	if err != nil {
		return err
	}
	// Postgres reports rows matched by the WHERE, so 0 means no such user rather
	// than an unchanged value — surface it instead of a silent success.
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrInvalidData{Message: "user does not exist"}
	}
	return nil
}

// FilterUsersByOutlet drops users that belong to a different outlet than the
// caller, unless the caller is HQ (who reads across all outlets). The users
// table is RLS-exempt (auth looks users up before a tenant is bound), so the
// global user directory would otherwise leak every outlet's staff — this scopes
// it in app code. A no-op when enforcement is off.
func FilterUsersByOutlet(s *xorm.Session, callerID int64, users []*user.User) ([]*user.User, error) {
	if !db.MultiTenancyEnforced() || len(users) == 0 {
		return users, nil
	}

	var caller struct {
		TenantID int64 `xorm:"tenant_id"`
		IsHQ     bool  `xorm:"is_hq"`
	}
	has, err := s.SQL("SELECT tenant_id, is_hq FROM users WHERE id = ?", callerID).Get(&caller)
	if err != nil {
		return nil, err
	}
	if !has || caller.IsHQ {
		return users, nil
	}

	ids := make([]int64, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	sameOutlet := []int64{}
	if err := s.Table("users").Cols("id").
		Where("tenant_id = ?", caller.TenantID).In("id", ids).
		Find(&sameOutlet); err != nil {
		return nil, err
	}
	keep := make(map[int64]bool, len(sameOutlet))
	for _, id := range sameOutlet {
		keep[id] = true
	}

	out := users[:0]
	for _, u := range users {
		if keep[u.ID] {
			out = append(out, u)
		}
	}
	return out, nil
}
