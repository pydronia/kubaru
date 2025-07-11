package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pydronia/kubaru/internal/netutils"
)

// TODO:
// - improve network stuff (cleaner tls generation, generate based on actual ips? Maybe have a separete commandline flag to generate instead of asking for input)
// - Figure out if we need the addresses printed, consider a separate option to just print addresses then exit
// - Rate limiting for authentication
// - Write usage (-h flag)
// - Clean up/refactor code (packages maybe)
// - Write readme, add licence, other github stuff
// - make public
// - Nice logging/bandwidth monitoring (colors!)
// - Support changing directory contents
// - Make simple homepage with instructions (maybe)
// - Playback syncing service (write vlc, mpv plugin)

func main() {
	// Flags
	var host, port, path, user, pass string
	flag.StringVar(&host, "host", "::", "Host address to listen on")
	flag.StringVar(&port, "port", "443", "Port to listen on")
	flag.StringVar(&path, "path", "", "Path to directory to serve (required)")
	flag.StringVar(&user, "user", "user", "Username for basic auth")
	flag.StringVar(&pass, "pass", "", "Password for basic auth. Generate random password if not set")
	flag.Parse()

	// Check authentication credentials
	if envUser := os.Getenv("KUBARU_USER"); len(envUser) != 0 {
		user = envUser
	}
	if strings.ContainsRune(user, ':') {
		log.Fatalln("username cannot contain a colon")
	}
	if envPass := os.Getenv("KUBARU_PASS"); len(envPass) != 0 {
		pass = envPass
	}
	if len(pass) == 0 {
		randBuff := make([]byte, 12)
		rand.Read(randBuff)
		pass = base64.RawURLEncoding.EncodeToString(randBuff)
	} else if len(pass) <= 10 {
		log.Println("WARNING: password is recommended to be longer than 10 bytes")
	}

	// Check media files
	mediaFiles, err := generateMediaList(filepath.Clean(path))
	if err != nil {
		log.Fatalln("path must point to a valid directory:", err)
	}
	if len(mediaFiles) == 0 {
		log.Fatalln("provided path does not contain any valid media files")
	}

	_, err1 := os.Stat("cert.pem")
	_, err2 := os.Stat("key.pem")
	if errors.Is(err1, os.ErrNotExist) || errors.Is(err2, os.ErrNotExist) {
		fmt.Print("TLS cert not found. Enter comma-separated IPs/domains to generate for: ")
		var hosts string
		fmt.Scanln(&hosts)
		if err := netutils.GenerateTlsCert(hosts); err != nil {
			log.Fatalln(err)
		}
	}

	server := &http.Server{
		Addr:         net.JoinHostPort(host, port),
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Minute * 10,
		IdleTimeout:  time.Minute * 2,
	}

	http.Handle("GET /files/", netutils.BasicAuthMiddleware(user, pass, http.StripPrefix("/files/", http.FileServer(http.Dir(path)))))

	http.Handle("GET /playlist.m3u8", netutils.BasicAuthMiddleware(user, pass, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-mpegurl")
		baseUrl := fmt.Sprintf("https://%s:%s@%s/files/", user, pass, r.Host)
		m3uFile := generateM3uFile(baseUrl, mediaFiles)
		io.WriteString(w, m3uFile)
	})))

	http.Handle("GET /{$}", netutils.BasicAuthMiddleware(user, pass, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/playlist.m3u8", http.StatusSeeOther)
	})))

	log.Println("Listening on", server.Addr)
	log.Fatalln(server.ListenAndServeTLS("cert.pem", "key.pem"))
}

// Generate a list of the paths all media in the provided directory, filtering by extension.
// Skips dot-prefixed files and folders. Currently uses a hardcoded list of common audio and video extensions.
// TODO: add flag for including extra extensions.
func generateMediaList(rootPath string) ([]string, error) {
	var files []string
	allowedExts := map[string]struct{}{
		".webm": {},
		".mkv":  {},
		".ogv":  {},
		".ogg":  {},
		".avi":  {},
		".mov":  {},
		".wmv":  {},
		".mp4":  {},
		".m4p":  {},
		".m4v":  {},
		".mpg":  {},
		".mpeg": {},
		".flv":  {},
		".aac":  {},
		".aiff": {},
		".flac": {},
		".m4a":  {},
		".mp3":  {},
		".oga":  {},
		".opus": {},
		".wav":  {},
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != "." && d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		if d.Type().IsRegular() && !strings.HasPrefix(d.Name(), ".") {
			if _, ok := allowedExts[strings.ToLower(filepath.Ext(path))]; ok {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
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
