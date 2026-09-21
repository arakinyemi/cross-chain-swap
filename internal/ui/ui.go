// Package ui serves the built React interface (web/) out of the binary, plus
// the QR codes its order page renders. `npm --prefix web run build` writes the
// bundle into dist/, which go:embed picks up — so a release is still one
// binary with no static files to deploy alongside it.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

//go:embed all:dist
var embedded embed.FS

const missingBundle = `<!doctype html><html><head><meta charset="utf-8">
<title>Interface not built</title>
<style>body{font:15px ui-sans-serif,system-ui;background:#0c131c;color:#e8eef6;padding:40px;line-height:1.6}
code{background:#18242f;padding:2px 6px;border-radius:5px}</style></head>
<body><h1>Interface not built</h1>
<p>The React bundle is missing from this binary. Build it, then rebuild the server:</p>
<p><code>npm --prefix web ci &amp;&amp; npm --prefix web run build</code></p>
<p>Or run the Vite dev server against this API: <code>npm --prefix web run dev</code></p>
</body></html>`

// Handler serves the bundle. Unknown paths fall back to index.html so deep
// links like /order/<id> reach the client router.
func Handler() http.Handler {
	assets, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}

	files := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if info, err := fs.Stat(assets, strings.TrimPrefix(r.URL.Path, "/")); err == nil && !info.IsDir() {
				// Hashed bundle filenames are safe to cache hard.
				if strings.HasPrefix(r.URL.Path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
		}

		page, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(missingBundle))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(page)
	})
}

// QRHandler renders a deposit address as a PNG QR code.
func QRHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := r.URL.Query().Get("data")
		if data == "" || len(data) > 512 {
			http.Error(w, "data query parameter is required", http.StatusBadRequest)
			return
		}

		size := 220
		if raw := r.URL.Query().Get("size"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 64 && parsed <= 512 {
				size = parsed
			}
		}

		png, err := qrcode.Encode(data, qrcode.Medium, size)
		if err != nil {
			http.Error(w, "could not render QR code", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(png)
	}
}
