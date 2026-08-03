package acceptance

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestArchViewsAndRememberedModelWithoutUsers(t *testing.T) {
	mate := startMateConfigured(t, archConfiguration, "arch", "dataeditor", "test/dataeditor-target")

	response := mate.request(t, http.MethodGet, "/arch/operations?model=dataeditor-target", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("operations view: status=%d body=%s", response.StatusCode, body)
	}
	for _, expected := range []string{
		"Operations View",
		`data-selected-model="dataeditor-target"`,
		`class="panel lifecycle-controls"`,
		"Runtime overview",
		"Reload selected model",
		"Restart selected model",
		"Restart application",
		"<mate-d2",
		"data-structure",
		"data-status",
		"data-events",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("operations view does not contain %q: %s", expected, body)
		}
	}

	response = mate.request(t, http.MethodGet, "/arch/architecture?model=dataeditor-target", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Architecture View") || !strings.Contains(body, `data-selected-model="dataeditor-target"`) {
		t.Fatalf("architecture view: status=%d body=%s", response.StatusCode, body)
	}
	if strings.Contains(body, `class="panel lifecycle-controls"`) || strings.Contains(body, "data-node-details") {
		t.Fatalf("architecture view contains operations controls: %s", body)
	}

	response = mate.request(t, http.MethodGet, "/arch/detail?model=dataeditor-target", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "target.level") || !strings.Contains(body, "level") {
		t.Fatalf("model detail: status=%d body=%s", response.StatusCode, body)
	}

	response = mate.request(t, http.MethodGet, "/dataeditor", "", "")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, `class="button primary" href="/dataeditor?model=dataeditor-target"`) {
		t.Fatalf("remembered model in Data Editor: status=%d body=%s", response.StatusCode, body)
	}

	for _, asset := range []string{"/arch/static/arch.css", "/arch/static/arch.js", "/arch/static/mate-d2.js"} {
		response = mate.request(t, http.MethodGet, asset, "", "")
		_ = responseBody(t, response)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("Arch asset %s: status=%d", asset, response.StatusCode)
		}
	}
}

func TestArchAuthorizationAndModelRestartWithUsers(t *testing.T) {
	mate := startMateConfigured(t, archConfiguration, "arch", "test/dataeditor-target", "users")

	response := mate.request(t, http.MethodGet, "/arch/operations?model=dataeditor-target", "", "")
	_ = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("operations view without session: status=%d", response.StatusCode)
	}

	form := url.Values{"model": {"dataeditor-target"}}
	response = mate.request(t, http.MethodPost, "/arch/lifecycle/restart-model", "application/x-www-form-urlencoded", form.Encode())
	_ = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("model restart without session: status=%d", response.StatusCode)
	}

	login(t, mate, adminUsername, adminPassword)
	response = mate.request(t, http.MethodGet, "/arch/operations?model=dataeditor-target", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authenticated operations view: status=%d body=%s", response.StatusCode, body)
	}

	form.Set("csrf_token", csrfToken(t, body))
	response = mate.request(t, http.MethodPost, "/arch/lifecycle/restart-model", "application/x-www-form-urlencoded", form.Encode())
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "accepted") {
		t.Fatalf("model restart: status=%d body=%s", response.StatusCode, body)
	}

	eventually(t, func() (bool, string) {
		response := mate.request(t, http.MethodGet, "/arch/detail?model=dataeditor-target", "", "")
		body := responseBody(t, response)
		return response.StatusCode == http.StatusOK && strings.Contains(body, `"key":"starts","type":"number","value":2`), body
	})
}

const archConfiguration = "administration_models: [arch]"
