//go:build windows

package server

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/s5i/tassist/acc"
	"github.com/s5i/tassist/exp"
	"golang.org/x/sync/errgroup"

	_ "embed"
)

func New(accStorage *acc.Storage, expCache *exp.Cache, version string) (*Server, error) {
	s := &Server{
		acc:             accStorage,
		exp:             expCache,
		version:         version,
		ping:            time.Now(),
		pingCheckPeriod: 2 * time.Second,
		pingTimeout:     15 * time.Second,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndexHTML)
	mux.HandleFunc("/style.css", s.handleStyleCSS)
	mux.HandleFunc("/main.js", s.handleMainJS)
	mux.HandleFunc("/favicon.ico", s.handleFaviconIco)
	mux.HandleFunc("/api/ping", s.handlePing)
	mux.HandleFunc("/api/version", s.handleVersion)
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

	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)

	eg.Go(func() error {
		s.srv = &http.Server{Handler: s.mux}
		return s.srv.Serve(ln)
	})

	eg.Go(func() error {
		<-ctx.Done()
		s.srv.Shutdown(ctx)
		return ctx.Err()
	})

	eg.Go(func() error {
		for {
			var ok bool
			s.pingMu.Lock()
			ok = s.ping.Add(s.pingTimeout).After(time.Now())
			s.pingMu.Unlock()

			if !ok {
				log.Printf("No pings received in the last %v; quitting.", s.pingTimeout)
				cancel()
				return context.Canceled
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.pingCheckPeriod):
			}
		}
	})

	eg.Go(func() error {
		exec.Command("explorer", "http://"+s.ln.Addr().String()).Start()
		return nil
	})

	return eg.Wait()
}

type Server struct {
	acc *acc.Storage
	exp *exp.Cache

	srv *http.Server
	mux *http.ServeMux
	ln  net.Listener

	version string

	cancel          func()
	ping            time.Time
	pingTimeout     time.Duration
	pingCheckPeriod time.Duration
	pingMu          sync.Mutex
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
	w.Write(favicon)
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
		Level        int `json:"level,omitempty"`
		TotalExp     int `json:"total_exp,omitempty"`
		RemainingExp int `json:"remaining_exp,omitempty"`

		SessionDelta       int `json:"session_delta,omitempty"`
		SessionDurationSec int `json:"session_duration_sec,omitempty"`
		SessionRate        int `json:"session_rate,omitempty"`

		Running bool `json:"running"`
		Paused  bool `json:"paused"`
	}{
		Level:        stats.Level,
		TotalExp:     stats.TotalExp,
		RemainingExp: stats.RemainingExp,

		SessionDelta:       stats.SessionDelta,
		SessionDurationSec: int(stats.SessionDuration / time.Second),
		SessionRate:        stats.SessionRate,

		Running: stats.Running,
		Paused:  stats.Paused,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	s.pingMu.Lock()
	defer s.pingMu.Unlock()
	s.ping = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Version string `json:"version"`
	}{Version: s.version})
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
	//go:embed static/favicon.ico
	favicon []byte
)
