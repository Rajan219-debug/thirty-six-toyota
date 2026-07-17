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
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/modules/keyvalue"

	"github.com/labstack/echo/v5"
	"github.com/ulule/limiter/v3"
)

// authRouteKind classifies how an unauthenticated auth endpoint is protected.
type authRouteKind int

const (
	// Not an auth route.
	authRouteNone authRouteKind = iota
	// Strict per-IP window only.
	authRoutePlain
	// Per-IP window + per-account backoff: failures count, a successful login
	// clears the account's failure state.
	authRouteLogin
	// Per-IP window + per-account backoff where every request counts: a
	// "successful" password-reset request still sends a mail, so success must
	// not refill the budget (mailbox-flooding protection).
	authRouteAttempt
)

// classifyAuthRoute maps a request path to its auth-route kind. Works for both
// API versions: the auth endpoints share paths under /api/v1 and /api/v2.
func classifyAuthRoute(path string) authRouteKind {
	p := path
	for _, prefix := range []string{"/api/v1", "/api/v2"} {
		if strings.HasPrefix(p, prefix) {
			p = strings.TrimPrefix(p, prefix)
			break
		}
	}

	switch p {
	case "/login":
		return authRouteLogin
	case "/user/password/token":
		return authRouteAttempt
	case "/register", "/user/password/reset", "/user/confirm", "/user/token/refresh", "/oauth/token":
		return authRoutePlain
	}
	if strings.HasPrefix(p, "/auth/openid/") {
		return authRoutePlain
	}
	if strings.HasPrefix(p, "/shares/") && strings.HasSuffix(p, "/auth") {
		return authRoutePlain
	}
	return authRouteNone
}

// authIPRateLimiter is shared between the v1 and v2 route groups so both API
// versions draw from one per-IP budget. Route setup is single-threaded.
var authIPRateLimiter *limiter.Limiter

func getAuthIPRateLimiter() *limiter.Limiter {
	if authIPRateLimiter == nil {
		authIPRateLimiter = createRateLimiter(limiter.Rate{
			Period: config.RateLimitNoAuthRoutesPeriod.GetDuration() * time.Second,
			Limit:  config.RateLimitNoAuthRoutesLimit.GetInt64(),
		})
	}
	return authIPRateLimiter
}

// --- per-account failure state, stored via keyvalue (memory or redis) ---

type authFailureState struct {
	Fails        int64
	BackoffUntil time.Time
	StaleAt      time.Time
}

func authFailureKey(account string) string {
	return "authfail:" + account
}

// getAuthFailureState returns the account's failure state, treating an expired
// record as absent so old failures are forgotten, never punished forever.
func getAuthFailureState(account string) (s authFailureState, exists bool) {
	found, err := keyvalue.GetWithValue(authFailureKey(account), &s)
	if err != nil {
		// Fail open: a broken keyvalue store must not lock everyone out.
		log.Errorf("auth backoff: reading state for %q: %v", account, err)
		return authFailureState{}, false
	}
	if !found || time.Now().After(s.StaleAt) {
		return authFailureState{}, false
	}
	return s, true
}

// authAccountBlocked reports whether the account is inside a backoff window.
func authAccountBlocked(account string) (retryAfter time.Duration, blocked bool) {
	s, exists := getAuthFailureState(account)
	if !exists {
		return 0, false
	}
	if wait := time.Until(s.BackoffUntil); wait > 0 {
		return wait, true
	}
	return 0, false
}

// calcAuthBackoff returns the backoff for the n-th failure past the free
// attempts: base * 2^(n-1), capped at max — a slowdown, never a lockout.
func calcAuthBackoff(over int64) time.Duration {
	base := config.RateLimitAuthBackoffBase.GetDuration() * time.Second
	maxBackoff := config.RateLimitAuthBackoffMax.GetDuration() * time.Second
	if over > 20 { // 2^20 * base is already way past any sane cap
		over = 20
	}
	d := base << uint(over-1)
	if d <= 0 || d > maxBackoff {
		return maxBackoff
	}
	return d
}

func recordAuthFailure(account string) {
	s, _ := getAuthFailureState(account)
	s.Fails++
	if over := s.Fails - config.RateLimitAuthFreeAttempts.GetInt64(); over > 0 {
		s.BackoffUntil = time.Now().Add(calcAuthBackoff(over))
	}
	s.StaleAt = time.Now().Add(config.RateLimitAuthFailureReset.GetDuration() * time.Second)
	if err := keyvalue.Put(authFailureKey(account), s); err != nil {
		log.Errorf("auth backoff: storing state for %q: %v", account, err)
	}
}

func clearAuthFailures(account string) {
	if err := keyvalue.Del(authFailureKey(account)); err != nil {
		log.Errorf("auth backoff: clearing state for %q: %v", account, err)
	}
}

// extractAccountIdentifier peeks the JSON body for the targeted account
// (login: username, password reset: email) and restores the body for the
// handler. Returns "" when there is nothing to key on.
func extractAccountIdentifier(c *echo.Context) string {
	req := c.Request()
	if req.Body == nil {
		return ""
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return ""
	}
	req.Body = io.NopCloser(bytes.NewReader(body))

	var creds struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	_ = json.Unmarshal(body, &creds) // not JSON = nothing to key on
	id := creds.Username
	if id == "" {
		id = creds.Email
	}
	id = strings.ToLower(strings.TrimSpace(id))
	// Usernames and emails are capped at 250 by the account schema; don't let
	// an oversized junk value become an unbounded keyvalue key.
	if len(id) > 250 {
		id = id[:250]
	}
	return id
}

// AuthRouteRateLimit protects the unauthenticated auth endpoints (login,
// register, password reset, …) with a strict per-IP window plus, for
// credential endpoints, per-account exponential backoff on failures. The
// account dimension catches distributed attacks on one account that rotate
// IPs; the backoff always expires, so there is no hard lockout to abuse for
// denial of service against a victim's account.
//
// limitUnmatched controls what happens to paths that are not recognized auth
// routes: the v1 unauthenticated group passes true (everything in it deserves
// the strict limit), the v2 group passes false (its group holds the whole API,
// so only matched auth paths are limited here).
func AuthRouteRateLimit(ipLimiter *limiter.Limiter, limitUnmatched bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			kind := classifyAuthRoute(c.Request().URL.Path)
			if kind == authRouteNone {
				if !limitUnmatched {
					return next(c)
				}
				kind = authRoutePlain
			}

			// IP window first: a flood gets cut off before we touch bodies
			// or account state.
			if err := enforceRateLimit(c, ipLimiter, c.RealIP()); err != nil {
				return err
			}

			var account string
			if (kind == authRouteLogin || kind == authRouteAttempt) && c.Request().Method == http.MethodPost {
				account = extractAccountIdentifier(c)
			}
			if account != "" {
				if retryAfter, blocked := authAccountBlocked(account); blocked {
					c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(math.Ceil(retryAfter.Seconds())), 10))
					log.Infof("Auth backoff active for account %q from %s on %s", account, c.RealIP(), c.Request().URL.Path)
					return echo.NewHTTPError(http.StatusTooManyRequests, "Too many failed attempts, please wait before retrying.")
				}
			}

			err := next(c)

			if account != "" {
				// v1 handlers surface failures as returned errors; huma (v2)
				// writes the error response itself and returns nil, so check
				// the committed status too.
				failed := err != nil
				if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed && r.Status >= http.StatusBadRequest {
					failed = true
				}
				switch {
				case kind == authRouteAttempt:
					recordAuthFailure(account)
				case failed:
					recordAuthFailure(account)
				default:
					clearAuthFailures(account)
				}
			}
			return err
		}
	}
}
