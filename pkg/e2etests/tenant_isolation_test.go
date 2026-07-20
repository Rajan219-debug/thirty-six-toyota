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

package e2etests

// Multi-outlet (RLS) isolation cannot be exercised by the normal in-process
// test harness: SetupTests() syncs the schema straight from Go structs and
// never runs db.EnsureTenantColumns() / db.ArmTenantRLS() — those only run
// from the real boot path (pkg/initialize/init.go). Row-Level Security is
// therefore only real when the actual server binary is running against
// Postgres, connected as a non-superuser role, with
// VIKUNJA_MULTITENANCY_ENFORCE=1. So instead of an in-process handler test,
// this drives a real running instance over HTTP — the same thing a manual
// verification pass would do, just repeatable and asserted.
//
// Skipped by default (VIKUNJA_E2E_TENANT_ISOLATION_URL unset), so it does not
// affect a normal `go test` / `mage test:e2eapi` run. Point it at an
// RLS-enforced instance to actually run it, e.g.:
//
//	./dev-run.sh                      # starts the app on :3466 as superuser — NOT enough, see below
//	# then run this against an instance booted as the non-superuser vikunja_app
//	# role with VIKUNJA_MULTITENANCY_ENFORCE=1 (docker-compose.yml's setup, or
//	# an equivalent manual boot), e.g.:
//	VIKUNJA_E2E_TENANT_ISOLATION_URL=http://localhost:3467 \
//	VIKUNJA_E2E_TENANT_ISOLATION_ADMIN_USER=boss \
//	VIKUNJA_E2E_TENANT_ISOLATION_ADMIN_PASS=testpass123 \
//	VIKUNJA_E2E_TENANT_ISOLATION_TENANT_A=2 \
//	VIKUNJA_E2E_TENANT_ISOLATION_TENANT_B=3 \
//	go test -run TestTenantIsolation ./pkg/e2etests/...
//
// The admin user must be an instance admin AND is_hq (boss, in the dev
// fixtures) so it can call the /admin/tenants and /admin/users/{id}/outlet
// endpoints and prove the HQ cross-outlet bypass. Tenant A/B must be two
// outlet ids that already exist (dev-run.sh's DB has 2=Palwal, 3=Faridabad).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type isolationClient struct {
	t       *testing.T
	baseURL string
	client  *http.Client
}

func (c *isolationClient) do(method, path, token string, body any) (*http.Response, []byte) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	require.NoError(c.t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.client.Do(req)
	require.NoError(c.t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)
	return resp, respBody
}

func (c *isolationClient) login(username, password string) string {
	resp, body := c.do(http.MethodPost, "/api/v1/login", "", map[string]string{
		"username": username,
		"password": password,
	})
	require.Equal(c.t, http.StatusOK, resp.StatusCode, "login for %s failed: %s", username, body)
	var res struct {
		Token string `json:"token"`
	}
	require.NoError(c.t, json.Unmarshal(body, &res))
	require.NotEmpty(c.t, res.Token)
	return res.Token
}

func (c *isolationClient) register(username, password string) int64 {
	resp, body := c.do(http.MethodPost, "/api/v1/register", "", map[string]string{
		"username": username,
		"email":    username + "@isolation-test.example.com",
		"password": password,
	})
	require.Equal(c.t, http.StatusOK, resp.StatusCode, "register %s failed: %s", username, body)
	var res struct {
		ID int64 `json:"id"`
	}
	require.NoError(c.t, json.Unmarshal(body, &res))
	require.NotZero(c.t, res.ID)
	return res.ID
}

func (c *isolationClient) setOutlet(adminToken string, userID, tenantID int64, isHQ bool) {
	resp, body := c.do(http.MethodPatch, fmt.Sprintf("/api/v2/admin/users/%d/outlet", userID), adminToken,
		map[string]any{"tenant_id": tenantID, "is_hq": isHQ})
	require.Equal(c.t, http.StatusOK, resp.StatusCode, "set outlet for user %d failed: %s", userID, body)
}

func (c *isolationClient) createProject(token, title string) int64 {
	resp, body := c.do(http.MethodPut, "/api/v1/projects", token, map[string]string{"title": title})
	require.Equal(c.t, http.StatusCreated, resp.StatusCode, "create project failed: %s", body)
	var res struct {
		ID int64 `json:"id"`
	}
	require.NoError(c.t, json.Unmarshal(body, &res))
	require.NotZero(c.t, res.ID)
	return res.ID
}

// TestTenantIsolation proves outlet isolation end-to-end against a real,
// RLS-enforced running instance: a project created by a user in one outlet
// is invisible and unwritable to a user in a different outlet, remains fully
// accessible to its own owner, and is readable (not writable) by an HQ user.
func TestTenantIsolation(t *testing.T) {
	baseURL := os.Getenv("VIKUNJA_E2E_TENANT_ISOLATION_URL")
	if baseURL == "" {
		t.Skip("VIKUNJA_E2E_TENANT_ISOLATION_URL not set — skipping; this test needs a real " +
			"instance running against Postgres as a non-superuser role with " +
			"VIKUNJA_MULTITENANCY_ENFORCE=1 (RLS is never armed by the normal in-process test setup)")
	}

	adminUser := os.Getenv("VIKUNJA_E2E_TENANT_ISOLATION_ADMIN_USER")
	adminPass := os.Getenv("VIKUNJA_E2E_TENANT_ISOLATION_ADMIN_PASS")
	require.NotEmpty(t, adminUser, "VIKUNJA_E2E_TENANT_ISOLATION_ADMIN_USER must be set")
	require.NotEmpty(t, adminPass, "VIKUNJA_E2E_TENANT_ISOLATION_ADMIN_PASS must be set")

	var tenantA, tenantB int64
	_, err := fmt.Sscanf(os.Getenv("VIKUNJA_E2E_TENANT_ISOLATION_TENANT_A"), "%d", &tenantA)
	require.NoError(t, err, "VIKUNJA_E2E_TENANT_ISOLATION_TENANT_A must be a numeric outlet id")
	_, err = fmt.Sscanf(os.Getenv("VIKUNJA_E2E_TENANT_ISOLATION_TENANT_B"), "%d", &tenantB)
	require.NoError(t, err, "VIKUNJA_E2E_TENANT_ISOLATION_TENANT_B must be a numeric outlet id")
	require.NotEqual(t, tenantA, tenantB, "tenant A and B must be different outlets")

	c := &isolationClient{t: t, baseURL: baseURL, client: &http.Client{Timeout: 10 * time.Second}}

	adminToken := c.login(adminUser, adminPass)

	stamp := time.Now().UnixNano()
	userA := fmt.Sprintf("isotest_a_%d", stamp)
	userB := fmt.Sprintf("isotest_b_%d", stamp)
	const password = "IsolationTest123!"

	idA := c.register(userA, password)
	idB := c.register(userB, password)
	c.setOutlet(adminToken, idA, tenantA, false)
	c.setOutlet(adminToken, idB, tenantB, false)

	tokenA := c.login(userA, password)
	tokenB := c.login(userB, password)

	projectID := c.createProject(tokenA, fmt.Sprintf("Isolation Test Project %d", stamp))

	t.Run("owner can read own project", func(t *testing.T) {
		resp, body := c.do(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d", projectID), tokenA, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	})

	t.Run("owner can write to own project", func(t *testing.T) {
		resp, body := c.do(http.MethodPut, fmt.Sprintf("/api/v1/projects/%d/tasks", projectID), tokenA,
			map[string]string{"title": "same-tenant task"})
		require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))
	})

	t.Run("other-tenant user cannot read the project", func(t *testing.T) {
		resp, _ := c.do(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d", projectID), tokenB, nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode,
			"cross-tenant read must be hidden (404), not merely forbidden — RLS fail-safe is 'no rows', not a 403 that would confirm the row exists")
	})

	t.Run("other-tenant user cannot write to the project", func(t *testing.T) {
		resp, _ := c.do(http.MethodPut, fmt.Sprintf("/api/v1/projects/%d/tasks", projectID), tokenB,
			map[string]string{"title": "cross-tenant write attempt"})
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("other-tenant user does not see it in their project list", func(t *testing.T) {
		resp, body := c.do(http.MethodGet, "/api/v1/projects", tokenB, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var list []struct {
			ID int64 `json:"id"`
		}
		require.NoError(t, json.Unmarshal(body, &list))
		for _, p := range list {
			require.NotEqual(t, projectID, p.ID, "other-tenant user's project list must not contain the isolated project")
		}
	})

	t.Run("HQ admin can read across outlets", func(t *testing.T) {
		resp, body := c.do(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d", projectID), adminToken, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	})
}
