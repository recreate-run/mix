package testserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// StartTestServer starts an HTTP test server for E2E tests
func StartTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.NewServeMux()

	// Cookie test pages
	handler.HandleFunc("/echo-cookies", func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		var parts []string
		for _, c := range cookies {
			parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Join(parts, ", ")))
	})

	handler.HandleFunc("/check-cookies", func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		var parts []string
		for _, c := range cookies {
			parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Join(parts, ", ")))
	})

	handler.HandleFunc("/admin/check-cookies", func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		var parts []string
		for _, c := range cookies {
			parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Join(parts, ", ")))
	})

	// Storage test pages
	handler.HandleFunc("/login-form", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			username := r.FormValue("username")
			if username == "testuser" {
				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    "abc123",
					Path:     "/",
					HttpOnly: true,
				})
				http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
			<form method="POST">
				<input type="text" name="username" id="username" />
				<input type="password" name="password" id="password" />
				<button type="submit" id="submit">Login</button>
			</form>
		</body></html>`))
	})

	handler.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		w.Header().Set("Content-Type", "text/plain")
		if err == nil && cookie.Value == "abc123" {
			_, _ = w.Write([]byte("Welcome back, testuser"))
		} else {
			_, _ = w.Write([]byte("Please login"))
		}
	})

	handler.HandleFunc("/storage-test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
			<button id="show">Show Storage</button>
			<div id="output"></div>
			<script>
				document.getElementById('show').onclick = () => {
					const items = [];
					for (let i = 0; i < localStorage.length; i++) {
						const key = localStorage.key(i);
						items.push(key + '=' + localStorage.getItem(key));
					}
					document.getElementById('output').textContent = items.join(', ');
				};
			</script>
		</body></html>`))
	})

	return httptest.NewServer(handler)
}
