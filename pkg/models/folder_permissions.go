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
	"code.vikunja.io/api/pkg/web"
	"xorm.io/xorm"
)

// Documents is a shared, company-wide drive — every authenticated (non-link-share)
// user can read every folder/document, mirroring Users/Departments' openness in
// this instance. Only the uploader may rename/delete.

// CanRead checks if a user can see this folder. Every logged-in user can.
func (f *Folder) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if _, is := a.(*LinkSharing); is {
		return false, 0, nil
	}
	owner, err := f.isFolderOwner(s, a)
	if err != nil {
		return false, 0, err
	}
	if owner {
		return true, int(PermissionAdmin), nil
	}
	return true, int(PermissionRead), nil
}

// CanCreate checks if the user can create a folder. Any logged-in user can.
func (f *Folder) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}
	return true, nil
}

// CanUpdate checks if a user can update a folder. Only the creator can.
func (f *Folder) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return f.isFolderOwner(s, a)
}

// CanDelete checks if a user can delete a folder. Only the creator can.
func (f *Folder) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return f.isFolderOwner(s, a)
}

func (f *Folder) isFolderOwner(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}

	orig := &Folder{}
	exists, err := s.Where("id = ?", f.ID).Get(orig)
	if err != nil || !exists {
		return false, err
	}
	return orig.CreatedByID == a.GetID(), nil
}
