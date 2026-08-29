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
	mux.HandleFunc("GET /api/v1/lookup", s.handleLookup)
	mux.HandleFunc("GET /api/v1/search", s.handleSearch)
	mux.HandleFunc("GET /api/v1/regions", s.handleRegions)
	mux.HandleFunc("GET /api/v1/versions", s.handleVersions)
	mux.HandleFunc("POST /api/v1/request", s.handleRequest)
	mux.HandleFunc("GET /api/v1/queue", s.handleQueue)
	mux.HandleFunc("GET /api/v1/cache", s.handleCache)
	mux.HandleFunc("GET /api/v1/jobs/{id}", s.handleJob)
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.handleJob)
	mux.HandleFunc("GET /api/v1/tasks/{id}/file", s.handleTaskFile)
	mux.HandleFunc("GET /api/v1/download/{bundle_id}", s.handleDownloadByBundle)
}

func (s *Server) handleLookup(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.URL.Query().Get("bundle_id")
	}
	region := r.URL.Query().Get("region")
	results, err := itunes.Lookup(id, region)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	region := r.URL.Query().Get("region")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := itunes.Search(q, region, limit)
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
	if bundleID == "" {
		bundleID = r.URL.Query().Get("id")
	}
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
		"job_id": t.ID,
		"status": t.Status,
		"task":   t,
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

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleDownloadByBundle(w http.ResponseWriter, r *http.Request) {
	bundleID := r.PathValue("bundle_id")
	region := r.URL.Query().Get("region")
	tasks, err := s.Tasks.Cache(50)
	if err != nil {
		writeErr(w, err)
		return
	}
	for _, t := range tasks {
		if t.BundleID == bundleID && t.Status == "completed" && t.OutputPath != "" {
			if region == "" || t.Region == region {
				serveFile(w, r, t.OutputPath, bundleID+".ipa")
				return
			}
		}
	}
	writeErr(w, errStatus(http.StatusNotFound, "cached ipa not found"))
}
