package release

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type authTransport func(*http.Request) (*http.Response, error)

func (f authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func authResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func isolateReleaseAuth(t *testing.T, transport authTransport) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", dir)
	t.Setenv("GH_CONFIG_DIR", dir)
	oldAPI, oldClient := apiBaseURL, httpClient
	apiBaseURL = "https://api.github.com"
	httpClient = &http.Client{Transport: transport, CheckRedirect: checkRedirect}
	t.Cleanup(func() { apiBaseURL, httpClient = oldAPI, oldClient })
	return dir
}

func TestMain(m *testing.M) {
	if mode := os.Getenv("DEVRITES_RELEASE_AUTH_HELPER"); mode != "" {
		if strings.Join(os.Args[1:], " ") != "auth token --hostname github.com" ||
			os.Getenv("GH_PROMPT_DISABLED") != "1" || os.Getenv("GIT_TERMINAL_PROMPT") != "0" {
			os.Exit(91)
		}
		var input [1]byte
		if n, err := os.Stdin.Read(input[:]); n != 0 || err != io.EOF {
			os.Exit(92)
		}
		switch mode {
		case "saved":
			fmt.Fprintln(os.Stdout, "saved-token")
			fmt.Fprint(os.Stderr, "private-helper-stderr")
		case "failure":
			fmt.Fprint(os.Stderr, "private-helper-stderr")
			os.Exit(1)
		case "partial":
			fmt.Fprint(os.Stdout, "partial-secret")
			os.Exit(1)
		case "empty":
			fmt.Fprintln(os.Stdout)
		case "timeout":
			time.Sleep(time.Minute)
		case "overlong":
			fmt.Fprint(os.Stdout, strings.Repeat("x", 4097))
		case "flood":
			chunk := strings.Repeat("x", 4096)
			for {
				if _, err := io.WriteString(os.Stdout, chunk); err != nil {
					os.Exit(1)
				}
			}
		case "marker":
			if err := os.WriteFile(os.Getenv("AUTH_TEST_MARKER"), []byte("called"), 0o600); err != nil {
				os.Exit(93)
			}
			fmt.Fprint(os.Stdout, "helper-token")
		default:
			os.Exit(94)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func fakeAuthCLI(t *testing.T, dir, mode string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = source.Close() }()
	name := "gh"
	if runtime.GOOS == "windows" {
		name += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	target, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	t.Setenv("DEVRITES_RELEASE_AUTH_HELPER", mode)
}

func TestCredentialOutputLimit(t *testing.T) {
	var output credentialOutput
	full := strings.Repeat("x", len(output.data))
	if n, err := io.WriteString(&output, full); n != len(full) || err != nil {
		t.Fatalf("exact limit write = %d, %v", n, err)
	}
	if n, err := io.WriteString(&output, "private-overflow"); n != 0 || err == nil || strings.Contains(err.Error(), "private-overflow") {
		t.Fatalf("overflow write = %d, %v", n, err)
	}
	if output.n != len(output.data) {
		t.Fatal("overflow changed captured length")
	}
}

func TestLatestCredentialPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name        string
		ghToken     string
		githubToken string
		want        string
	}{
		{name: "GH_TOKEN wins", ghToken: "first-token", githubToken: "second-token", want: "first-token"},
		{name: "GITHUB_TOKEN fallback", githubToken: "second-token", want: "second-token"},
		{name: "missing gh remains anonymous"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
				calls++
				want := ""
				if tc.want != "" {
					want = "Bearer " + tc.want
				}
				if req.Header.Get("Authorization") != want {
					t.Error("unexpected credential selection")
				}
				if req.URL.String() != "https://api.github.com/repos/owner/repo/releases/latest" {
					t.Errorf("unexpected endpoint: %s", req.URL)
				}
				return authResponse(req, http.StatusOK, `{"tag_name":"v7.3.3"}`), nil
			})
			t.Setenv("GH_TOKEN", tc.ghToken)
			t.Setenv("GITHUB_TOKEN", tc.githubToken)
			if tag, err := Latest(context.Background(), "owner/repo"); err != nil || tag != "v7.3.3" {
				t.Fatalf("Latest = %q, %v", tag, err)
			}
			if calls != 1 {
				t.Fatalf("requests = %d, want 1", calls)
			}
		})
	}
}

func TestLatestHeadlessGHCredential(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode string
		want string
	}{
		{name: "saved login", mode: "saved", want: "Bearer saved-token"},
		{name: "not logged in", mode: "failure"},
		{name: "failed helper discards partial token", mode: "partial"},
		{name: "empty credential", mode: "empty"},
		{name: "bounded helper timeout", mode: "timeout"},
		{name: "overlong helper discards entire token", mode: "overlong"},
		{name: "streaming helper bounded", mode: "flood"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			dir := isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("Authorization") != tc.want {
					t.Error("unexpected helper credential")
				}
				return authResponse(req, http.StatusOK, `{"tag_name":"v7.3.3"}`), nil
			})
			fakeAuthCLI(t, dir, tc.mode)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			start := time.Now()
			if tag, err := Latest(ctx, "owner/repo"); err != nil || tag != "v7.3.3" {
				t.Fatalf("Latest = %q, %v", tag, err)
			}
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Fatalf("helper exceeded bounded lookup time: %s", elapsed)
			}
			if calls != 1 {
				t.Fatalf("requests = %d, want 1", calls)
			}
		})
	}
}

func TestLatestEnvironmentSkipsGH(t *testing.T) {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		t.Run(name, func(t *testing.T) {
			dir := isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "Bearer env-token" {
					t.Error("environment token not used")
				}
				return authResponse(req, http.StatusOK, `{"tag_name":"v7.3.3"}`), nil
			})
			marker := filepath.Join(dir, "called")
			t.Setenv("AUTH_TEST_MARKER", marker)
			fakeAuthCLI(t, dir, "marker")
			t.Setenv(name, "env-token")
			if _, err := Latest(context.Background(), "owner/repo"); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("gh invoked despite environment credential: %v", err)
			}
		})
	}
}

func TestLatestInvalidCredentialDoesNotFallBack(t *testing.T) {
	isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
		t.Fatal("invalid credential reached transport")
		return nil, nil
	})
	t.Setenv("GH_TOKEN", "private-token\nheader-injection")
	t.Setenv("GITHUB_TOKEN", "otherwise-valid")
	_, err := Latest(context.Background(), "owner/repo")
	if err == nil || !strings.Contains(err.Error(), "invalid GitHub credential") || strings.Contains(err.Error(), "private-token") {
		t.Fatalf("unsafe or missing credential error: %v", err)
	}
}

func TestLatestAuthDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		headers   http.Header
		want      []string
		forbidden []string
	}{
		{name: "invalid token", status: 401, want: []string{"HTTP 401 Unauthorized", "GH_TOKEN/GITHUB_TOKEN", "gh auth login"}},
		{name: "forbidden", status: 403, want: []string{"HTTP 403 Forbidden", "gh auth login"}},
		{name: "primary limit", status: 403, headers: http.Header{"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {"1700000000"}}, want: []string{"rate limit reached", "2023-11-14T22:13:20Z"}},
		{name: "secondary limit", status: 429, headers: http.Header{"Retry-After": {"60"}}, want: []string{"HTTP 429 Too Many Requests", "retry after 60 seconds"}},
		{name: "retry date", status: 429, headers: http.Header{"Retry-After": {"Wed, 21 Oct 2015 07:28:00 GMT"}}, want: []string{"retry after 2015-10-21T07:28:00Z"}},
		{name: "untrusted headers", status: 403, headers: http.Header{"X-Ratelimit-Reset": {"secret-header"}, "Retry-After": {"secret-header"}}, forbidden: []string{"secret-header", "rate limit resets", "retry after"}},
		{name: "out of range headers", status: 429, headers: http.Header{"X-Ratelimit-Reset": {"253402300800"}, "Retry-After": {"18446744073709551616"}}, forbidden: []string{"253402300800", "18446744073709551616", "rate limit resets", "retry after"}},
		{name: "negative headers", status: 429, headers: http.Header{"X-Ratelimit-Reset": {"-1"}, "Retry-After": {"-2"}}, forbidden: []string{"rate limit resets", "retry after"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("Authorization") != "Bearer private-token" {
					t.Error("explicit credential missing")
				}
				resp := authResponse(req, tc.status, "private-token private-response-body")
				resp.Status = "private-status-text"
				resp.Header = tc.headers
				return resp, nil
			})
			t.Setenv("GH_TOKEN", "private-token")
			_, err := Latest(context.Background(), "owner/repo")
			if err == nil {
				t.Fatal("expected HTTP error")
			}
			for _, text := range tc.want {
				if !strings.Contains(err.Error(), text) {
					t.Errorf("error %q missing %q", err, text)
				}
			}
			for _, text := range append(tc.forbidden, "private-token", "private-response-body", "private-status-text") {
				if strings.Contains(err.Error(), text) {
					t.Errorf("error contains untrusted text %q", text)
				}
			}
			if calls != 1 {
				t.Fatalf("requests = %d; authentication failures must not retry anonymously", calls)
			}
		})
	}
}

func TestGitHubCredentialOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin string
		want   bool
	}{
		{origin: "https://api.github.com", want: true},
		{origin: "https://API.GITHUB.COM:443", want: true},
		{origin: "http://api.github.com"},
		{origin: "https://api.github.com:444"},
		{origin: "https://api.github.com."},
		{origin: "https://evil.api.github.com"},
		{origin: "https://api.github.com.evil.example"},
		{origin: "https://user@api.github.com"},
		{origin: "https://github.com"},
		{origin: "http://127.0.0.1:1234"},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			u, err := url.Parse(tc.origin)
			if err != nil {
				t.Fatal(err)
			}
			if got := isGitHubAPI(u); got != tc.want {
				t.Fatalf("credential origin = %t, want %t", got, tc.want)
			}
			isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
				if got := req.Header.Get("Authorization") == "Bearer private-token"; got != tc.want {
					t.Errorf("attached credential = %t, want %t", got, tc.want)
				}
				return authResponse(req, http.StatusOK, `{"tag_name":"v7.3.3"}`), nil
			})
			t.Setenv("GH_TOKEN", "private-token")
			apiBaseURL = tc.origin
			_, err = Latest(context.Background(), "owner/repo")
			if tc.origin == "http://api.github.com" {
				if err == nil {
					t.Fatal("non-loopback HTTP accepted")
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLatestRedirectCredentials(t *testing.T) {
	for _, tc := range []struct {
		name    string
		target  string
		want    string
		blocked bool
	}{
		{name: "same origin", target: "https://api.github.com/next", want: "Bearer private-token"},
		{name: "default port", target: "https://api.github.com:443/next", want: "Bearer private-token"},
		{name: "subdomain", target: "https://evil.api.github.com/next"},
		{name: "foreign origin", target: "https://evil.example/next"},
		{name: "nondefault port", target: "https://api.github.com:444/next"},
		{name: "userinfo", target: "https://user@api.github.com/next", want: "Basic dXNlcjo="},
		{name: "downgrade", target: "http://api.github.com/next", blocked: true},
		{name: "loopback", target: "http://127.0.0.1/next", blocked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.Header.Get("Authorization") != "Bearer private-token" {
						t.Fatal("initial request missing credential")
					}
					resp := authResponse(req, http.StatusFound, "")
					resp.Header.Set("Location", tc.target)
					return resp, nil
				}
				if tc.blocked {
					t.Fatal("unsafe redirect reached transport")
				}
				if req.Header.Get("Authorization") != tc.want {
					t.Error("redirect received unexpected credential")
				}
				return authResponse(req, http.StatusOK, `{"tag_name":"v7.3.3"}`), nil
			})
			t.Setenv("GH_TOKEN", "private-token")
			_, err := Latest(context.Background(), "owner/repo")
			if (err != nil) != tc.blocked {
				t.Fatalf("redirect error = %v; blocked = %t", err, tc.blocked)
			}
			wantCalls := 2
			if tc.blocked {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("requests = %d, want %d", calls, wantCalls)
			}
		})
	}
}

func TestDownloadsNeverUseGitHubCredential(t *testing.T) {
	for _, origin := range []string{"https://github.com", "https://api.github.com"} {
		t.Run(origin, func(t *testing.T) {
			calls := 0
			isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("Authorization") != "" {
					t.Error("download or checksum received credential")
				}
				if req.URL.Path == "/asset" || req.URL.Path == "/asset.sha256" {
					resp := authResponse(req, http.StatusFound, "")
					resp.Header.Set("Location", "https://api.github.com/redirected"+req.URL.Path)
					return resp, nil
				}
				body := "asset-content"
				if strings.HasSuffix(req.URL.Path, ".sha256") {
					body = fmt.Sprintf("%x  asset\n", sha256.Sum256([]byte(body)))
				}
				return authResponse(req, http.StatusOK, body), nil
			})
			t.Setenv("GH_TOKEN", "private-token")
			if err := downloadVerified(context.Background(), origin+"/asset", filepath.Join(t.TempDir(), "asset"), "asset", 0o600); err != nil {
				t.Fatal(err)
			}
			if calls != 4 {
				t.Fatalf("requests = %d, want asset, checksum, and both redirects", calls)
			}
		})
	}
}

func TestGHCredentialHonorsCallerCancellation(t *testing.T) {
	dir := isolateReleaseAuth(t, func(req *http.Request) (*http.Response, error) {
		t.Fatal("canceled lookup reached transport")
		return nil, nil
	})
	fakeAuthCLI(t, dir, "timeout")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := Latest(ctx, "owner/repo")
	if err == nil || !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("Latest cancellation = %v", err)
	}
}
