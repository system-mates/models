package acceptance

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

const (
	operatorUsername = "operator.user"
	operatorPassword = "mate-operator-password"
	updatedPassword  = "mate-updated-operator-password"
)

var (
	operatorIDPattern  = regexp.MustCompile(`(?s)<h2>operator\.user</h2>.*?name="user_id" value="([^"]+)"`)
	serviceIDPattern   = regexp.MustCompile(`(?s)<h3>acceptance-service</h3>.*?name="service_id" value="([^"]+)"`)
	tokenIDPattern     = regexp.MustCompile(`(?s)<strong>acceptance-token</strong>.*?name="token_id" value="([^"]+)"`)
	issuedTokenPattern = regexp.MustCompile(`<code>([A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)</code>`)
)

func TestUsersAdministrationLifecycle(t *testing.T) {
	mate := startMate(t, "users")

	response := mate.request(t, http.MethodGet, "/users/login", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Sign in") {
		t.Fatalf("login page: status=%d body=%s", response.StatusCode, body)
	}

	response = mate.request(t, http.MethodGet, "/users", "", "")
	_ = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("administration without session: status=%d", response.StatusCode)
	}

	response = mate.request(t, http.MethodPost, "/_auth/login", "application/json", `{"username":"admin","password":"wrong-password"}`)
	body = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized || !strings.Contains(body, "invalid_credentials") {
		t.Fatalf("invalid login: status=%d body=%s", response.StatusCode, body)
	}

	login(t, mate, adminUsername, adminPassword)
	response = mate.request(t, http.MethodGet, "/users", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Manage users") || !strings.Contains(body, ">admin</h2>") {
		t.Fatalf("admin page: status=%d body=%s", response.StatusCode, body)
	}

	form := url.Values{
		"csrf_token":   {csrfToken(t, body)},
		"username":     {operatorUsername},
		"display_name": {"Acceptance Operator"},
		"roles":        {"operator"},
		"password":     {operatorPassword},
	}
	response = mate.request(t, http.MethodPost, "/users/create", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, ">"+operatorUsername+"</h2>") || !strings.Contains(body, "Acceptance Operator") {
		t.Fatalf("create user: status=%d body=%s", response.StatusCode, body)
	}
	operatorID := capture(t, operatorIDPattern, body, "operator user ID")

	form = url.Values{
		"csrf_token": {csrfToken(t, body)},
		"user_id":    {operatorID},
		"roles":      {"operator,auditor"},
	}
	response = mate.request(t, http.MethodPost, "/users/roles", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, `value="operator,auditor"`) {
		t.Fatalf("update roles: status=%d body=%s", response.StatusCode, body)
	}

	form = url.Values{
		"csrf_token": {csrfToken(t, body)},
		"user_id":    {operatorID},
		"password":   {updatedPassword},
	}
	response = mate.request(t, http.MethodPost, "/users/password", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("reset password: status=%d body=%s", response.StatusCode, body)
	}

	token := csrfToken(t, body)
	response = mate.requestWithHeaders(t, http.MethodPost, "/_auth/logout", "application/json", `{}`, map[string]string{"X-CSRF-Token": token})
	body = responseBody(t, response)
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: status=%d body=%s", response.StatusCode, body)
	}
	response = mate.request(t, http.MethodGet, "/users", "", "")
	_ = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("administration after logout: status=%d", response.StatusCode)
	}

	mate.replaceCookieJar(t)
	response = mate.request(t, http.MethodPost, "/_auth/login", "application/json", `{"username":"operator.user","password":"mate-operator-password"}`)
	body = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized || !strings.Contains(body, "invalid_credentials") {
		t.Fatalf("old password after reset: status=%d body=%s", response.StatusCode, body)
	}
	login(t, mate, operatorUsername, updatedPassword)
	response = mate.request(t, http.MethodGet, "/users", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusForbidden || !strings.Contains(body, "forbidden") {
		t.Fatalf("non-admin administration: status=%d body=%s", response.StatusCode, body)
	}
}

func TestUsersServiceBearerTokenLifecycle(t *testing.T) {
	mate := startMate(t, "users", "test/auth-target")
	login(t, mate, adminUsername, adminPassword)

	response := mate.request(t, http.MethodGet, "/users", "", "")
	body := responseBody(t, response)
	form := url.Values{
		"csrf_token":  {csrfToken(t, body)},
		"name":        {"acceptance-service"},
		"description": {"Acceptance service account"},
		"roles":       {"operator"},
	}
	response = mate.request(t, http.MethodPost, "/users/services/create", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Acceptance service account") {
		t.Fatalf("create service: status=%d body=%s", response.StatusCode, body)
	}
	serviceID := capture(t, serviceIDPattern, body, "service ID")

	form = url.Values{
		"csrf_token":     {csrfToken(t, body)},
		"principal_kind": {"service"},
		"principal_id":   {serviceID},
		"token_name":     {"acceptance-token"},
		"expires_at":     {""},
	}
	response = mate.request(t, http.MethodPost, "/users/tokens/create", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Copy bearer token now") {
		t.Fatalf("create token: status=%d body=%s", response.StatusCode, body)
	}
	bearerToken := capture(t, issuedTokenPattern, body, "issued bearer token")
	tokenID := capture(t, tokenIDPattern, body, "token ID")

	adminJar := mate.client.Jar
	mate.client.Jar = nil
	response = mate.requestWithHeaders(t, http.MethodGet, "/auth-target/secured", "", "", map[string]string{"Authorization": "Bearer " + bearerToken})
	securedBody := responseBody(t, response)
	mate.client.Jar = adminJar
	if response.StatusCode != http.StatusOK || !strings.Contains(securedBody, `"kind":"service"`) || !strings.Contains(securedBody, `"name":"acceptance-service"`) {
		t.Fatalf("service bearer request: status=%d body=%s", response.StatusCode, securedBody)
	}

	form = url.Values{"csrf_token": {csrfToken(t, body)}, "token_id": {tokenID}}
	response = mate.request(t, http.MethodPost, "/users/tokens/revoke", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "revoked") {
		t.Fatalf("revoke token: status=%d body=%s", response.StatusCode, body)
	}

	mate.client.Jar = nil
	response = mate.requestWithHeaders(t, http.MethodGet, "/auth-target/secured", "", "", map[string]string{"Authorization": "Bearer " + bearerToken})
	_ = responseBody(t, response)
	mate.client.Jar = adminJar
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked bearer token: status=%d", response.StatusCode)
	}
}

func TestUsersSessionPersistsAcrossMateRestart(t *testing.T) {
	mate := startMate(t, "users")
	login(t, mate, adminUsername, adminPassword)

	response := mate.request(t, http.MethodGet, "/users", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, ">admin</h2>") {
		t.Fatalf("administration before restart: status=%d body=%s", response.StatusCode, body)
	}

	mate.restart(t)
	response = mate.request(t, http.MethodGet, "/users", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, ">admin</h2>") {
		t.Fatalf("persisted session after restart: status=%d body=%s", response.StatusCode, body)
	}
}

func TestUsersExpiredSessionsAreRejected(t *testing.T) {
	for name, update := range map[string]string{
		"idle":     "UPDATE sessions SET idle_expires_at='2000-01-01T00:00:00.000Z'",
		"absolute": "UPDATE sessions SET absolute_expires_at='2000-01-01T00:00:00.000Z'",
	} {
		t.Run(name, func(t *testing.T) {
			mate := startMate(t, "users")
			login(t, mate, adminUsername, adminPassword)
			mate.stop(t)

			database, err := sql.Open("sqlite", filepath.Join(mate.dataPath, "models", "users", "model.db"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := database.Exec(update); err != nil {
				_ = database.Close()
				t.Fatal(err)
			}
			if err := database.Close(); err != nil {
				t.Fatal(err)
			}

			mate.startProcess(t)
			mate.waitUntilReady(t)
			response := mate.request(t, http.MethodGet, "/users", "", "")
			body := responseBody(t, response)
			if response.StatusCode != http.StatusUnauthorized || !strings.Contains(body, "authentication_required") {
				t.Fatalf("expired %s session: status=%d body=%s", name, response.StatusCode, body)
			}
		})
	}
}

func capture(t *testing.T, pattern *regexp.Regexp, body, description string) string {
	t.Helper()
	match := pattern.FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("response does not contain %s: %s", description, body)
	}
	return match[1]
}

func login(t *testing.T, mate *mateInstance, username, password string) {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	response := mate.request(t, http.MethodPost, "/_auth/login", "application/json", string(payload))
	body := responseBody(t, response)
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("login %q: status=%d body=%s", username, response.StatusCode, body)
	}
}
