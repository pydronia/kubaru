package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pydronia/kubaru/internal/tlsutils"
)

type KubaruConfig struct {
	User, Pass, Host, Port, Path string
	MediaFiles                   []string
}

func NewConfig(user, pass, host, port, path string) (KubaruConfig, error) {
	cfg := KubaruConfig{
		User: user,
		Pass: pass,
		Host: host,
		Port: port,
		Path: path,
	}
	err := cfg.validateCredentials()
	if err != nil {
		return KubaruConfig{}, err
	}
	err = cfg.validateMediaPath()
	if err != nil {
		return KubaruConfig{}, fmt.Errorf("path must point to a valid directory: %w", err)
	}
	err = tlsutils.CheckTLSCert()
	if err != nil {
		return KubaruConfig{}, err
	}
	return cfg, nil
}

func (cfg *KubaruConfig) validateCredentials() error {
	if cfg.User == "" {
		cfg.User = os.Getenv("KUBARU_USER")
		if cfg.User == "" {
			cfg.User = "user"
		}
	}
	if strings.ContainsRune(cfg.User, ':') {
		return errors.New("username cannot contain a colon")
	}

	if cfg.Pass == "" {
		cfg.Pass = os.Getenv("KUBARU_PASS")
	}
	if cfg.Pass == "" {
		randBuff := make([]byte, 12)
		rand.Read(randBuff)
		cfg.Pass = base64.RawURLEncoding.EncodeToString(randBuff)
	} else if len(cfg.Pass) <= 10 {
		log.Println("WARNING: Passowrd is recommended to be longer than 10 bytes")
	}
	return nil
}

func (cfg *KubaruConfig) validateMediaPath() error {
	if cfg.Path == "" {
		return errors.New("no path provided")
	}
	cfg.Path = filepath.Clean(cfg.Path)
	mediaFiles, err := generateMediaList(cfg.Path)
	if err != nil {
		return err
	}
	if len(mediaFiles) == 0 {
		log.Println("WARNING: provided path does not contain any valid media files")
	}
	cfg.MediaFiles = mediaFiles
	return nil
}

// Generate a list of the paths all media in the provided directory, filtering by extension.
// Skips dot-prefixed files and folders. Currently uses a hardcoded list of common audio and video extensions.
// TODO: add option for including extra extensions.
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
