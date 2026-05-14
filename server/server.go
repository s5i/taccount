//go:build windows

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/s5i/tassist/acc"
	"github.com/s5i/tassist/exp"
	"github.com/s5i/tassist/online"
	"github.com/s5i/tassist/ping"
	"github.com/s5i/tassist/settings"

	"golang.org/x/sync/errgroup"

	_ "embed"
)

var ErrRestart = fmt.Errorf("server is restarting...")

func New(tmpDir string, accStorage *acc.Storage, expCache *exp.Cache, pinger *ping.Pinger, online *online.Online, version string, stStorage *settings.Storage) (*Server, error) {
	s := &Server{
		tmpDir:               tmpDir,
		acc:                  accStorage,
		exp:                  expCache,
		pinger:               pinger,
		online:               online,
		version:              version,
		stStorage:            stStorage,
		keepalive:            time.Now(),
		keepaliveCheckPeriod: 2 * time.Second,
		keepaliveFails:       3,
		keepaliveTimeout:     15 * time.Second,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndexHTML)
	mux.HandleFunc("/style.css", s.handleStyleCSS)
	mux.HandleFunc("/main.js", s.handleMainJS)
	mux.HandleFunc("/favicon.ico", s.handleFaviconIco)
	mux.HandleFunc("/api/keepalive", s.handleKeepalive)
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
	mux.HandleFunc("/api/world/ping", s.handleWorldPing)
	mux.HandleFunc("/api/world/online", s.handleWorldOnline)
	mux.HandleFunc("/api/preset/switch", s.handlePresetSwitch)
	mux.HandleFunc("/api/preset/list", s.handlePresetList)
	mux.HandleFunc("/api/update/check", s.handleUpdateCheck)
	mux.HandleFunc("/api/update/execute", s.handleUpdateExecute)
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
	s.cancel = cancel

	s.srv = &http.Server{Handler: s.mux}

	eg.Go(func() error {
		return s.srv.Serve(ln)
	})

	eg.Go(func() error {
		<-ctx.Done()
		s.srv.Shutdown(ctx)
		return ctx.Err()
	})

	eg.Go(func() error {
		fails := 0
		for {
			s.keepaliveMu.Lock()
			if time.Now().After(s.keepalive.Add(s.keepaliveTimeout)) {
				fails++
			} else {
				fails = 0
			}
			s.keepaliveMu.Unlock()

			if fails >= s.keepaliveFails {
				log.Printf("Last %d ping checks showed no activity within the last %v; quitting.", s.keepaliveFails, s.keepaliveTimeout)
				cancel()
				return context.Canceled
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.keepaliveCheckPeriod):
			}
		}
	})

	eg.Go(func() error {
		exec.Command("explorer", "http://"+s.ln.Addr().String()).Start()
		return nil
	})

	err = eg.Wait()

	if s.restart {
		return ErrRestart
	}

	return err
}

type Server struct {
	tmpDir    string
	acc       *acc.Storage
	exp       *exp.Cache
	pinger    *ping.Pinger
	online    *online.Online
	stStorage *settings.Storage

	srv *http.Server
	mux *http.ServeMux
	ln  net.Listener

	version string

	cancel               func()
	keepalive            time.Time
	keepaliveTimeout     time.Duration
	keepaliveFails       int
	keepaliveCheckPeriod time.Duration
	keepaliveMu          sync.Mutex

	updateReady   bool
	updaterPath   string
	updaterSource string
	updateMu      sync.Mutex

	restart bool
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

	log.Printf("Account ID=%q renamed to %q.", req.ID, req.Name)

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

	log.Printf("Account ID=%q deleted.", req.ID)

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

	if err := acc.RegRestore(s.stStorage.Get().RegistryPath, row.A, row.B, row.C); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Account %q (ID=%q) loaded.", row.Name, row.ID)

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

	a, b, c, err := acc.RegSnapshot(s.stStorage.Get().RegistryPath)
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

	log.Printf("Account %q (ID=%q) stored.", name, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entryJSON{ID: id, Name: name})
}

func (s *Server) handleExpReset(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session reset.")
	s.exp.Reset()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStart(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session started.")
	s.exp.Start()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpStop(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session stopped.")
	s.exp.Stop()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpPause(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session paused.")
	s.exp.Pause()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleExpUnpause(w http.ResponseWriter, r *http.Request) {
	log.Printf("Exp session unpaused.")
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

func (s *Server) handleWorldPing(w http.ResponseWriter, r *http.Request) {
	stats := s.pinger.Stats()
	ret := struct {
		OK         bool    `json:"ok"`
		RTTMSec    int     `json:"rtt_msec"`
		PacketLoss float64 `json:"packet_loss"`
	}{
		OK:         stats.OK,
		RTTMSec:    int(stats.RTT / time.Millisecond),
		PacketLoss: stats.PacketLoss,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}

func (s *Server) handleWorldOnline(w http.ResponseWriter, r *http.Request) {
	online, ok := s.online.Get()
	ret := struct {
		OK     bool `json:"ok"`
		Online int  `json:"online"`
	}{
		OK:     ok,
		Online: online,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ret)
}

func (s *Server) handleKeepalive(w http.ResponseWriter, r *http.Request) {
	s.keepaliveMu.Lock()
	defer s.keepaliveMu.Unlock()
	s.keepalive = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Version string `json:"version"`
	}{Version: s.version})
}

func (s *Server) handlePresetSwitch(w http.ResponseWriter, r *http.Request) {
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

	if err := s.stStorage.SwitchPreset(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))

	s.restart = true
	s.cancel()
}

func (s *Server) handlePresetList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Active    string   `json:"active"`
		Available []string `json:"available"`
	}{
		Active:    s.stStorage.Preset(),
		Available: slices.Sorted(maps.Keys(settings.Presets)),
	})
}

func (s *Server) handleUpdateExecute(w http.ResponseWriter, r *http.Request) {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	if !s.updateReady {
		http.Error(w, "Update is not ready.", http.StatusNotFound)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))

	go exec.Command("cmd", "/C", "start", s.updaterPath, os.Args[0], s.updaterSource).Run()
	time.Sleep(3 * time.Second)

	s.cancel()
}

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	ret := struct {
		Available bool   `json:"available"`
		Version   string `json:"version,omitempty"`
	}{}

	w.Header().Set("Content-Type", "application/json")
	defer func() { json.NewEncoder(w).Encode(ret) }()

	matched, err := regexp.MatchString(`^v\d+\.\d+\.\d+$`, s.version)
	if err != nil || !matched {
		return
	}

	resp, err := http.Get("https://api.github.com/repos/s5i/tassist/releases/latest")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var releaseData struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releaseData); err != nil {
		return
	}

	if releaseData.TagName == s.version {
		return
	}

	var tassistURL, updaterURL string
	for _, asset := range releaseData.Assets {
		switch asset.Name {
		case "tassist.exe":
			tassistURL = asset.BrowserDownloadURL
		case "updater.exe":
			updaterURL = asset.BrowserDownloadURL
		}
	}
	if tassistURL == "" || updaterURL == "" {
		return
	}

	sourcePath := filepath.Join(s.tmpDir, fmt.Sprintf("tassist_%s.exe", releaseData.TagName))
	if err := downloadFile(sourcePath, tassistURL); err != nil {
		return
	}

	updaterPath := filepath.Join(s.tmpDir, "updater.exe")
	if err := downloadFile(updaterPath, updaterURL); err != nil {
		return
	}

	ret.Available = true
	ret.Version = releaseData.TagName

	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	s.updateReady = true
	s.updaterPath = updaterPath
	s.updaterSource = sourcePath
}

func downloadFile(path string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
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
