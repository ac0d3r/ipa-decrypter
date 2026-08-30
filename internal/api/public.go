package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/itunes"
	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/search", s.handleSearch)
	mux.HandleFunc("GET /api/v1/regions", s.handleRegions)
	mux.HandleFunc("GET /api/v1/versions", s.handleVersions)
	mux.HandleFunc("GET /api/v1/cache", s.handleCache)
	mux.HandleFunc("POST /api/v1/request", s.handleRequest)
	mux.HandleFunc("GET /api/v1/queue", s.handleQueue)
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.handleTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}/file", s.handleTaskFile)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	bundleID := strings.TrimSpace(r.URL.Query().Get("bundle_id"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	var (
		results []itunes.AppResult
		err     error
	)
	if bundleID != "" {
		results, err = itunes.Lookup(bundleID, region)
	} else if q != "" {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		results, err = itunes.Search(q, region, limit)
	} else {
		writeErr(w, errStatus(http.StatusBadRequest, "bundle_id or q required"))
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleRegions(w http.ResponseWriter, r *http.Request) {
	regions, err := s.Tasks.Regions()
	if err != nil {
		writeErr(w, err)
		return
	}
	if regions == nil {
		regions = []db.RegionInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"regions": regions})
}

func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request) {
	bundleID := r.URL.Query().Get("bundle_id")
	region := r.URL.Query().Get("region")
	appID, _ := strconv.ParseInt(r.URL.Query().Get("app_id"), 10, 64)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := s.Versions.ListPage(bundleID, region, appID, offset, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	var in services.TaskRequestInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, err)
		return
	}
	t, err := s.Tasks.Request(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task_id": t.ID,
		"status":  t.Status,
		"task":    t,
	})
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	q, err := s.Tasks.Queue()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"queue": q, "workers": 1})
}

func (s *Server) handleCache(w http.ResponseWriter, r *http.Request) {
	bundleIDs := strings.TrimSpace(r.URL.Query().Get("bundle_ids"))
	if bundleIDs != "" {
		region := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("region")))
		ids := strings.Split(bundleIDs, ",")
		var trimmed []string
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				trimmed = append(trimmed, id)
			}
		}
		byBundle, err := s.Tasks.CachedByBundles(region, trimmed)
		if err != nil {
			writeErr(w, err)
			return
		}
		if byBundle == nil {
			byBundle = map[string][]db.Task{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"by_bundle": byBundle})
		return
	}
	items, err := s.Tasks.Cache(20)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cache": items})
}

func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.Tasks.Get(id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleTaskFile(w http.ResponseWriter, r *http.Request) {
	t, err := s.Tasks.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	if t.Status != "completed" || t.OutputPath == "" {
		writeErr(w, errStatus(http.StatusConflict, "task not ready"))
		return
	}
	name := t.BundleID + ".ipa"
	serveFile(w, r, t.OutputPath, name)
}
