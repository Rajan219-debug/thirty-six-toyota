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

// Notes are private: a user can only ever read or change their own notes.

// CanRead checks if a user can see this note. Only the owner can.
func (n *Note) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	owner, err := n.isNoteOwner(s, a)
	if err != nil {
		return false, 0, err
	}
	if !owner {
		return false, 0, nil
	}
	return true, int(PermissionAdmin), nil
}

// CanCreate checks if the user can create a note. Any logged-in user can.
func (n *Note) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}
	return true, nil
}

// CanUpdate checks if a user can update a note. Only the owner can.
func (n *Note) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return n.isNoteOwner(s, a)
}

// CanDelete checks if a user can delete a note. Only the owner can.
func (n *Note) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return n.isNoteOwner(s, a)
}

func (n *Note) isNoteOwner(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}

	orig := &Note{}
	exists, err := s.Where("id = ?", n.ID).Get(orig)
	if err != nil || !exists {
		return false, err
	}
	return orig.CreatedByID == a.GetID(), nil
}
