package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Tristan578/taskboard/internal/db"
	"github.com/Tristan578/taskboard/internal/github"
	"github.com/Tristan578/taskboard/internal/models"
	"github.com/Tristan578/taskboard/internal/workflow"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"os"
)

type Server struct {
	store     *db.Store
	router    chi.Router
	startedAt time.Time
}

func New(store *db.Store, webFS fs.FS) *Server {
	s := &Server{store: store, startedAt: time.Now()}
	s.setupRoutes(webFS)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	slog.Info("server started", "port", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv.ListenAndServe()
}

// responseRecorder wraps http.ResponseWriter to capture the status code.
// It implements http.Hijacker by delegating to the underlying writer so that
// WebSocket upgrades work transparently.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rr.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
}

func (rr *responseRecorder) Flush() {
	if fl, ok := rr.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

// requestLogger is middleware that logs each HTTP request with method, path,
// status code, and duration.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rr, r)
		// #nosec G706 -- slog uses structured key-value pairs, not format strings
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rr.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}

func (s *Server) setupRoutes(webFS fs.FS) {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(localBoundary)

	r.Route("/api", func(r chi.Router) {
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", s.listProjects)
			r.Post("/", s.createProject)
			r.Get("/{id}", s.getProject)
			r.Put("/{id}", s.updateProject)
			r.Delete("/{id}", s.deleteProject)
		})

		r.Route("/teams", func(r chi.Router) {
			r.Get("/", s.listTeams)
			r.Post("/", s.createTeam)
			r.Get("/{id}", s.getTeam)
			r.Put("/{id}", s.updateTeam)
			r.Delete("/{id}", s.deleteTeam)
		})

		r.Route("/tickets", func(r chi.Router) {
			r.Get("/", s.listTickets)
			r.Post("/", s.createTicket)
			r.Get("/{id}", s.getTicket)
			r.Put("/{id}", s.updateTicket)
			r.Post("/{id}/move", s.moveTicket)
			r.Post("/{id}/activity", s.reportActivity)
			r.Post("/{id}/complete", s.completeTicket)
			r.Post("/{id}/resolve", s.resolveConflict)
			r.Delete("/{id}", s.deleteTicket)
			r.Post("/{id}/subtasks", s.addSubtask)
		})

		r.Route("/subtasks", func(r chi.Router) {
			r.Post("/{id}/toggle", s.toggleSubtask)
			r.Delete("/{id}", s.deleteSubtask)
		})

		r.Route("/labels", func(r chi.Router) {
			r.Get("/", s.listLabels)
			r.Post("/", s.createLabel)
			r.Put("/{id}", s.updateLabel)
			r.Delete("/{id}", s.deleteLabel)
		})

		r.Get("/board", s.getBoard)
		r.Get("/terminal/ws", s.handleTerminalWS)
		r.Get("/health", s.healthCheck)
		r.Get("/sync/status", s.syncStatus)
		r.Post("/sync/retry", s.retrySync)
	})

	if webFS != nil {
		fileServer := http.FileServer(http.FS(webFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fs.Stat(webFS, r.URL.Path[1:]); err != nil {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	s.router = r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func parsePagination(r *http.Request) (limit, offset int) {
	if v := r.URL.Query().Get("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &offset)
	}
	return
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, offset := parsePagination(r)

	var projects []models.Project
	var total int
	var err error

	if limit > 0 {
		projects, total, err = s.store.ListProjects(status, limit, offset)
	} else {
		projects, total, err = s.store.ListProjects(status)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []models.Project{}
	}

	if limit > 0 {
		writeJSON(w, http.StatusOK, models.PaginatedResult[models.Project]{
			Data: projects, Total: total, Limit: limit, Offset: offset,
			HasMore: offset+len(projects) < total,
		})
	} else {
		writeJSON(w, http.StatusOK, projects)
	}
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetProject(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.Prefix == "" {
		writeError(w, http.StatusBadRequest, "name and prefix are required")
		return
	}
	p, err := s.store.CreateProject(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	p, err := s.store.UpdateProject(chi.URLParam(r, "id"), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.store.GetProject(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := s.store.DeleteProject(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTeams(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	var teams []models.Team
	var total int
	var err error

	if limit > 0 {
		teams, total, err = s.store.ListTeams(limit, offset)
	} else {
		teams, total, err = s.store.ListTeams()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if teams == nil {
		teams = []models.Team{}
	}

	if limit > 0 {
		writeJSON(w, http.StatusOK, models.PaginatedResult[models.Team]{
			Data: teams, Total: total, Limit: limit, Offset: offset,
			HasMore: offset+len(teams) < total,
		})
	} else {
		writeJSON(w, http.StatusOK, teams)
	}
}

func (s *Server) getTeam(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.GetTeam(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) createTeam(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTeamRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	t, err := s.store.CreateTeam(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) updateTeam(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateTeamRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	t, err := s.store.UpdateTeam(chi.URLParam(r, "id"), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) deleteTeam(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := s.store.GetTeam(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "team not found")
		return
	}
	if err := s.store.DeleteTeam(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTickets(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	filter := models.TicketFilter{
		ProjectID: r.URL.Query().Get("projectId"),
		TeamID:    r.URL.Query().Get("teamId"),
		Status:    r.URL.Query().Get("status"),
		Priority:  r.URL.Query().Get("priority"),
		Limit:     limit,
		Offset:    offset,
	}
	tickets, total, err := s.store.ListTickets(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tickets == nil {
		tickets = []models.Ticket{}
	}

	if limit > 0 {
		writeJSON(w, http.StatusOK, models.PaginatedResult[models.Ticket]{
			Data: tickets, Total: total, Limit: limit, Offset: offset,
			HasMore: offset+len(tickets) < total,
		})
	} else {
		writeJSON(w, http.StatusOK, tickets)
	}
}

func (s *Server) getTicket(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.GetTicket(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) createTicket(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ProjectID == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "projectId and title are required")
		return
	}

	t, err := s.store.CreateTicket(req)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) updateTicket(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	id := chi.URLParam(r, "id")
	existing, err := s.store.GetTicket(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	t, err := s.store.UpdateTicket(id, req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (s *Server) moveTicket(w http.ResponseWriter, r *http.Request) {
	var req models.MoveTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}

	id := chi.URLParam(r, "id")
	existing, err := s.store.GetTicket(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	t, err := s.store.MoveTicket(id, req)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (s *Server) deleteTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := s.store.GetTicket(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	if err := s.store.DeleteTicket(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) addSubtask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSubtaskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	st, err := s.store.AddSubtask(chi.URLParam(r, "id"), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

func (s *Server) toggleSubtask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := s.store.ToggleSubtask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if st == nil {
		writeError(w, http.StatusNotFound, "subtask not found")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) deleteSubtask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	st, err := s.store.GetSubtask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if st == nil {
		writeError(w, http.StatusNotFound, "subtask not found")
		return
	}
	if err := s.store.DeleteSubtask(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listLabels(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	var labels []models.Label
	var total int
	var err error

	if limit > 0 {
		labels, total, err = s.store.ListLabels(limit, offset)
	} else {
		labels, total, err = s.store.ListLabels()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if labels == nil {
		labels = []models.Label{}
	}

	if limit > 0 {
		writeJSON(w, http.StatusOK, models.PaginatedResult[models.Label]{
			Data: labels, Total: total, Limit: limit, Offset: offset,
			HasMore: offset+len(labels) < total,
		})
	} else {
		writeJSON(w, http.StatusOK, labels)
	}
}

func (s *Server) createLabel(w http.ResponseWriter, r *http.Request) {
	var req models.CreateLabelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.Color == "" {
		writeError(w, http.StatusBadRequest, "name and color are required")
		return
	}
	l, err := s.store.CreateLabel(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (s *Server) updateLabel(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateLabelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	l, err := s.store.UpdateLabel(chi.URLParam(r, "id"), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if l == nil {
		writeError(w, http.StatusNotFound, "label not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) deleteLabel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l, err := s.store.GetLabel(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if l == nil {
		writeError(w, http.StatusNotFound, "label not found")
		return
	}
	if err := s.store.DeleteLabel(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getBoard(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("projectId")
	board, err := s.store.GetBoard(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, board)
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: sameOrigin,
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK

	if err := s.store.Ping(); err != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name": "player2-kanban", "pid": os.Getpid(),
		"status": status,
		"uptime": time.Since(s.startedAt).String(),
	})
}

func (s *Server) syncStatus(w http.ResponseWriter, r *http.Request) {
	ss, err := s.store.GetSyncStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"githubConfigured": os.Getenv("GITHUB_TOKEN") != "",
		"lastError":        ss.LastError,
		"pendingJobs":      ss.PendingJobs,
		"failedJobs":       ss.FailedJobs,
		"lastSyncAt":       ss.LastSyncAt,
	}

	if rl := github.GetLastRateLimit(); rl != nil {
		resp["rateLimit"] = rl
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) reportActivity(w http.ResponseWriter, r *http.Request) {
	var a models.ActivityRequest
	if err := decodeJSON(r, &a); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	t, err := s.store.ReportActivity(chi.URLParam(r, "id"), a)
	if err != nil {
		writeError(w, 409, err.Error())
		return
	}
	writeJSON(w, 200, t)
}
func (s *Server) completeTicket(w http.ResponseWriter, r *http.Request) {
	var a struct {
		Evidence string `json:"evidence"`
	}
	if err := decodeJSON(r, &a); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	t, err := workflow.Complete(r.Context(), s.store, chi.URLParam(r, "id"), a.Evidence)
	if err != nil {
		writeError(w, 409, err.Error())
		return
	}
	writeJSON(w, 200, t)
}
func (s *Server) resolveConflict(w http.ResponseWriter, r *http.Request) {
	var a struct {
		Choice string `json:"choice"`
	}
	if err := decodeJSON(r, &a); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if os.Getenv("GITHUB_TOKEN") == "" {
		writeError(w, 409, "GITHUB_TOKEN is not configured")
		return
	}
	err := github.ResolveConflict(r.Context(), github.NewClient(r.Context(), os.Getenv("GITHUB_TOKEN")), s.store, chi.URLParam(r, "id"), a.Choice)
	if err != nil {
		writeError(w, 409, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"resolved": true})
}
func (s *Server) retrySync(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RetryFailedSyncJobs(); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"queued": true})
}

func writeStoreError(w http.ResponseWriter, err error) {
	var invalid *db.ValidationError
	if errors.As(err, &invalid) {
		writeError(w, 400, err.Error())
	} else {
		writeError(w, 500, err.Error())
	}
}
