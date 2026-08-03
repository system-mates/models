package acceptance

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestDataEditorLifecycleWithoutUsers(t *testing.T) {
	mate := startMate(t, "dataeditor", "test/dataeditor-target")

	response := mate.request(t, http.MethodGet, "/dataeditor?model=dataeditor-target", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Data Editor") {
		t.Fatalf("data editor page: status=%d body=%s", response.StatusCode, body)
	}
	for _, expected := range []string{
		`href="/dataeditor?model=dataeditor-target"`,
		`name="key" value="target.level"`,
		`name="key" value="speed"`,
		`name="key" value="level"`,
		`name="key" value="on"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("data editor page does not contain %q: %s", expected, body)
		}
	}

	body = setDataValue(t, mate, body, "state", "acceptance.value", `{"enabled":true,"count":2}`)
	if !strings.Contains(body, `name="key" value="acceptance.value"`) || !strings.Contains(body, "enabled") {
		t.Fatalf("created state value is not rendered: %s", body)
	}
	body = setDataValue(t, mate, body, "state", "acceptance.value", "42")
	if !strings.Contains(body, `name="key" value="acceptance.value"`) || !strings.Contains(body, ">42</textarea>") {
		t.Fatalf("updated state value is not rendered: %s", body)
	}
	body = setDataValue(t, mate, body, "share", "acceptance.share", `["a","b"]`)
	if !strings.Contains(body, `name="key" value="acceptance.share"`) {
		t.Fatalf("created share value is not rendered: %s", body)
	}

	body = deleteDataValue(t, mate, body, "state", "acceptance.value")
	if strings.Contains(body, `name="key" value="acceptance.value"`) {
		t.Fatalf("deleted state value remains rendered: %s", body)
	}
	body = deleteDataValue(t, mate, body, "share", "acceptance.share")
	if strings.Contains(body, `name="key" value="acceptance.share"`) {
		t.Fatalf("deleted share value remains rendered: %s", body)
	}
}

func TestDataEditorRequiresAUserWhenUsersIsLoaded(t *testing.T) {
	mate := startMate(t, "dataeditor", "test/dataeditor-target", "users")

	response := mate.request(t, http.MethodGet, "/dataeditor?model=dataeditor-target", "", "")
	_ = responseBody(t, response)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("data editor without session: status=%d", response.StatusCode)
	}

	login(t, mate, adminUsername, adminPassword)
	response = mate.request(t, http.MethodGet, "/dataeditor?model=dataeditor-target", "", "")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "Data Editor") {
		t.Fatalf("authenticated data editor page: status=%d body=%s", response.StatusCode, body)
	}
	body = setDataValue(t, mate, body, "state", "acceptance.authenticated", "true")
	if !strings.Contains(body, `name="key" value="acceptance.authenticated"`) {
		t.Fatalf("authenticated state update is not rendered: %s", body)
	}
}

func setDataValue(t *testing.T, mate *mateInstance, page, kind, key, value string) string {
	t.Helper()
	form := url.Values{
		"csrf_token": {csrfToken(t, page)},
		"model":      {"dataeditor-target"},
		"kind":       {kind},
		"key":        {key},
		"value":      {value},
	}
	response := mate.request(t, http.MethodPost, "/dataeditor/entry/set", "application/x-www-form-urlencoded", form.Encode())
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("set %s value %q: status=%d body=%s", kind, key, response.StatusCode, body)
	}
	return body
}

func deleteDataValue(t *testing.T, mate *mateInstance, page, kind, key string) string {
	t.Helper()
	form := url.Values{
		"csrf_token": {csrfToken(t, page)},
		"model":      {"dataeditor-target"},
		"kind":       {kind},
		"key":        {key},
	}
	response := mate.request(t, http.MethodPost, "/dataeditor/entry/delete", "application/x-www-form-urlencoded", form.Encode())
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("delete %s value %q: status=%d body=%s", kind, key, response.StatusCode, body)
	}
	return body
}
