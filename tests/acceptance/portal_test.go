package acceptance

import (
	"net/http"
	"strings"
	"testing"
)

func TestPortalWithoutUsers(t *testing.T) {
	mate := startMate(t, "portal", "logs")

	response := mate.request(t, http.MethodGet, "/portal", "", "")
	body := responseBody(t, response)
	for _, expected := range []string{
		"Mate Portal",
		`<h2>logs</h2>`,
		`href="/logs" data-portal-path="/logs">Logs</a>`,
		`data-portal-path="/users/login" hidden>Sign in</button>`,
		`class="portal-content"`,
	} {
		if response.StatusCode != http.StatusOK || !strings.Contains(body, expected) {
			t.Fatalf("portal missing %q: status=%d body=%s", expected, response.StatusCode, body)
		}
	}

	response = mate.request(t, http.MethodGet, "/portal/language-reference", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Mate YAML Language") {
		t.Fatalf("language reference: status=%d body=%s", response.StatusCode, body)
	}

	response = mate.request(t, http.MethodGet, "/portal/static/portal.css", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, ".portal-shell") {
		t.Fatalf("portal stylesheet: status=%d body=%s", response.StatusCode, body)
	}
}

func TestPortalCanOwnRootRoute(t *testing.T) {
	mate := startMateConfigured(t, "root_model: portal", "portal")

	for _, path := range []string{"/", "/portal"} {
		response := mate.request(t, http.MethodGet, path, "", "")
		body := responseBody(t, response)
		if response.StatusCode != http.StatusOK || !strings.Contains(body, "Mate Portal") {
			t.Fatalf("portal route %s: status=%d body=%s", path, response.StatusCode, body)
		}
	}
}

func TestPortalRecognizesUsersModel(t *testing.T) {
	mate := startMate(t, "portal", "users")

	response := mate.request(t, http.MethodGet, "/portal", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("portal with Users: status=%d body=%s", response.StatusCode, body)
	}
	for _, expected := range []string{
		`<h2>users</h2>`,
		`href="/users" data-portal-path="/users">Users</a>`,
		`data-portal-path="/users/login">Sign in</button>`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("portal with Users missing %q: %s", expected, body)
		}
	}
	if strings.Contains(body, `data-portal-path="/users/login" hidden`) {
		t.Fatal("portal hid sign-in control while Users was loaded")
	}
}
