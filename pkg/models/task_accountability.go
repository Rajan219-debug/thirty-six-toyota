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

import "time"

// Task lifecycle status values. Computed on read from Done, DueDate, and the
// closure-confirmation fields — never stored directly.
const (
	TaskStatusPending             = "pending"
	TaskStatusOngoing             = "ongoing"
	TaskStatusOverdue             = "overdue"
	TaskStatusPendingConfirmation = "pending_confirmation"
	TaskStatusCompleted           = "completed"
	TaskStatusCompletedLate       = "completed_late"
)

// Priority buckets used by the "Do Now / Do Soon / Do Later" prioritized view.
const (
	PriorityBucketDoNow   = "do_now"
	PriorityBucketDoSoon  = "do_soon"
	PriorityBucketDoLater = "do_later"
)

// Weights for the priority score formula: Score = W1*Urgency + W2*Importance + W3*DeadlineProximity.
// All three inputs are normalized to a 0-5 scale before weighting.
const (
	priorityWeightUrgency           = 0.3
	priorityWeightImportance        = 0.3
	priorityWeightDeadlineProximity = 0.4

	// Deadline proximity ramps up over this window; a due date further out than
	// this contributes ~0, one that's already passed contributes the full 5.
	priorityDeadlineWindow = 7 * 24 * time.Hour

	// Urgency and importance are rated 1-5; 0 means nobody rated them (the
	// column defaults to 0 and an update that omits the field resets it). An
	// unrated input is scored as the neutral middle rather than 0: scoring it 0
	// would rank an unrated task *below* one explicitly rated lowest (1), and
	// because deadline proximity alone caps at 0.4*5 = 2.0, an unrated task
	// could never reach Do Now (>=3.5) no matter how overdue it got.
	priorityUnratedDefault = 3
)

// CalculateTaskStatus derives a task's lifecycle status from its done state,
// due date, and closure-confirmation fields. It never mutates t.
func CalculateTaskStatus(t *Task) string {
	isOverdue := !t.DueDate.IsZero() && t.DueDate.Before(time.Now())

	if t.Done {
		if t.ConfirmedByID != 0 && !t.DoneAt.IsZero() && !t.DueDate.IsZero() && t.DoneAt.After(t.DueDate) {
			return TaskStatusCompletedLate
		}
		return TaskStatusCompleted
	}

	if !t.CompletionClaimedAt.IsZero() {
		return TaskStatusPendingConfirmation
	}

	if isOverdue {
		return TaskStatusOverdue
	}

	if t.PercentDone > 0 {
		return TaskStatusOngoing
	}

	return TaskStatusPending
}

// CalculatePriorityScore computes the weighted priority score from urgency,
// importance (both 0-5 inputs), and deadline proximity, and buckets the
// result into Do Now / Do Soon / Do Later for the prioritized view.
func CalculatePriorityScore(urgency, importance int64, dueDate time.Time) (score float64, bucket string) {
	if urgency == 0 {
		urgency = priorityUnratedDefault
	}
	if importance == 0 {
		importance = priorityUnratedDefault
	}

	deadlineProximity := 0.0
	if !dueDate.IsZero() {
		remaining := time.Until(dueDate)
		switch {
		case remaining <= 0:
			deadlineProximity = 5
		case remaining >= priorityDeadlineWindow:
			deadlineProximity = 0
		default:
			deadlineProximity = 5 * (1 - float64(remaining)/float64(priorityDeadlineWindow))
		}
	}

	score = priorityWeightUrgency*float64(urgency) +
		priorityWeightImportance*float64(importance) +
		priorityWeightDeadlineProximity*deadlineProximity

	switch {
	case score >= 3.5:
		bucket = PriorityBucketDoNow
	case score >= 2:
		bucket = PriorityBucketDoSoon
	default:
		bucket = PriorityBucketDoLater
	}

	return score, bucket
}
