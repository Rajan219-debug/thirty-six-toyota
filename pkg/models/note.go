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

// Note is a private sticky note owned by a single user (Notes feature).
// Each user only ever sees their own notes.
type Note struct {
	// The unique, numeric id of this note.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"note" readOnly:"true" doc:"The unique, numeric id of this note."`
	// The note's title.
	Title string `xorm:"varchar(250) not null" json:"title" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"The note's title."`
	// The note's body text.
	Body string `xorm:"longtext null" json:"body" valid:"runelength(0|8000)" maxLength:"8000" doc:"The note's body text."`
	// The note's color swatch (amber, mint or lavender). Purely presentational.
	Color string `xorm:"varchar(20) not null" json:"color" valid:"in(amber|mint|lavender)" minLength:"1" maxLength:"20" doc:"The note's color swatch. Must be exactly one of: amber, mint, lavender."`
	// Whether the note is pinned to the top of the list.
	Pinned bool `xorm:"not null" json:"pinned" doc:"Whether the note is pinned to the top of the list."`

	CreatedByID int64 `xorm:"bigint not null index" json:"-"`

	// A timestamp when this note was created. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this note was created. You cannot change this value."`
	// A timestamp when this note was last updated. You cannot change this value.
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when this note was last updated. You cannot change this value."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for notes
func (*Note) TableName() string {
	return "notes"
}

// Create creates a new note owned by the authenticated user
func (n *Note) Create(s *xorm.Session, a web.Auth) (err error) {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return
	}

	n.ID = 0
	n.CreatedByID = u.ID

	_, err = s.Insert(n)
	return
}

// Update updates a note's editable fields
func (n *Note) Update(s *xorm.Session, a web.Auth) (err error) {
	_, err = s.
		ID(n.ID).
		Cols("title", "body", "color", "pinned").
		Update(n)
	if err != nil {
		return
	}

	return n.ReadOne(s, a)
}

// Delete removes a note
func (n *Note) Delete(s *xorm.Session, _ web.Auth) (err error) {
	_, err = s.ID(n.ID).Delete(&Note{})
	return
}

// ReadAll returns every note owned by the authenticated user
func (n *Note) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	notes := []*Note{}

	limit, start := getLimitFromPageIndex(page, perPage)
	query := s.Where("created_by_id = ? AND title like ?", a.GetID(), "%"+search+"%").
		OrderBy("pinned desc, updated desc")
	if limit > 0 {
		query = query.Limit(limit, start)
	}
	if err = query.Find(&notes); err != nil {
		return nil, 0, 0, err
	}

	numberOfTotalItems, err = s.
		Where("created_by_id = ? AND title like ?", a.GetID(), "%"+search+"%").
		Count(&Note{})
	return notes, len(notes), numberOfTotalItems, err
}

// ReadOne returns a single note
func (n *Note) ReadOne(s *xorm.Session, _ web.Auth) (err error) {
	exists, err := s.Where("id = ?", n.ID).Get(n)
	if err != nil {
		return
	}
	if !exists {
		return ErrNoteDoesNotExist{NoteID: n.ID}
	}
	return
}
