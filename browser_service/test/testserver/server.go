package testserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

	// Download test pages
	handler.HandleFunc("/download-file", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename=test.txt")
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Test file content"))
	})

	handler.HandleFunc("/multi-download-page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
			<a href="/files/file1.txt" download>Download 1</a>
			<a href="/files/file2.txt" download>Download 2</a>
			<a href="/files/file3.txt" download>Download 3</a>
		</body></html>`))
	})

	handler.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("File content for " + filename))
	})

	handler.HandleFunc("/document.pdf", func(w http.ResponseWriter, r *http.Request) {
		// Minimal valid PDF
		pdfContent := "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R>>endobj xref\n0 4\n0000000000 65535 f\n0000000009 00000 n\n0000000056 00000 n\n0000000114 00000 n\ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n190\n%%EOF"
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=document.pdf")
		_, _ = w.Write([]byte(pdfContent))
	})

	handler.HandleFunc("/trigger-download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
			<a href="/download-file" id="download-link">Click to download</a>
		</body></html>`))
	})

	handler.HandleFunc("/no-download-page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>Regular page with no downloads</body></html>"))
	})

	// Stealth test pages
	handler.HandleFunc("/detect-automation", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><article><script>
			document.querySelector('article').textContent = 'webdriver: ' + navigator.webdriver;
		</script></article></body></html>`))
	})

	handler.HandleFunc("/echo-user-agent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(fmt.Sprintf(`<html><body><article>%s</article></body></html>`, r.UserAgent())))
	})

	handler.HandleFunc("/viewport-info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><article><script>
			document.querySelector('article').textContent = 'Width: ' + window.innerWidth + ', Height: ' + window.innerHeight;
		</script></article></body></html>`))
	})

	// Popups test pages
	handler.HandleFunc("/alert-page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><article><script>
			alert('Test alert message');
			document.querySelector('article').textContent = 'Alert displayed';
		</script></article></body></html>`))
	})

	handler.HandleFunc("/confirm-page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><article id="result">Ready</article></body></html>`))
	})

	handler.HandleFunc("/prompt-page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><article id="result">Ready</article></body></html>`))
	})

	handler.HandleFunc("/clipboard-test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
			<button id="write">Write to clipboard</button>
			<button id="read">Read from clipboard</button>
			<div id="output"></div>
			<script>
				document.getElementById('write').onclick = async () => {
					try {
						await navigator.clipboard.writeText('test data');
						document.getElementById('output').textContent = 'Write success';
					} catch (e) {
						document.getElementById('output').textContent = 'Write error: ' + e.message;
					}
				};
				document.getElementById('read').onclick = async () => {
					try {
						const text = await navigator.clipboard.readText();
						document.getElementById('output').textContent = 'Read: ' + text;
					} catch (e) {
						document.getElementById('output').textContent = 'Read error: ' + e.message;
					}
				};
			</script>
		</body></html>`))
	})

	return httptest.NewServer(handler)
}
