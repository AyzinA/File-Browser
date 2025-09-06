package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Embed templates and static into the binary.
//go:embed templates/* static/*
var content embed.FS

var (
	rootDir  string
	useTLS   bool
	certFile string
	keyFile  string
	tpl      *template.Template
)

type fileRow struct {
	Name      string
	RelPath   string
	IsDir     bool
	SizeBytes int64
	ModTime   time.Time
}

type crumb struct {
	Name string
	URL  string
}

type pageData struct {
	Rows        []fileRow
	Path        string // current relative path inside ROOT_DIR
	Q           string
	Sort        string
	Order       string
	Breadcrumbs []crumb
}

func main() {
	rootDir = getenv("ROOT_DIR", "/data")
	addr := getenv("HOST", "0.0.0.0") + ":" + getenv("PORT", "8080")
	useTLS = strings.ToLower(getenv("USE_TLS", "false")) == "true"
	certFile = getenv("CERT_FILE", "certs/cert.pem")
	keyFile = getenv("KEY_FILE", "certs/key.pem")

	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		log.Fatalf("failed to ensure ROOT_DIR: %v", err)
	}

	funcs := template.FuncMap{
		"human": func(sz int64, isDir bool) string {
			if isDir {
				return "—"
			}
			const k = 1024.0
			f := float64(sz)
			u := []string{"B", "KB", "MB", "GB", "TB"}
			i := 0
			for f >= k && i < len(u)-1 {
				f /= k
				i++
			}
			return fmt.Sprintf("%.2f %s", f, u[i])
		},
	}
	var err error
	tpl, err = template.New("").Funcs(funcs).ParseFS(content, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", listHandler)
	mux.HandleFunc("/download", downloadHandler)
	// Serve embedded static at /static/
	mux.Handle("/static/", http.FileServer(http.FS(content)))

	log.Printf("File Browser serving %s on %s (TLS=%v)", rootDir, addr, useTLS)
	if useTLS {
		mustExist(certFile, "certificate")
		mustExist(keyFile, "private key")
		log.Fatal(http.ListenAndServeTLS(addr, certFile, keyFile, mux))
	} else {
		log.Fatal(http.ListenAndServe(addr, mux))
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := pick(r.URL.Query().Get("sort"), "name", "size", "mod")
	order := pick(r.URL.Query().Get("order"), "asc", "desc")

	rel := filepath.Clean(strings.TrimPrefix(r.URL.Query().Get("path"), "/"))
	if rel == "." {
		rel = ""
	}
	abs := filepath.Join(rootDir, rel)
	if !isWithin(abs, rootDir) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows := make([]fileRow, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		row := fileRow{
			Name:      e.Name(),
			RelPath:   filepath.ToSlash(filepath.Join(rel, e.Name())),
			IsDir:     e.IsDir(),
			SizeBytes: info.Size(),
			ModTime:   info.ModTime(),
		}
		if q == "" || strings.Contains(strings.ToLower(row.Name), strings.ToLower(q)) {
			rows = append(rows, row)
		}
	}

	// sort (folders first for name/size)
	sort.Slice(rows, func(i, j int) bool {
		less := false
		switch sortBy {
		case "size":
			if rows[i].IsDir != rows[j].IsDir {
				less = rows[i].IsDir && !rows[j].IsDir
			} else {
				less = rows[i].SizeBytes < rows[j].SizeBytes
			}
		case "mod":
			less = rows[i].ModTime.Before(rows[j].ModTime)
		default: // name
			if rows[i].IsDir != rows[j].IsDir {
				less = rows[i].IsDir && !rows[j].IsDir
			} else {
				less = strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
			}
		}
		if order == "desc" {
			return !less
		}
		return less
	})

	data := pageData{
		Rows:        rows,
		Path:        rel,
		Q:           q,
		Sort:        sortBy,
		Order:       order,
		Breadcrumbs: crumbsFor(rel),
	}
	if err := tpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	rel := filepath.Clean(strings.TrimPrefix(r.URL.Query().Get("path"), "/"))
	abs := filepath.Join(rootDir, rel)
	if !isWithin(abs, rootDir) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if info.IsDir() {
		http.Redirect(w, r, "/?"+url.Values{"path": {rel}}.Encode(), http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(abs)))
	http.ServeFile(w, r, abs)
}

func crumbsFor(rel string) []crumb {
	if rel == "" {
		return nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	acc := ""
	out := make([]crumb, 0, len(parts))
	for _, p := range parts {
		if p == "." || p == "" {
			continue
		}
		if acc == "" {
			acc = p
		} else {
			acc = acc + "/" + p
		}
		out = append(out, crumb{
			Name: p,
			URL:  "/?" + url.Values{"path": {acc}}.Encode(),
		})
	}
	return out
}

func pick(v string, opts ...string) string {
	v = strings.ToLower(v)
	for _, o := range opts {
		if v == o {
			return v
		}
	}
	return opts[0]
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func isWithin(p, root string) bool {
	pAbs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pAbs = filepath.Clean(pAbs)
	rootAbs = filepath.Clean(rootAbs)
	sep := string(os.PathSeparator)
	return strings.HasPrefix(pAbs+sep, rootAbs+sep) || pAbs == rootAbs
}

func mustExist(path, what string) {
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("%s not found at %s: %v", what, path, err)
	}
}
