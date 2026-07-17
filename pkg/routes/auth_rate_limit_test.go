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

package routes

import (
	"testing"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/modules/keyvalue"

	"github.com/stretchr/testify/assert"
)

func TestClassifyAuthRoute(t *testing.T) {
	assert.Equal(t, authRouteLogin, classifyAuthRoute("/api/v1/login"))
	assert.Equal(t, authRouteLogin, classifyAuthRoute("/api/v2/login"))
	assert.Equal(t, authRouteAttempt, classifyAuthRoute("/api/v1/user/password/token"))
	assert.Equal(t, authRoutePlain, classifyAuthRoute("/api/v1/register"))
	assert.Equal(t, authRoutePlain, classifyAuthRoute("/api/v2/user/token/refresh"))
	assert.Equal(t, authRoutePlain, classifyAuthRoute("/api/v1/shares/abc123/auth"))
	assert.Equal(t, authRoutePlain, classifyAuthRoute("/api/v1/auth/openid/google/callback"))
	assert.Equal(t, authRouteNone, classifyAuthRoute("/api/v2/tasks/prioritized"))
	assert.Equal(t, authRouteNone, classifyAuthRoute("/api/v2/docs"))
}

func TestCalcAuthBackoff(t *testing.T) {
	config.InitDefaultConfig()
	// defaults: base 2s, max 900s
	assert.Equal(t, 2*time.Second, calcAuthBackoff(1))
	assert.Equal(t, 4*time.Second, calcAuthBackoff(2))
	assert.Equal(t, 8*time.Second, calcAuthBackoff(3))
	assert.Equal(t, 900*time.Second, calcAuthBackoff(10)) // 1024s capped
	assert.Equal(t, 900*time.Second, calcAuthBackoff(63)) // shift-overflow guard
}

func TestAuthAccountBackoffStateMachine(t *testing.T) {
	config.InitDefaultConfig()
	keyvalue.InitStorage()
	const account = "brute@example.com"
	clearAuthFailures(account)

	// The free attempts (default 3) never block.
	for i := 0; i < 3; i++ {
		recordAuthFailure(account)
		_, blocked := authAccountBlocked(account)
		assert.False(t, blocked, "failure %d should still be free", i+1)
	}

	// The 4th failure starts the backoff window.
	recordAuthFailure(account)
	retryAfter, blocked := authAccountBlocked(account)
	assert.True(t, blocked)
	assert.Greater(t, retryAfter, time.Duration(0))
	assert.LessOrEqual(t, retryAfter, 2*time.Second)

	// Further failures grow the window exponentially.
	recordAuthFailure(account)
	retryAfter, blocked = authAccountBlocked(account)
	assert.True(t, blocked)
	assert.Greater(t, retryAfter, 2*time.Second)
	assert.LessOrEqual(t, retryAfter, 4*time.Second)

	// A successful login clears everything.
	clearAuthFailures(account)
	_, blocked = authAccountBlocked(account)
	assert.False(t, blocked)

	// A stale record (past the failure-reset horizon) counts as absent.
	assert.NoError(t, keyvalue.Put(authFailureKey(account), authFailureState{
		Fails:        50,
		BackoffUntil: time.Now().Add(time.Hour),
		StaleAt:      time.Now().Add(-time.Minute),
	}))
	_, blocked = authAccountBlocked(account)
	assert.False(t, blocked, "expired failure state must be forgotten")
	clearAuthFailures(account)
}
