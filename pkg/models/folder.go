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

// Folder is a shared, company-wide document folder (Documents feature).
// ponytail: flat, one level deep — no nested sub-folders. Add ParentFolderID
// if the drive ever needs a real tree.
type Folder struct {
	// The unique, numeric id of this folder.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"folder" readOnly:"true" doc:"The unique, numeric id of this folder."`
	// The folder's display name.
	Title string `xorm:"varchar(250) not null" json:"title" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"The folder's display name."`

	CreatedByID int64 `xorm:"bigint not null" json:"-"`
	// The user who created this folder.
	CreatedBy *user.User `xorm:"-" json:"created_by" readOnly:"true" doc:"The user who created this folder."`

	// A timestamp when this folder was created. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this folder was created. You cannot change this value."`
	// A timestamp when this folder was last updated. You cannot change this value.
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"A timestamp when this folder was last updated. You cannot change this value."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for folders
func (*Folder) TableName() string {
	return "folders"
}

// Create creates a new folder
func (f *Folder) Create(s *xorm.Session, a web.Auth) (err error) {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return
	}

	f.ID = 0
	f.CreatedBy = u
	f.CreatedByID = u.ID

	_, err = s.Insert(f)
	return
}

// Update updates a folder's title
func (f *Folder) Update(s *xorm.Session, a web.Auth) (err error) {
	_, err = s.
		ID(f.ID).
		Cols("title").
		Update(f)
	if err != nil {
		return
	}

	return f.ReadOne(s, a)
}

// Delete deletes a folder and every document inside it
func (f *Folder) Delete(s *xorm.Session, _ web.Auth) (err error) {
	docs := []*Document{}
	if err = s.Where("folder_id = ?", f.ID).Find(&docs); err != nil {
		return
	}
	for _, d := range docs {
		if err = d.deleteFileAndRow(s); err != nil {
			return
		}
	}

	_, err = s.ID(f.ID).Delete(&Folder{})
	return
}

// ReadAll returns every folder (shared, company-wide drive)
func (f *Folder) ReadAll(s *xorm.Session, _ web.Auth, search string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	folders := []*Folder{}

	limit, start := getLimitFromPageIndex(page, perPage)
	query := s.Where("title like ?", "%"+search+"%")
	if limit > 0 {
		query = query.Limit(limit, start)
	}
	if err = query.Find(&folders); err != nil {
		return nil, 0, 0, err
	}

	if err = addCreatedByToFolders(s, folders); err != nil {
		return nil, 0, 0, err
	}

	numberOfTotalItems, err = s.Where("title like ?", "%"+search+"%").Count(&Folder{})
	return folders, len(folders), numberOfTotalItems, err
}

// ReadOne returns a single folder
func (f *Folder) ReadOne(s *xorm.Session, _ web.Auth) (err error) {
	exists, err := s.Where("id = ?", f.ID).Get(f)
	if err != nil {
		return
	}
	if !exists {
		return ErrFolderDoesNotExist{FolderID: f.ID}
	}

	f.CreatedBy, err = user.GetUserByID(s, f.CreatedByID)
	return
}

func addCreatedByToFolders(s *xorm.Session, folders []*Folder) error {
	if len(folders) == 0 {
		return nil
	}

	userIDs := make([]int64, 0, len(folders))
	for _, fo := range folders {
		userIDs = append(userIDs, fo.CreatedByID)
	}

	users, err := getUsersOrLinkSharesFromIDs(s, userIDs)
	if err != nil {
		return err
	}

	for _, fo := range folders {
		if createdBy, has := users[fo.CreatedByID]; has {
			fo.CreatedBy = createdBy
		}
	}
	return nil
}
