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

// CanRead checks if a user can see this document — same rule as its folder.
func (d *Document) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	orig := &Document{}
	exists, err := s.Where("id = ?", d.ID).Get(orig)
	if err != nil {
		return false, 0, err
	}
	if !exists {
		return false, 0, nil
	}
	folder := &Folder{ID: orig.FolderID}
	return folder.CanRead(s, a)
}

// CanCreate checks if a user can upload a document into d.FolderID.
func (d *Document) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	folder := &Folder{ID: d.FolderID}
	can, _, err := folder.CanRead(s, a)
	return can, err
}

// CanDelete checks if a user can delete a document. Only the uploader can.
func (d *Document) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}

	orig := &Document{}
	exists, err := s.Where("id = ?", d.ID).Get(orig)
	if err != nil || !exists {
		return false, err
	}
	return orig.CreatedByID == a.GetID(), nil
}
