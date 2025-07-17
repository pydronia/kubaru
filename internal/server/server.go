/*
Package server includes the required functions to
start a file server with the provided config.
*/
package server

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/pydronia/kubaru/internal/config"
)

// Start a server for Kubaru with the provided [config.KubaruConfig].
func StartServer(cfg *config.KubaruConfig) error {
	server := &http.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Minute * 10,
		IdleTimeout:  time.Minute * 2,
	}

	http.Handle("GET /files/", basicAuthMiddleware(cfg.User, cfg.Pass, http.StripPrefix("/files/", http.FileServer(http.Dir(cfg.Path)))))

	http.Handle("GET /playlist.m3u8", basicAuthMiddleware(cfg.User, cfg.Pass, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-mpegurl")
		baseUrl := fmt.Sprintf("https://%s:%s@%s/files/", cfg.User, cfg.Pass, r.Host)
		m3uFile := generateM3uFile(baseUrl, cfg.MediaFiles)
		io.WriteString(w, m3uFile)
	})))

	http.Handle("GET /{$}", basicAuthMiddleware(cfg.User, cfg.Pass, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/playlist.m3u8", http.StatusSeeOther)
	})))

	fmt.Printf("Username: %s\nPassword: %s\n\n", cfg.User, cfg.Pass)
	fmt.Printf("Now listening on %s\n", server.Addr)
	fmt.Printf("Access the media playlist locally at https://%s:%s@localhost:%s/\n", cfg.User, cfg.Pass, cfg.Port)
	return server.ListenAndServeTLS("cert.pem", "key.pem")
}

// Generate a M3U file with a list of URLs to all provided media files
func generateM3uFile(url string, mediaFiles []string) string {
	var builder strings.Builder
	builder.WriteString("#EXTM3U\n#PLAYLIST:Media Files\n")
	for _, file := range mediaFiles {
		builder.WriteString("#EXTINF:0," + path.Base(file) + "\n")
		builder.WriteString(url + file + "\n")
	}
	return builder.String()
}

// Returns a [http.Handler] that ensures that the incoming request includes authorization
// with HTTP Basic Authentication, with the provided credentials user and pass.
func basicAuthMiddleware(user, pass string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		b64, ok := strings.CutPrefix(authHeader, "Basic ")
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		decodedBytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		correctCredentials := []byte(user + ":" + pass)
		if subtle.ConstantTimeCompare(correctCredentials, decodedBytes) == 0 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
