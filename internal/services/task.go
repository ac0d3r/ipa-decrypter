package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/ipatool"
	"github.com/ac0d3r/ipa-decrypter/internal/logger"
	"github.com/ac0d3r/ipa-decrypter/internal/unfaird"
	"github.com/google/uuid"
)

var taskLog = logger.WithComponent("task")

type TaskService struct {
	cfg      config.Config
	db       *db.DB
	accounts *AccountService
	devices  *DeviceService
	mu       sync.Mutex
	running  map[string]struct{}
}

func NewTaskService(cfg config.Config, database *db.DB, accounts *AccountService, devices *DeviceService) *TaskService {
	return &TaskService{
		cfg:      cfg,
		db:       database,
		accounts: accounts,
		devices:  devices,
		running:  map[string]struct{}{},
	}
}

type TaskRequestInput struct {
	BundleID          string `json:"bundle_id"`
	AppName           string `json:"app_name"`
	Region            string `json:"region"`
	ExternalVersionID string `json:"external_version_id"`
	DisplayVersion    string `json:"display_version"`
	IconURL           string `json:"icon_url"`
	SizeBytes         int64  `json:"size_bytes"`
	AppID             int64  `json:"app_id"`
}

func (s *TaskService) List() ([]db.Task, error) {
	return s.db.ListTasks()
}

func (s *TaskService) Get(id string) (db.Task, error) {
	return s.db.GetTask(id)
}

func (s *TaskService) Queue() ([]db.Task, error) {
	return s.db.ListActiveTasks()
}

func (s *TaskService) Cache(limit int) ([]db.Task, error) {
	return s.db.ListCompletedTasks(limit)
}

func (s *TaskService) CachedByBundles(region string, bundleIDs []string) (map[string][]db.Task, error) {
	tasks, err := s.db.ListCompletedTasksForBundles(region, bundleIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]db.Task)
	seen := make(map[string]map[string]bool)
	for _, t := range tasks {
		verKey := t.DisplayVersion
		if verKey == "" {
			verKey = t.ExternalVersionID
		}
		if verKey == "" {
			verKey = t.ID
		}
		if seen[t.BundleID] == nil {
			seen[t.BundleID] = make(map[string]bool)
		}
		if seen[t.BundleID][verKey] {
			continue
		}
		seen[t.BundleID][verKey] = true
		out[t.BundleID] = append(out[t.BundleID], t)
	}
	return out, nil
}

func (s *TaskService) Regions() ([]db.RegionInfo, error) {
	accounts, err := s.accounts.List()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []db.RegionInfo
	for _, a := range accounts {
		if a.Status != "logged_in" || a.Region == "" {
			continue
		}
		code := strings.ToLower(a.Region)
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, db.RegionInfo{Code: code, Label: strings.ToUpper(code)})
	}
	return out, nil
}

func (s *TaskService) Request(in TaskRequestInput) (db.Task, error) {
	in.BundleID = strings.TrimSpace(in.BundleID)
	in.AppName = strings.TrimSpace(in.AppName)
	in.Region = strings.ToLower(strings.TrimSpace(in.Region))
	if in.BundleID == "" {
		return db.Task{}, fmt.Errorf("bundle_id required")
	}
	if in.Region == "" {
		return db.Task{}, fmt.Errorf("region required")
	}
	if in.AppName == "" {
		in.AppName = in.BundleID
	}

	tasks, _ := s.db.ListActiveTasks()
	extVer := strings.TrimSpace(in.ExternalVersionID)
	for _, t := range tasks {
		if t.BundleID == in.BundleID && t.Region == in.Region && t.ExternalVersionID == extVer {
			taskLog.WithFields(map[string]any{
				"task_id":   t.ID,
				"bundle_id": in.BundleID,
				"region":    in.Region,
			}).Info("reusing active task")
			return t, nil
		}
	}

	if failed, ok := s.findReusableFailed(in.BundleID, in.Region, extVer); ok {
		task, err := s.prepareRetry(failed, in)
		if err != nil {
			return db.Task{}, err
		}
		taskLog.WithFields(map[string]any{
			"task_id":   task.ID,
			"bundle_id": task.BundleID,
			"region":    task.Region,
			"path":      task.EncryptedPath,
		}).Info("retrying failed task with existing encrypted ipa")
		go s.run(task.ID)
		return task, nil
	}

	acc, err := s.pickAccount(in.Region)
	if err != nil {
		return db.Task{}, err
	}
	dev, err := s.pickDevice()
	if err != nil {
		return db.Task{}, err
	}

	now := time.Now().UTC()
	task := db.Task{
		ID:                uuid.NewString(),
		BundleID:          in.BundleID,
		AppName:           in.AppName,
		Region:            in.Region,
		ExternalVersionID: extVer,
		DisplayVersion:    strings.TrimSpace(in.DisplayVersion),
		IconURL:           strings.TrimSpace(in.IconURL),
		SizeBytes:         in.SizeBytes,
		AppID:             in.AppID,
		Status:            "queued",
		Detail:            "Queued",
		AccountID:         acc.ID,
		DeviceID:          dev.ID,
		Logs:              []string{"Task queued"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.db.CreateTask(task); err != nil {
		return db.Task{}, err
	}
	taskLog.WithFields(map[string]any{
		"task_id":    task.ID,
		"bundle_id":  task.BundleID,
		"region":     task.Region,
		"account_id": task.AccountID,
		"device_id":  task.DeviceID,
		"version":    task.ExternalVersionID,
	}).Info("task created")
	go s.run(task.ID)
	return task, nil
}

func (s *TaskService) pickAccount(region string) (db.Account, error) {
	accounts, err := s.accounts.List()
	if err != nil {
		return db.Account{}, err
	}
	for _, a := range accounts {
		if a.Status == "logged_in" && strings.EqualFold(a.Region, region) {
			return a, nil
		}
	}
	return db.Account{}, fmt.Errorf("no logged-in account for region %s", region)
}

func (s *TaskService) pickDevice() (db.Device, error) {
	devices, err := s.devices.List()
	if err != nil {
		return db.Device{}, err
	}
	for _, d := range devices {
		if d.Status == "online" {
			return d, nil
		}
	}
	if len(devices) == 0 {
		return db.Device{}, fmt.Errorf("no decrypt device configured (add one in admin)")
	}
	d := devices[0]
	updated, err := s.devices.Health(d.ID)
	if err == nil && updated.Status == "online" {
		return updated, nil
	}
	return db.Device{}, fmt.Errorf("no online decrypt device available")
}

func (s *TaskService) run(id string) {
	s.mu.Lock()
	if _, ok := s.running[id]; ok {
		s.mu.Unlock()
		return
	}
	s.running[id] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, id)
		s.mu.Unlock()
	}()

	task, err := s.db.GetTask(id)
	if err != nil {
		return
	}
	taskLog.WithFields(map[string]any{
		"task_id":   task.ID,
		"bundle_id": task.BundleID,
		"region":    task.Region,
	}).Info("task pipeline started")

	if encryptedReady(&task) {
		s.log(&task, "Reusing encrypted IPA, skipping download")
	} else {
		s.log(&task, "Starting download")
		task.Status = "downloading"
		task.Detail = "Downloading from App Store"
		task.UpdatedAt = time.Now().UTC()
		_ = s.db.UpdateTask(task)

		if err := s.download(&task); err != nil {
			s.fail(&task, err.Error())
			return
		}
	}

	s.log(&task, "Sending to unfaird")
	task.Status = "decrypting"
	task.Detail = "Decrypting on device"
	task.UpdatedAt = time.Now().UTC()
	_ = s.db.UpdateTask(task)

	if err := s.decrypt(&task); err != nil {
		s.fail(&task, err.Error())
		return
	}

	task.Status = "completed"
	task.Detail = "Ready"
	task.Error = ""
	task.UpdatedAt = time.Now().UTC()
	s.log(&task, "Decrypt complete")
	taskLog.WithFields(map[string]any{
		"task_id": task.ID,
		"path":    task.OutputPath,
	}).Info("task completed")
	_ = s.db.UpdateTask(task)
}

func (s *TaskService) download(task *db.Task) error {
	home, err := s.accounts.HomeDir(task.AccountID)
	if err != nil {
		return err
	}

	out := filepath.Join(s.cfg.IPAsDir, task.ID+".ipa")
	_ = os.Remove(out)
	ctx, cancel := ipatool.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	res, runErr := s.accounts.Runner().Download(ctx, home, task.BundleID, task.AppID, out, task.ExternalVersionID, true)
	task.DownloadStdout = res.Stdout
	task.DownloadStderr = res.Stderr
	task.UpdatedAt = time.Now().UTC()

	if runErr != nil || res.ExitCode != 0 {
		msg := firstNonEmpty(errString(runErr), ipatool.CombinedOutput(res))
		taskLog.WithFields(map[string]any{"task_id": task.ID, "error": msg}).Error("download failed")
		_ = s.db.UpdateTask(*task)
		return fmt.Errorf("download: %s", msg)
	}

	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		if alt := findIPA(s.cfg.IPAsDir, task.ID); alt != "" {
			out = alt
		} else {
			taskLog.WithField("task_id", task.ID).Error("download output missing")
			_ = s.db.UpdateTask(*task)
			return fmt.Errorf("download finished but ipa file missing")
		}
	}

	task.EncryptedPath = out
	_ = s.db.UpdateTask(*task)
	taskLog.WithFields(map[string]any{"task_id": task.ID, "path": out}).Info("download succeeded")
	return nil
}

func (s *TaskService) decrypt(task *db.Task) error {
	device, err := s.db.GetDevice(task.DeviceID)
	if err != nil {
		return err
	}

	source := task.EncryptedPath
	if source == "" {
		return fmt.Errorf("encrypted ipa path missing")
	}

	client := unfaird.New(device.BaseURL(), &http.Client{Timeout: 2 * time.Hour})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	taskLog.WithFields(map[string]any{
		"task_id": task.ID,
		"device":  device.BaseURL(),
		"source":  source,
	}).Info("decrypt started")

	resp, err := client.DecryptFile(ctx, source)
	task.UpdatedAt = time.Now().UTC()
	if err != nil {
		taskLog.WithFields(map[string]any{"task_id": task.ID, "error": err.Error()}).Error("decrypt request failed")
		_ = s.db.UpdateTask(*task)
		return fmt.Errorf("decrypt: %w", err)
	}

	code := resp.Exit.Code
	task.DecryptExitCode = &code
	task.DecryptStdout = resp.Exit.Stdout
	task.DecryptStderr = resp.Exit.Stderr
	task.UnfairdDownloadURL = resp.Exit.DownloadURL
	task.ValidateUntil = resp.Exit.ValidateUntil
	task.UnfairdJobID = unfaird.JobIDFromDownloadURL(resp.Exit.DownloadURL)

	if code != 0 {
		taskLog.WithFields(map[string]any{"task_id": task.ID, "exit_code": code}).Error("unfaird decrypt failed")
		_ = s.db.UpdateTask(*task)
		return fmt.Errorf("unfaird exit code %d", code)
	}

	local := filepath.Join(s.cfg.DecryptedDir, task.ID+".ipa")
	if err := client.DownloadOutput(ctx, resp.Exit.DownloadURL, local); err != nil {
		taskLog.WithFields(map[string]any{"task_id": task.ID, "error": err.Error()}).Error("decrypt output fetch failed")
		_ = s.db.UpdateTask(*task)
		return fmt.Errorf("decrypt ok but fetch output failed: %w", err)
	}

	if st, err := os.Stat(local); err == nil {
		task.SizeBytes = st.Size()
	}
	task.OutputPath = local
	if enc := task.EncryptedPath; enc != "" {
		if err := os.Remove(enc); err != nil && !os.IsNotExist(err) {
			taskLog.WithFields(map[string]any{"task_id": task.ID, "path": enc, "error": err.Error()}).Warn("failed to remove encrypted ipa")
		} else {
			task.EncryptedPath = ""
			s.log(task, "Removed encrypted IPA")
		}
	}
	_ = s.db.UpdateTask(*task)
	taskLog.WithFields(map[string]any{"task_id": task.ID, "path": local}).Info("decrypt succeeded")
	return nil
}

func (s *TaskService) findReusableFailed(bundleID, region, externalVersionID string) (db.Task, bool) {
	tasks, err := s.db.ListFailedTasksForVersion(bundleID, region, externalVersionID)
	if err != nil {
		return db.Task{}, false
	}
	for _, t := range tasks {
		if encryptedReady(&t) {
			return t, true
		}
	}
	return db.Task{}, false
}

func (s *TaskService) prepareRetry(failed db.Task, in TaskRequestInput) (db.Task, error) {
	acc, err := s.pickAccount(in.Region)
	if err != nil {
		return db.Task{}, err
	}
	dev, err := s.pickDevice()
	if err != nil {
		return db.Task{}, err
	}

	now := time.Now().UTC()
	failed.Status = "queued"
	failed.Error = ""
	failed.Detail = "Queued"
	failed.AppName = in.AppName
	failed.DisplayVersion = strings.TrimSpace(in.DisplayVersion)
	failed.IconURL = strings.TrimSpace(in.IconURL)
	if in.SizeBytes > 0 {
		failed.SizeBytes = in.SizeBytes
	}
	failed.AppID = in.AppID
	failed.AccountID = acc.ID
	failed.DeviceID = dev.ID
	failed.DecryptExitCode = nil
	failed.DecryptStdout = ""
	failed.DecryptStderr = ""
	failed.UnfairdJobID = ""
	failed.UnfairdDownloadURL = ""
	failed.ValidateUntil = 0
	failed.OutputPath = ""
	failed.UpdatedAt = now
	s.log(&failed, "Retry queued (encrypted IPA on disk)")

	if err := s.db.UpdateTask(failed); err != nil {
		return db.Task{}, err
	}
	return failed, nil
}

func encryptedReady(task *db.Task) bool {
	if task.EncryptedPath == "" {
		return false
	}
	st, err := os.Stat(task.EncryptedPath)
	return err == nil && st.Size() > 0
}

func findIPA(dir, taskID string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	exact := filepath.Join(dir, taskID+".ipa")
	if st, err := os.Stat(exact); err == nil && st.Size() > 0 {
		return exact
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, taskID) && strings.HasSuffix(strings.ToLower(name), ".ipa") {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

func (s *TaskService) log(task *db.Task, line string) {
	ts := time.Now().UTC().Format("15:04:05")
	task.Logs = append(task.Logs, ts+" "+line)
	if len(task.Logs) > 200 {
		task.Logs = task.Logs[len(task.Logs)-200:]
	}
}

func (s *TaskService) fail(task *db.Task, msg string) {
	task.Status = "failed"
	task.Error = msg
	task.Detail = "Failed"
	task.UpdatedAt = time.Now().UTC()
	s.log(task, msg)
	taskLog.WithFields(map[string]any{
		"task_id":   task.ID,
		"bundle_id": task.BundleID,
		"error":     msg,
	}).Error("task failed")
	_ = s.db.UpdateTask(*task)
}
