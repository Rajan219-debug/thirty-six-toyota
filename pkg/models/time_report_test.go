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

func TestWeekStart(t *testing.T) {
	monday := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	t.Run("mid-week day maps to its Monday", func(t *testing.T) {
		assert.Equal(t, monday, weekStart(time.Date(2026, 7, 9, 15, 30, 0, 0, time.UTC)))
	})

	t.Run("Sunday belongs to the week begun the previous Monday", func(t *testing.T) {
		assert.Equal(t, monday, weekStart(time.Date(2026, 7, 12, 23, 59, 59, 0, time.UTC)))
	})

	t.Run("a Monday is its own week start", func(t *testing.T) {
		assert.Equal(t, monday, weekStart(monday))
	})
}

func TestWeeklyTrend(t *testing.T) {
	week := func(offset int) time.Time {
		return time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC).AddDate(0, 0, offset*7)
	}

	t.Run("empty input gives an empty series", func(t *testing.T) {
		assert.Empty(t, weeklyTrend(map[time.Time]int64{}))
	})

	t.Run("gaps between buckets are filled with zero weeks", func(t *testing.T) {
		got := weeklyTrend(map[time.Time]int64{week(0): 2, week(2): 1})
		assert.Equal(t, []*LateCompletionWeek{
			{WeekStart: week(0), Count: 2},
			{WeekStart: week(1), Count: 0},
			{WeekStart: week(2), Count: 1},
		}, got)
	})

	t.Run("a single week stays a single bucket", func(t *testing.T) {
		got := weeklyTrend(map[time.Time]int64{week(1): 3})
		assert.Equal(t, []*LateCompletionWeek{{WeekStart: week(1), Count: 3}}, got)
	})
}
