package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/itunes"
	"github.com/ac0d3r/ipa-decrypter/internal/services"
	"github.com/gin-gonic/gin"
)

func (s *Server) registerPublicRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	v1.GET("/queue", s.handleQueue)
	v1.GET("/cache", s.handleCache)
	v1.GET("/regions", s.handleRegions)
	v1.GET("/search", s.rateLimit(searchRateMax), s.handleSearch)

	authed := v1.Group("", s.requireAuth())
	authed.GET("/versions", s.rateLimit(versionRateMax), s.handleVersions)
	authed.POST("/request", s.rateLimit(coreAPIRateMax), s.quotaMiddleware("decrypt"), s.handleRequest)
	authed.GET("/tasks/:id", s.handleTask)
	authed.HEAD("/tasks/:id/file", s.quotaMiddleware("download"), s.handleTaskFileHead)
	authed.GET("/tasks/:id/file", s.rateLimit(coreAPIRateMax), s.quotaMiddleware("download"), s.handleTaskFile)
}

func (s *Server) handleSearch(c *gin.Context) {
	region := c.Query("region")
	bundleID := strings.TrimSpace(c.Query("bundle_id"))
	q := strings.TrimSpace(c.Query("q"))

	var (
		results []itunes.AppResult
		err     error
	)
	switch {
	case bundleID != "":
		results, err = itunes.Lookup(bundleID, region)
	case q != "":
		limit, _ := strconv.Atoi(c.Query("limit"))
		results, err = itunes.Search(q, region, limit)
	default:
		writeErr(c, errStatus(http.StatusBadRequest, "bundle_id or q required"))
		return
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (s *Server) handleRegions(c *gin.Context) {
	regions, err := s.Tasks.Regions()
	if err != nil {
		writeErr(c, err)
		return
	}
	if regions == nil {
		regions = []db.RegionInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"regions": regions})
}

func (s *Server) handleVersions(c *gin.Context) {
	bundleID := c.Query("bundle_id")
	region := c.Query("region")
	appID, _ := strconv.ParseInt(c.Query("app_id"), 10, 64)
	offset, _ := strconv.Atoi(c.Query("offset"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := s.Versions.ListPage(bundleID, region, appID, offset, limit)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (s *Server) handleRequest(c *gin.Context) {
	var in services.TaskRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeErr(c, err)
		return
	}
	t, err := s.Tasks.Request(currentUserID(c), in)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"task_id": t.ID,
		"status":  t.Status,
		"task":    t,
	})
}

func (s *Server) handleQueue(c *gin.Context) {
	q, err := s.Tasks.Queue()
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"queue": q, "workers": 1})
}

func (s *Server) handleCache(c *gin.Context) {
	bundleIDs := strings.TrimSpace(c.Query("bundle_ids"))
	if bundleIDs != "" {
		region := strings.ToLower(strings.TrimSpace(c.Query("region")))
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
			writeErr(c, err)
			return
		}
		if byBundle == nil {
			byBundle = map[string][]db.Task{}
		}
		c.JSON(http.StatusOK, gin.H{"by_bundle": byBundle})
		return
	}
	items, err := s.Tasks.Cache(20)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"cache": items})
}

func (s *Server) handleTask(c *gin.Context) {
	t, err := s.Tasks.Get(c.Param("id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":        t.ID,
		"bundle_id": t.BundleID,
		"status":    t.Status,
		"error":     t.Error,
		"logs":      t.Logs,
	})
}

func (s *Server) taskFileReady(c *gin.Context) (db.Task, error) {
	t, err := s.Tasks.Get(c.Param("id"))
	if err != nil {
		return t, err
	}
	if t.Status != "completed" || t.OutputPath == "" {
		return t, errStatus(http.StatusConflict, "task not ready")
	}
	if _, err := os.Stat(t.OutputPath); err != nil {
		return t, errStatus(http.StatusNotFound, "file not found")
	}
	return t, nil
}

func (s *Server) handleTaskFileHead(c *gin.Context) {
	if _, err := s.taskFileReady(c); err != nil {
		writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleTaskFile(c *gin.Context) {
	t, err := s.taskFileReady(c)
	if err != nil {
		writeErr(c, err)
		return
	}
	if err := s.Auth.ConsumeUserQuota(currentUserID(c), "download"); err != nil {
		writeErr(c, err)
		return
	}
	c.FileAttachment(t.OutputPath, t.BundleID+".ipa")
}
