package acceptance

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	adminUsername = "admin"
	adminPassword = "mate-acceptance-admin-password"
	csrfSecret    = "mate-acceptance-csrf-secret-at-least-32-bytes"
)

var (
	modelsRoot string
	mateBinary string
	buildRoot  string
)

var csrfInput = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

func TestMain(m *testing.M) {
	var err error
	modelsRoot, err = repositoryRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	mateBinary, buildRoot, err = resolveMateBinary(modelsRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	if buildRoot != "" {
		_ = os.RemoveAll(buildRoot)
	}
	os.Exit(code)
}

func repositoryRoot() (string, error) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("locate acceptance test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "system", "models")); err != nil {
		return "", fmt.Errorf("locate models repository: %w", err)
	}
	return root, nil
}

func resolveMateBinary(root string) (binary, temporaryRoot string, err error) {
	if configured := strings.TrimSpace(os.Getenv("MATE_BINARY")); configured != "" {
		binary, err = filepath.Abs(configured)
		if err != nil {
			return "", "", fmt.Errorf("resolve MATE_BINARY: %w", err)
		}
		if info, statErr := os.Stat(binary); statErr != nil || info.IsDir() {
			return "", "", fmt.Errorf("MATE_BINARY %q is not an executable file", binary)
		}
		return binary, "", nil
	}
	runtimeRoot := filepath.Join(filepath.Dir(root), "runtime")
	if _, err := os.Stat(filepath.Join(runtimeRoot, "go.mod")); err != nil {
		return "", "", fmt.Errorf("MATE_BINARY is not set and sibling runtime repository is unavailable")
	}
	temporaryRoot, err = os.MkdirTemp("", "mate-models-acceptance-build-")
	if err != nil {
		return "", "", err
	}
	binary = filepath.Join(temporaryRoot, "mate")
	command := exec.Command("go", "build", "-trimpath", "-o", binary, "./cmd/app")
	command.Dir = runtimeRoot
	command.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		_ = os.RemoveAll(temporaryRoot)
		return "", "", fmt.Errorf("build sibling Mate runtime: %w\n%s", buildErr, output)
	}
	return binary, temporaryRoot, nil
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(value)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

type mateInstance struct {
	baseURL     string
	dataPath    string
	client      *http.Client
	command     *exec.Cmd
	logs        *lockedBuffer
	done        chan error
	arguments   []string
	environment []string
}

func startMate(t *testing.T, models ...string) *mateInstance {
	return startMateConfigured(t, "", models...)
}

func startMateConfigured(t *testing.T, instanceConfiguration string, models ...string) *mateInstance {
	t.Helper()
	root := t.TempDir()
	modelsDir := filepath.Join(root, "models")
	if err := os.Mkdir(modelsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	withUsers := false
	for _, reference := range models {
		model, source, err := modelSource(reference)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(source); err != nil {
			t.Fatalf("model %q: %v", reference, err)
		}
		if err := copyTree(source, filepath.Join(modelsDir, model)); err != nil {
			t.Fatalf("copy model %q: %v", reference, err)
		}
		withUsers = withUsers || model == "users"
	}
	configuration := strings.TrimSpace(instanceConfiguration)
	if configuration != "" {
		configuration += "\n\n"
	}
	configuration += "logging:\n  console:\n    level: error\n  persistence:\n    enabled: true\n    level: info\n    retention: 7d\n"
	if err := os.WriteFile(filepath.Join(root, "mate.yaml"), []byte(configuration), 0o644); err != nil {
		t.Fatal(err)
	}
	address := availableAddress(t)
	arguments := []string{"-data-path", filepath.Join(root, "data"), "-http-listen", address}
	if withUsers {
		arguments = append(arguments, "-csrf-secret="+csrfSecret)
	}
	arguments = append(arguments, root)
	logs := &lockedBuffer{}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	instance := &mateInstance{
		baseURL:     "http://" + address,
		dataPath:    filepath.Join(root, "data"),
		client:      &http.Client{Jar: jar, Timeout: 3 * time.Second},
		logs:        logs,
		arguments:   append([]string(nil), arguments...),
		environment: append(os.Environ(), "MATE_ADMIN_USERNAME="+adminUsername, "MATE_ADMIN_PASSWORD="+adminPassword),
	}
	instance.startProcess(t)
	t.Cleanup(func() { instance.stop(t) })
	instance.waitUntilReady(t)
	return instance
}

func (m *mateInstance) startProcess(t *testing.T) {
	t.Helper()
	m.command = exec.Command(mateBinary, m.arguments...)
	m.command.Env = m.environment
	m.command.Stdout, m.command.Stderr = m.logs, m.logs
	if err := m.command.Start(); err != nil {
		t.Fatalf("start Mate: %v", err)
	}
	m.done = make(chan error, 1)
	go func(command *exec.Cmd, done chan<- error) { done <- command.Wait() }(m.command, m.done)
}

func (m *mateInstance) restart(t *testing.T) {
	t.Helper()
	m.stop(t)
	m.startProcess(t)
	m.waitUntilReady(t)
}

func modelSource(reference string) (name, source string, err error) {
	collection := "system"
	name = reference
	if prefix, remainder, found := strings.Cut(reference, "/"); found {
		collection, name = prefix, remainder
	}
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return "", "", fmt.Errorf("invalid model reference %q", reference)
	}
	switch collection {
	case "system":
		source = filepath.Join(modelsRoot, "system", "models", name)
	case "example":
		source = filepath.Join(modelsRoot, "example", "models", name)
	case "test":
		source = filepath.Join(modelsRoot, "tests", "models", name)
	default:
		return "", "", fmt.Errorf("unknown model collection %q in %q", collection, reference)
	}
	return name, source, nil
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		return closeErr
	})
}

func availableAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func (m *mateInstance) waitUntilReady(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-m.done:
			m.done <- err
			t.Fatalf("Mate exited during startup: %v\n%s", err, m.logs.String())
		default:
		}
		response, err := m.client.Get(m.baseURL + "/assets/static/mate-layout.css")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("Mate did not become ready\n%s", m.logs.String())
}

func (m *mateInstance) stop(t *testing.T) {
	t.Helper()
	if m.command == nil || m.command.Process == nil {
		return
	}
	_ = m.command.Process.Signal(os.Interrupt)
	select {
	case <-m.done:
	case <-time.After(3 * time.Second):
		_ = m.command.Process.Kill()
		select {
		case <-m.done:
		case <-time.After(time.Second):
		}
	}
	if t.Failed() {
		t.Logf("Mate process output:\n%s", m.logs.String())
	}
	m.command = nil
	m.done = nil
}

func (m *mateInstance) request(t *testing.T, method, path, contentType, body string) *http.Response {
	return m.requestWithHeaders(t, method, path, contentType, body, nil)
}

func (m *mateInstance) requestWithHeaders(t *testing.T, method, path, contentType, body string, headers map[string]string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, m.baseURL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Origin", m.baseURL)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := m.client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", method, path, err, m.logs.String())
	}
	return response
}

func csrfToken(t *testing.T, body string) string {
	t.Helper()
	match := csrfInput.FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatal("response does not contain a CSRF token")
	}
	return match[1]
}

func (m *mateInstance) replaceCookieJar(t *testing.T) {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	m.client.Jar = jar
}

func responseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	value, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(value)
}

func eventually(t *testing.T, check func() (bool, string)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var detail string
	for {
		ok, current := check()
		if ok {
			return
		}
		detail = current
		select {
		case <-ctx.Done():
			t.Fatalf("condition was not met: %s", detail)
		case <-time.After(25 * time.Millisecond):
		}
	}
}
