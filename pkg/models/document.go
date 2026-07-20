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
	"io"
	"time"

	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// Document is an uploaded file inside a Folder (Documents feature).
type Document struct {
	// The unique, numeric id of this document.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" param:"document" readOnly:"true" doc:"The unique, numeric id of this document."`
	// The id of the folder this document belongs to.
	FolderID int64 `xorm:"bigint not null index" json:"folder_id" doc:"The id of the folder this document belongs to."`
	FileID   int64 `xorm:"bigint not null" json:"-"`

	CreatedByID int64 `xorm:"bigint not null" json:"-"`
	// The user who uploaded this document.
	CreatedBy *user.User `xorm:"-" json:"created_by" readOnly:"true" doc:"The user who uploaded this document."`

	// Metadata of the uploaded file (name, mime type, size). The bytes are fetched from the download endpoint, not this field.
	File *files.File `xorm:"-" json:"file" readOnly:"true" doc:"Metadata of the uploaded file (name, mime type, size). The bytes are fetched from the download endpoint, not this field."`

	// A timestamp when this document was uploaded. You cannot change this value.
	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"A timestamp when this document was uploaded. You cannot change this value."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName returns the table name for documents
func (*Document) TableName() string {
	return "documents"
}

// DocumentToUpload is a transport-neutral file to upload into a folder.
type DocumentToUpload struct {
	Reader   io.ReadSeeker
	Filename string
	Size     uint64
}

// UploadDocuments checks create access to the folder, then stores each file,
// collecting per-file failures rather than aborting. The caller owns the
// session and the commit.
func UploadDocuments(s *xorm.Session, a web.Auth, folderID int64, uploads []*DocumentToUpload) (success []*Document, failures []error, err error) {
	folder := &Folder{ID: folderID}
	can, _, err := folder.CanRead(s, a)
	if err != nil {
		return nil, nil, err
	}
	if !can {
		return nil, nil, ErrGenericForbidden{}
	}

	for _, upload := range uploads {
		doc := &Document{FolderID: folderID}
		if err := doc.newDocument(s, upload.Reader, upload.Filename, upload.Size, a); err != nil {
			failures = append(failures, err)
			continue
		}
		success = append(success, doc)
	}
	return success, failures, nil
}

func (d *Document) newDocument(s *xorm.Session, f io.ReadSeeker, realname string, realsize uint64, a web.Auth) error {
	file, err := files.CreateWithSession(s, f, realname, realsize, a)
	if err != nil {
		return err
	}
	d.File = file
	d.FileID = file.ID

	d.CreatedBy, err = GetUserOrLinkShareUser(s, a)
	if err != nil {
		if err2 := file.Delete(s); err2 != nil {
			return err2
		}
		return err
	}
	d.CreatedByID = d.CreatedBy.ID

	_, err = s.Insert(d)
	if err != nil {
		if err2 := file.Delete(s); err2 != nil {
			return err2
		}
		return err
	}

	return nil
}

// LoadDocumentForDownload checks read access and loads the document with its open file.
func LoadDocumentForDownload(s *xorm.Session, a web.Auth, documentID int64) (d *Document, err error) {
	d = &Document{ID: documentID}
	can, _, err := d.CanRead(s, a)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, ErrGenericForbidden{}
	}

	if err := d.ReadOne(s, a); err != nil {
		return nil, err
	}
	if err := d.File.LoadFileByID(); err != nil {
		return nil, err
	}
	return d, nil
}

// ReadOne returns a document
func (d *Document) ReadOne(s *xorm.Session, _ web.Auth) (err error) {
	exists, err := s.Where("id = ?", d.ID).Get(d)
	if err != nil {
		return
	}
	if !exists {
		return ErrDocumentDoesNotExist{DocumentID: d.ID}
	}

	d.File = &files.File{ID: d.FileID}
	if err = d.File.LoadFileMetaByID(); err != nil {
		return
	}

	d.CreatedBy, err = user.GetUserByID(s, d.CreatedByID)
	return
}

// ReadAll returns every document inside a folder
func (d *Document) ReadAll(s *xorm.Session, a web.Auth, _ string, page int, perPage int) (result interface{}, resultCount int, numberOfTotalItems int64, err error) {
	folder := &Folder{ID: d.FolderID}
	canRead, _, err := folder.CanRead(s, a)
	if err != nil {
		return nil, 0, 0, err
	}
	if !canRead {
		return nil, 0, 0, ErrGenericForbidden{}
	}

	docs := []*Document{}
	limit, start := getLimitFromPageIndex(page, perPage)
	query := s.Where("folder_id = ?", d.FolderID)
	if limit > 0 {
		query = query.Limit(limit, start)
	}
	if err = query.Find(&docs); err != nil {
		return nil, 0, 0, err
	}

	if len(docs) == 0 {
		return docs, 0, 0, nil
	}

	fileIDs := make([]int64, 0, len(docs))
	userIDs := make([]int64, 0, len(docs))
	for _, doc := range docs {
		fileIDs = append(fileIDs, doc.FileID)
		userIDs = append(userIDs, doc.CreatedByID)
	}

	fs := make(map[int64]*files.File)
	if err = s.In("id", fileIDs).Find(&fs); err != nil {
		return nil, 0, 0, err
	}

	users, err := getUsersOrLinkSharesFromIDs(s, userIDs)
	if err != nil {
		return nil, 0, 0, err
	}

	for _, doc := range docs {
		if createdBy, has := users[doc.CreatedByID]; has {
			doc.CreatedBy = createdBy
		}
		if fl, exists := fs[doc.FileID]; exists {
			doc.File = fl
		}
	}

	numberOfTotalItems, err = s.Where("folder_id = ?", d.FolderID).Count(&Document{})
	return docs, len(docs), numberOfTotalItems, err
}

// Delete removes a document
func (d *Document) Delete(s *xorm.Session, _ web.Auth) error {
	return d.deleteFileAndRow(s)
}

func (d *Document) deleteFileAndRow(s *xorm.Session) error {
	if err := d.ReadOne(s, nil); err != nil && !IsErrDocumentDoesNotExist(err) {
		return err
	}

	if _, err := s.ID(d.ID).Delete(&Document{}); err != nil {
		return err
	}

	if d.File == nil {
		return nil
	}
	err := d.File.Delete(s)
	if err != nil && files.IsErrFileDoesNotExist(err) {
		return nil
	}
	return err
}
