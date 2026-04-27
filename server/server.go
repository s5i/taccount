//go:build windows

package server

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/s5i/tassist/acc"
	"github.com/s5i/tassist/assets"
	"github.com/s5i/tassist/exp"
	"golang.org/x/sync/errgroup"

	_ "embed"
)

func New(storagePath string) (*Server, error) {
	st, err := acc.New(storagePath)
	if err != nil {
		return nil, err
	}

	expCache, err := exp.NewCache()
	if err != nil {
		return nil, err
	}

	s := &Server{
		acc:   st,
		exp:   expCache,
		ready: make(chan bool),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndexHTML)
	mux.HandleFunc("/style.css", s.handleStyleCSS)
	mux.HandleFunc("/main.js", s.handleMainJS)
	mux.HandleFunc("/favicon.ico", s.handleFaviconIco)
	mux.HandleFunc("/api/healthz", s.handleHealthz)
	mux.HandleFunc("/api/accounts/list", s.handleAccList)
	mux.HandleFunc("/api/accounts/rename", s.handleAccRename)
	mux.HandleFunc("/api/accounts/delete", s.handleAccDelete)
	mux.HandleFunc("/api/accounts/load", s.handleAccLoad)
	mux.HandleFunc("/api/accounts/store", s.handleAccStore)
	mux.HandleFunc("/api/exp/stats", s.handleExpStats)
	mux.HandleFunc("/api/exp/start", s.handleExpStart)
	mux.HandleFunc("/api/exp/stop", s.handleExpStop)
	mux.HandleFunc("/api/exp/pause", s.handleExpPause)
	mux.HandleFunc("/api/exp/unpause", s.handleExpUnpause)
	mux.HandleFunc("/api/exp/reset", s.handleExpReset)
	s.mux = mux

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.ln = ln
	defer s.ln.Close()

	log.Printf("Listening on %s", s.ln.Addr())

	close(s.ready)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return http.Serve(s.ln, s.mux)
	})
	eg.Go(func() error {
		return s.exp.Run(ctx)
	})

	return eg.Wait()
}

func (s *Server) Ready() <-chan bool {
	return s.ready
}

func (s *Server) Addr() string {
	return s.ln.Addr().String()
}

type Server struct {
	acc   *acc.Storage
	exp   *exp.Cache
	mux   *http.ServeMux
	ln    net.Listener
	ready chan bool
}

func (s *Server) handleIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *Server) handleStyleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(styleCSS)
}

func (s *Server) handleMainJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(mainJS)
}

func (s *Server) handleFaviconIco(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.Write(assets.Favicon)
}

func (s *Server) handleAccList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.acc.ListRows()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := []entryJSON{} // JSON nil != empty slice.
	for _, row := range rows {
		out = append(out, entryJSON{ID: row.ID, Name: row.Name})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAccRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.acc.RenameRow(req.ID, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAccDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.acc.DeleteRow(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAccLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	row, found, err := s.acc.FindRow(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return
	}

	if err := acc.RegRestore(row.A, row.B, row.C); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAccStore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a, b, c, err := acc.RegSnapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := uuid.New().String()[:8]
	name := req.Name
	if name == "" {
		req.Name = "Unnamed"
	}

	if err := s.acc.AddRow(id, name, a, b, c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entryJSON{ID: id, Name: name})
}

func (s *Server) handleExpReset(w http.ResponseWriter, r *http.Request) {
	s.exp.Reset()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStart(w http.ResponseWriter, r *http.Request) {
	s.exp.Start()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStop(w http.ResponseWriter, r *http.Request) {
	s.exp.Stop()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpPause(w http.ResponseWriter, r *http.Request) {
	s.exp.Pause()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpUnpause(w http.ResponseWriter, r *http.Request) {
	s.exp.Unpause()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStats(w http.ResponseWriter, r *http.Request) {
	stats := s.exp.Stats()
	ret := struct {
		LevelCurrent int  `json:"level_current,omitempty"`
		ExpCurrent   int  `json:"exp_current,omitempty"`
		ExpNextLevel int  `json:"exp_next_level,omitempty"`
		ExpPerHour   int  `json:"exp_per_hour,omitempty"`
		Running      bool `json:"running"`
		Paused       bool `json:"paused"`
	}{
		LevelCurrent: stats.LevelCurrent,
		ExpCurrent:   stats.ExpCurrent,
		ExpNextLevel: stats.ExpRemaining,
		ExpPerHour:   stats.ExpPerHour,
		Running:      stats.Running,
		Paused:       stats.Paused,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

type entryJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	//go:embed static/index.html
	indexHTML []byte
	//go:embed static/style.css
	styleCSS []byte
	//go:embed static/main.js
	mainJS []byte
)
