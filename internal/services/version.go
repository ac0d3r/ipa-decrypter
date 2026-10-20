package services

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/ipatool"
	"github.com/ac0d3r/ipa-decrypter/internal/itunes"
	"github.com/ac0d3r/ipa-decrypter/internal/logger"
)

var versionLog = logger.WithComponent("version")

const (
	versionPageSize   = 5
	versionIDCacheTTL = 30 * time.Minute
	// Upper bound of concurrent ipatool metadata calls per page.
	metadataConcurrency = 4
)

type VersionService struct {
	accounts *AccountService
	idCache  sync.Map // key -> cachedVersionIDs
}

type cachedVersionIDs struct {
	ids     []string
	appID   int64
	expires time.Time
}

type VersionPage struct {
	Versions []AppVersion `json:"versions"`
	Total    int          `json:"total"`
	Offset   int          `json:"offset"`
	Limit    int          `json:"limit"`
	HasMore  bool         `json:"has_more"`
}

func NewVersionService(accounts *AccountService) *VersionService {
	return &VersionService{accounts: accounts}
}

type AppVersion struct {
	ExternalVersionID string `json:"external_version_id"`
	DisplayVersion    string `json:"display_version,omitempty"`
	ReleaseDate       string `json:"release_date,omitempty"`
	Latest            bool   `json:"latest,omitempty"`
}

func (s *VersionService) ListPage(bundleID, region string, appID int64, offset, limit int) (VersionPage, error) {
	bundleID = strings.TrimSpace(bundleID)
	region = strings.ToLower(strings.TrimSpace(region))
	if bundleID == "" && appID <= 0 {
		return VersionPage{}, fmt.Errorf("bundle_id or app_id required")
	}
	if region == "" {
		return VersionPage{}, fmt.Errorf("region required")
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = versionPageSize
	}
	if limit > versionPageSize {
		limit = versionPageSize
	}

	_, home, err := s.accountForRegion(region)
	if err != nil {
		return VersionPage{}, err
	}

	appID = s.resolveAppID(bundleID, region, appID)
	ids, err := s.versionIDs(bundleID, region, appID, home)
	if err != nil {
		return VersionPage{}, err
	}
	total := len(ids)
	if total == 0 {
		return VersionPage{Versions: []AppVersion{}, Total: 0, Offset: offset, Limit: limit}, nil
	}
	if offset >= total {
		return VersionPage{Versions: []AppVersion{}, Total: total, Offset: offset, Limit: limit}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}
	pageIDs := ids[offset:end]
	versions, err := s.fetchMetadata(home, bundleID, appID, pageIDs)
	if err != nil {
		return VersionPage{}, err
	}

	versionLog.WithFields(map[string]any{
		"bundle_id": bundleID,
		"offset":    offset,
		"limit":     limit,
		"total":     total,
		"count":     len(versions),
	}).Info("versions page listed")

	return VersionPage{
		Versions: versions,
		Total:    total,
		Offset:   offset,
		Limit:    limit,
		HasMore:  end < total,
	}, nil
}

func (s *VersionService) versionIDs(bundleID, region string, appID int64, home string) ([]string, error) {
	key := versionCacheKey(bundleID, region, appID)
	if v, ok := s.idCache.Load(key); ok {
		entry := v.(cachedVersionIDs)
		if time.Now().Before(entry.expires) {
			return append([]string(nil), entry.ids...), nil
		}
		s.idCache.Delete(key)
	}

	versionLog.WithFields(map[string]any{
		"bundle_id": bundleID,
		"region":    region,
		"app_id":    appID,
	}).Info("listing version ids")

	ctx, cancel := ipatool.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ids, err := s.listVersionIDs(ctx, home, bundleID, appID)
	if err != nil && looksLikeLicenseRequired(err.Error()) {
		if acqErr := s.acquireLicense(ctx, home, bundleID); acqErr != nil {
			return nil, acqErr
		}
		s.idCache.Delete(key)
		ids, err = s.listVersionIDs(ctx, home, bundleID, appID)
	}
	if err != nil {
		return nil, err
	}
	sortVersionIDsDesc(ids)

	s.idCache.Store(key, cachedVersionIDs{
		ids:     append([]string(nil), ids...),
		appID:   appID,
		expires: time.Now().Add(versionIDCacheTTL),
	})
	return ids, nil
}

func (s *VersionService) listVersionIDs(ctx context.Context, home, bundleID string, appID int64) ([]string, error) {
	res, err := s.accounts.Runner().ListVersions(ctx, home, bundleID, appID)
	if err == nil && res.ExitCode == 0 {
		log, ok := ipatool.LastSuccessLog(res.Stdout)
		if !ok {
			return nil, fmt.Errorf("list-versions: no success log in output")
		}
		return ipatool.StringSliceField(log, "externalVersionIdentifiers"), nil
	}
	msg := ipatool.ResultMessage(res, err)
	if looksLikeLicenseRequired(msg) {
		versionLog.WithFields(map[string]any{"bundle_id": bundleID, "app_id": appID}).Info("license required for list-versions")
		return nil, fmt.Errorf("license is required")
	}
	if looksLikeAppleUnknownError(msg) {
		versionLog.WithFields(map[string]any{"bundle_id": bundleID, "app_id": appID, "error": msg}).Warn("list-versions unavailable for app")
		return []string{}, nil
	}
	versionLog.WithFields(map[string]any{"bundle_id": bundleID, "app_id": appID, "error": msg}).Error("list-versions failed")
	return nil, fmt.Errorf("list-versions failed: %s", msg)
}

func (s *VersionService) acquireLicense(ctx context.Context, home, bundleID string) error {
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return fmt.Errorf("license is required (bundle id missing for purchase)")
	}
	versionLog.WithField("bundle_id", bundleID).Info("acquiring app license")
	res, err := s.accounts.Runner().Purchase(ctx, home, bundleID)
	msg := ipatool.ResultMessage(res, err)
	if err == nil && res.ExitCode == 0 {
		return nil
	}
	if looksLikeLicenseAlreadyOwned(msg) {
		return nil
	}
	if looksLikeAppleHTTP204(msg) {
		versionLog.WithField("bundle_id", bundleID).Info("purchase returned HTTP 204, treating as success")
		return nil
	}
	versionLog.WithFields(map[string]any{"bundle_id": bundleID, "error": msg}).Warn("purchase failed")
	if msg == "" {
		msg = fmt.Sprintf("exit code %d (no output from ipatool)", res.ExitCode)
	}
	if looksLikeAppleUnknownError(msg) {
		return fmt.Errorf("purchase failed: Apple returned an unknown error (often paid apps, region mismatch, or Apple-owned apps): %s", msg)
	}
	return fmt.Errorf("license is required: purchase failed: %s", msg)
}

func (s *VersionService) resolveAppID(bundleID, region string, appID int64) int64 {
	if appID > 0 {
		return appID
	}
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return 0
	}
	results, err := itunes.Lookup(bundleID, region)
	if err != nil || len(results) == 0 {
		return 0
	}
	return results[0].AppID
}

func (s *VersionService) fetchMetadata(home, bundleID string, appID int64, ids []string) ([]AppVersion, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	type item struct {
		idx int
		v   AppVersion
	}
	ch := make(chan item, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, metadataConcurrency)

	for i, id := range ids {
		wg.Add(1)
		go func(idx int, externalID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			v := AppVersion{ExternalVersionID: externalID}
			metaCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			metaRes, metaErr := s.accounts.Runner().GetVersionMetadata(metaCtx, home, bundleID, appID, externalID)
			if metaErr == nil && metaRes.ExitCode == 0 {
				if metaLog, ok := ipatool.LastSuccessLog(metaRes.Stdout); ok {
					v.DisplayVersion = ipatool.StringField(metaLog, "displayVersion")
					v.ReleaseDate = ipatool.StringField(metaLog, "releaseDate")
				}
			}
			ch <- item{idx: idx, v: v}
		}(i, id)
	}
	wg.Wait()
	close(ch)

	out := make([]AppVersion, len(ids))
	for it := range ch {
		out[it.idx] = it.v
	}
	return out, nil
}

func versionCacheKey(bundleID, region string, appID int64) string {
	if bundleID != "" {
		return region + "|" + bundleID
	}
	return fmt.Sprintf("%s|app:%d", region, appID)
}

func sortVersionIDsDesc(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		a, errA := strconv.ParseInt(ids[i], 10, 64)
		b, errB := strconv.ParseInt(ids[j], 10, 64)
		if errA == nil && errB == nil {
			return a > b
		}
		return ids[i] > ids[j]
	})
}

func (s *VersionService) accountForRegion(region string) (db.Account, string, error) {
	accounts, err := s.accounts.List()
	if err != nil {
		return db.Account{}, "", err
	}
	for i := range accounts {
		a := &accounts[i]
		if a.Status == "logged_in" && strings.EqualFold(a.Region, region) {
			home, err := s.accounts.HomeDir(a.ID)
			if err != nil {
				return db.Account{}, "", err
			}
			return *a, home, nil
		}
	}
	return db.Account{}, "", fmt.Errorf("no logged-in account for region %s", region)
}
