package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func toKebab(name string) string {
	s := strings.ToLower(name)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func availablePath(dir, base, ext string) string {
	p := filepath.Join(dir, base+ext)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	for i := 1; ; i++ {
		p = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

func moveToTrash(src string) error {
	trash := filepath.Join(os.Getenv("HOME"), ".Trash")
	name := filepath.Base(src)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	dest := availablePath(trash, base, ext)
	return os.Rename(src, dest)
}

func convert(path string) error {
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	kebab := toKebab(base)
	out := availablePath(dir, kebab, ".jpg")

	log.Printf("%s → %s", name, filepath.Base(out))

	cmd := exec.Command("sips", "-s", "format", "jpeg", "-s", "formatOptions", "80", path, "--out", out)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sips failed: %w: %s", err, output)
	}

	if err := moveToTrash(path); err != nil {
		return fmt.Errorf("move to trash: %w", err)
	}

	return nil
}

func isHEIC(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".heic" || ext == ".heif"
}

func processExisting(dirs []string) {
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Printf("scan %s: %v", dir, err)
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !isHEIC(e.Name()) {
				continue
			}
			if err := convert(filepath.Join(dir, e.Name())); err != nil {
				log.Printf("error: %v", err)
			}
		}
	}
}

func main() {
	home := os.Getenv("HOME")
	dirs := []string{
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Desktop"),
	}

	log.SetFlags(log.Ldate | log.Ltime)
	log.Printf("heic-to-jpg started, watching %v", dirs)

	processExisting(dirs)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("fsnotify: %v", err)
	}
	defer watcher.Close()

	for _, d := range dirs {
		if err := watcher.Add(d); err != nil {
			log.Fatalf("watch %s: %v", d, err)
		}
	}

	pending := make(map[string]time.Time)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if isHEIC(ev.Name) && ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0 {
				pending[ev.Name] = time.Now()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watch error: %v", err)
		case <-tick.C:
			now := time.Now()
			for path, seen := range pending {
				if now.Sub(seen) < 300*time.Millisecond {
					continue
				}
				delete(pending, path)
				if _, err := os.Stat(path); err != nil {
					continue
				}
				if err := convert(path); err != nil {
					log.Printf("error: %v", err)
				}
			}
		}
	}
}
