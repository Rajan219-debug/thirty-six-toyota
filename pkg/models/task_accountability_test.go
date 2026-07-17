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

func TestCalculateTaskStatus(t *testing.T) {
	now := time.Now()

	t.Run("pending when not done, no due date, no progress", func(t *testing.T) {
		task := &Task{}
		assert.Equal(t, TaskStatusPending, CalculateTaskStatus(task))
	})

	t.Run("pending when due date is in the future", func(t *testing.T) {
		task := &Task{DueDate: now.Add(48 * time.Hour)}
		assert.Equal(t, TaskStatusPending, CalculateTaskStatus(task))
	})

	t.Run("ongoing when progress has been made", func(t *testing.T) {
		task := &Task{PercentDone: 0.25}
		assert.Equal(t, TaskStatusOngoing, CalculateTaskStatus(task))
	})

	t.Run("overdue when due date has passed and not done", func(t *testing.T) {
		task := &Task{DueDate: now.Add(-time.Hour)}
		assert.Equal(t, TaskStatusOverdue, CalculateTaskStatus(task))
	})

	t.Run("overdue wins over ongoing", func(t *testing.T) {
		task := &Task{DueDate: now.Add(-time.Hour), PercentDone: 0.5}
		assert.Equal(t, TaskStatusOverdue, CalculateTaskStatus(task))
	})

	t.Run("pending confirmation when a closure has been claimed", func(t *testing.T) {
		task := &Task{
			DueDate:             now.Add(-time.Hour),
			CompletionClaimedAt: now,
		}
		assert.Equal(t, TaskStatusPendingConfirmation, CalculateTaskStatus(task))
	})

	t.Run("completed when done on time", func(t *testing.T) {
		task := &Task{
			Done:    true,
			DueDate: now.Add(time.Hour),
			DoneAt:  now,
		}
		assert.Equal(t, TaskStatusCompleted, CalculateTaskStatus(task))
	})

	t.Run("completed when done without a due date", func(t *testing.T) {
		task := &Task{Done: true, DoneAt: now}
		assert.Equal(t, TaskStatusCompleted, CalculateTaskStatus(task))
	})

	t.Run("completed late when done after the due date with approver confirmation", func(t *testing.T) {
		task := &Task{
			Done:          true,
			DueDate:       now.Add(-24 * time.Hour),
			DoneAt:        now,
			ConfirmedByID: 42,
		}
		assert.Equal(t, TaskStatusCompletedLate, CalculateTaskStatus(task))
	})

	t.Run("plain completed when done after due date but never confirmed", func(t *testing.T) {
		// A task closed before the accountability flow existed, or closed
		// directly by its own assigner, has no ConfirmedByID.
		task := &Task{
			Done:    true,
			DueDate: now.Add(-24 * time.Hour),
			DoneAt:  now,
		}
		assert.Equal(t, TaskStatusCompleted, CalculateTaskStatus(task))
	})

	t.Run("done wins over a stale completion claim", func(t *testing.T) {
		task := &Task{
			Done:                true,
			DoneAt:              now,
			CompletionClaimedAt: now.Add(-time.Hour),
		}
		assert.Equal(t, TaskStatusCompleted, CalculateTaskStatus(task))
	})
}

func TestCalculatePriorityScore(t *testing.T) {
	t.Run("unrated inputs score as the neutral middle, still do_later with no due date", func(t *testing.T) {
		// Unrated (0) is scored as 3: 0.3*3 + 0.3*3 + 0.4*0 = 1.8 -> do_later
		score, bucket := CalculatePriorityScore(0, 0, time.Time{})
		assert.InDelta(t, 1.8, score, 0.001)
		assert.Equal(t, PriorityBucketDoLater, bucket)
	})

	t.Run("unrated overdue task reaches do_now", func(t *testing.T) {
		// Regression guard: unrated used to score 0, which capped an overdue
		// task at 0.4*5 = 2.0 (do_soon), so Do Now could never fill unless
		// someone hand-rated every task. 0.3*3 + 0.3*3 + 0.4*5 = 3.8 -> do_now
		score, bucket := CalculatePriorityScore(0, 0, time.Now().Add(-time.Hour))
		assert.InDelta(t, 3.8, score, 0.001)
		assert.Equal(t, PriorityBucketDoNow, bucket)
	})

	t.Run("max urgency and importance without a due date", func(t *testing.T) {
		// 0.3*5 + 0.3*5 + 0.4*0 = 3.0 -> do_soon
		score, bucket := CalculatePriorityScore(5, 5, time.Time{})
		assert.InDelta(t, 3.0, score, 0.001)
		assert.Equal(t, PriorityBucketDoSoon, bucket)
	})

	t.Run("past due date contributes full proximity", func(t *testing.T) {
		// 0.3*5 + 0.3*5 + 0.4*5 = 5.0 -> do_now
		score, bucket := CalculatePriorityScore(5, 5, time.Now().Add(-time.Hour))
		assert.InDelta(t, 5.0, score, 0.001)
		assert.Equal(t, PriorityBucketDoNow, bucket)
	})

	t.Run("far-future due date contributes nothing", func(t *testing.T) {
		score, bucket := CalculatePriorityScore(1, 1, time.Now().Add(30*24*time.Hour))
		assert.InDelta(t, 0.6, score, 0.001)
		assert.Equal(t, PriorityBucketDoLater, bucket)
	})

	t.Run("proximity ramps within the deadline window", func(t *testing.T) {
		// Halfway through the 7-day window: proximity ~2.5. Unrated inputs
		// contribute the 1.8 baseline, so 1.8 + 0.4*2.5 = ~2.8
		score, _ := CalculatePriorityScore(0, 0, time.Now().Add(priorityDeadlineWindow/2))
		assert.InDelta(t, 2.8, score, 0.01)
	})

	t.Run("overdue low-stakes task lands in do_soon", func(t *testing.T) {
		// 0.3*1 + 0.3*1 + 0.4*5 = 2.6 -> do_soon
		score, bucket := CalculatePriorityScore(1, 1, time.Now().Add(-time.Hour))
		assert.InDelta(t, 2.6, score, 0.001)
		assert.Equal(t, PriorityBucketDoSoon, bucket)
	})

	t.Run("bucket boundary at 3.5 is do_now", func(t *testing.T) {
		// 0.3*5 + 0.3*5 + 0.4*1.25 = 3.5 exactly
		_, bucket := CalculatePriorityScore(5, 5, time.Now().Add(priorityDeadlineWindow*3/4))
		assert.Equal(t, PriorityBucketDoNow, bucket)
	})
}
