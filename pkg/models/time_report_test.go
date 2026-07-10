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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntryDuration(t *testing.T) {
	start := time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC)

	t.Run("one hour", func(t *testing.T) {
		end := start.Add(time.Hour)
		assert.Equal(t, int64(3600), entryDuration(start, &end))
	})

	t.Run("running timer counts as zero", func(t *testing.T) {
		assert.Equal(t, int64(0), entryDuration(start, nil))
	})

	t.Run("negative interval clamps to zero", func(t *testing.T) {
		end := start.Add(-time.Hour)
		assert.Equal(t, int64(0), entryDuration(start, &end))
	})

	t.Run("zero-length entry is zero", func(t *testing.T) {
		assert.Equal(t, int64(0), entryDuration(start, &start))
	})

	t.Run("sub-second truncates down", func(t *testing.T) {
		end := start.Add(1500 * time.Millisecond)
		assert.Equal(t, int64(1), entryDuration(start, &end))
	})
}
