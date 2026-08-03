package acceptance

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestLogsWithoutUsers(t *testing.T) {
	mate := startMate(t, "logs")

	response := mate.request(t, http.MethodGet, "/logs", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Runtime logs") {
		t.Fatalf("initial logs page: status=%d body=%s", response.StatusCode, body)
	}

	eventually(t, func() (bool, string) {
		response := mate.request(t, http.MethodGet, "/logs?q=coordinator_activation", "", "")
		body := responseBody(t, response)
		return response.StatusCode == http.StatusOK && strings.Contains(body, "coordinator_activation"), body
	})

	token := csrfToken(t, body)
	form := url.Values{"csrf_token": {token}, "override_level": {"debug"}, "duration": {"5m"}}
	response = mate.request(t, http.MethodPost, "/logs/level", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "effective debug") {
		t.Fatalf("set log level: status=%d body=%s", response.StatusCode, body)
	}

	token = csrfToken(t, body)
	form = url.Values{"csrf_token": {token}}
	response = mate.request(t, http.MethodPost, "/logs/level/cancel", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "effective info") {
		t.Fatalf("cancel log level: status=%d body=%s", response.StatusCode, body)
	}
}

func TestLogsWithUsers(t *testing.T) {
	mate := startMate(t, "logs", "users")

	response := mate.request(t, http.MethodGet, "/logs", "", "")
	_ = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("logs without session: status=%d", response.StatusCode)
	}

	login := `{"username":"` + adminUsername + `","password":"` + adminPassword + `"}`
	response = mate.request(t, http.MethodPost, "/_auth/login", "application/json", login)
	body := responseBody(t, response)
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("admin login: status=%d body=%s", response.StatusCode, body)
	}

	response = mate.request(t, http.MethodGet, "/logs", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Runtime logs") {
		t.Fatalf("authenticated logs page: status=%d body=%s", response.StatusCode, body)
	}
}
