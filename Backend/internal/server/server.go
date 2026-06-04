package server

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"lumehub/internal/api"
	"lumehub/internal/auth"
	"lumehub/internal/config"
	"lumehub/internal/store"
)

func NewMux() *http.ServeMux {
	dataDir := config.DataDir()
	wwwDir := config.WWWDir()

	st := store.New(dataDir)
	if used, err := st.RecalculateStorageUsed(); err != nil {
		log.Printf("storage: initial scan failed: %v", err)
	} else {
		log.Printf("storage: used %d bytes", used)
	}
	if n, err := st.PurgeExpiredTrash(); err != nil {
		log.Printf("trash: purge failed: %v", err)
	} else if n > 0 {
		log.Printf("trash: purged %d expired item(s)", n)
	}
	authMgr := auth.New(st, config.AuthPassword())
	h := api.NewHandler(st, authMgr, log.Default())

	mux := http.NewServeMux()
	h.Register(mux)

	resRoot := filepath.Join(dataDir, "resource")
	_ = os.MkdirAll(resRoot, 0o755)
	fsRes := auth.ResourceDir(authMgr, st, http.Dir(resRoot))
	mux.Handle("/resource/", http.StripPrefix("/resource/", fsRes))
	systemResRoot := filepath.Join(dataDir, "system", "resource")
	_ = os.MkdirAll(systemResRoot, 0o755)
	mux.Handle("/resource/system/", http.StripPrefix("/resource/system/", http.FileServer(http.Dir(systemResRoot))))

	mux.HandleFunc("/", spaHandler(wwwDir))

	return mux
}

func spaHandler(wwwDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		urlPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		rel := strings.TrimPrefix(urlPath, "/")
		if rel == "" || rel == "." {
			serveIndex(w, r, wwwDir)
			return
		}

		local := filepath.Join(wwwDir, filepath.FromSlash(rel))
		fi, err := os.Stat(local)
		if err == nil && !fi.IsDir() {
			http.ServeFile(w, r, local)
			return
		}

		serveIndex(w, r, wwwDir)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request, wwwDir string) {
	index := filepath.Join(wwwDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("LumeHub: www/index.html 缺失。请在 Frontend 执行 npm run build（输出到 Backend/www），或设置环境变量 LUMEHUB_WWW。\n"))
		return
	}
	http.ServeFile(w, r, index)
}
