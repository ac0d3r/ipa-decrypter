package services

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/google/uuid"
)

func testTasks(t *testing.T) (*TaskService, *db.DB) {
	t.Helper()
	dir := t.TempDir()
	database, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	cfg := config.Config{
		IPAsDir:      filepath.Join(dir, "ipas"),
		DecryptedDir: filepath.Join(dir, "decrypted"),
		IPATOOLBin:   filepath.Join(dir, "missing-ipatool"),
	}
	if err := os.MkdirAll(cfg.IPAsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.DecryptedDir, 0o700); err != nil {
		t.Fatal(err)
	}
	s := NewTaskService(cfg, database, NewAccountService(cfg, database), NewDeviceService(database))
	t.Cleanup(func() { waitIdle(t, s) })
	return s, database
}

func seedAccountDevice(t *testing.T, database *db.DB) {
	t.Helper()
	now := time.Now().UTC()
	if err := database.CreateAccount(db.AccountRow{Account: db.Account{
		ID: "acc-us", Email: "a@example.com", Region: "us", Status: "logged_in",
		CreatedAt: now, UpdatedAt: now,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := database.CreateDevice(db.Device{
		ID: "dev-1", Name: "d", Host: "127.0.0.1", Port: 6347, Status: "online",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}

func waitIdle(t *testing.T, s *TaskService) {
	t.Helper()
	time.Sleep(30 * time.Millisecond)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		n := 0
		s.running.Range(func(_, _ any) bool { n++; return true })
		if n == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("task pipeline still running")
}

func TestDeleteTaskRemovesFiles(t *testing.T) {
	s, database := testTasks(t)
	out := filepath.Join(s.cfg.DecryptedDir, "done.ipa")
	enc := filepath.Join(s.cfg.IPAsDir, "enc.ipa")
	if err := os.WriteFile(out, []byte("out"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(enc, []byte("enc"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	existing := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		Status: "completed", OutputPath: out, EncryptedPath: enc,
		Logs: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(existing); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(existing.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetTask(existing.ID); err == nil {
		t.Fatal("expected task to be gone")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output still exists: %v", err)
	}
	if _, err := os.Stat(enc); !os.IsNotExist(err) {
		t.Fatalf("encrypted still exists: %v", err)
	}
}

func TestDeleteActiveTaskRejected(t *testing.T) {
	s, database := testTasks(t)
	now := time.Now().UTC()
	existing := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		Status: "queued", Logs: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(existing); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(existing.ID); !errors.Is(err, ErrTaskNotDeletable) {
		t.Fatalf("err = %v, want ErrTaskNotDeletable", err)
	}
	if _, err := database.GetTask(existing.ID); err != nil {
		t.Fatal(err)
	}
}

func TestRequestValidatesInput(t *testing.T) {
	s, _ := testTasks(t)
	if _, err := s.Request(TaskRequestInput{Region: "us"}); err == nil {
		t.Fatal("expected bundle_id required")
	}
	if _, err := s.Request(TaskRequestInput{BundleID: "com.app"}); err == nil {
		t.Fatal("expected region required")
	}
}

func TestRequestReusesActiveTask(t *testing.T) {
	s, database := testTasks(t)
	now := time.Now().UTC()
	existing := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		ExternalVersionID: "v1", Status: "downloading", Logs: []string{},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(existing); err != nil {
		t.Fatal(err)
	}
	s.running.Store(existing.ID, struct{}{})
	t.Cleanup(func() { s.running.Delete(existing.ID) })

	got, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "US", ExternalVersionID: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != existing.ID {
		t.Fatalf("id = %s, want %s", got.ID, existing.ID)
	}
}

func TestRequestReusesCompletedTask(t *testing.T) {
	s, database := testTasks(t)
	out := filepath.Join(s.cfg.DecryptedDir, "done.ipa")
	if err := os.WriteFile(out, []byte("ipa"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	existing := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		ExternalVersionID: "v3", DisplayVersion: "1.2.3", Status: "completed",
		OutputPath: out, Logs: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(existing); err != nil {
		t.Fatal(err)
	}

	got, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "us", ExternalVersionID: "v3"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != existing.ID {
		t.Fatalf("id = %s, want %s", got.ID, existing.ID)
	}
	if got.Status != "completed" {
		t.Fatalf("status = %s, want completed", got.Status)
	}

	latest, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "us", DisplayVersion: "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != existing.ID {
		t.Fatalf("latest id = %s, want %s", latest.ID, existing.ID)
	}
}

func TestRequestDoesNotReuseCompletedWithoutVersion(t *testing.T) {
	s, database := testTasks(t)
	seedAccountDevice(t, database)
	out := filepath.Join(s.cfg.DecryptedDir, "old.ipa")
	if err := os.WriteFile(out, []byte("ipa"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	existing := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		Status: "completed", OutputPath: out, Logs: []string{},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(existing); err != nil {
		t.Fatal(err)
	}

	got, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "us", AppName: "App"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == existing.ID {
		t.Fatal("reused completed task without version identity")
	}
	if got.Status != "queued" {
		t.Fatalf("status = %s, want queued", got.Status)
	}
}

func TestRequestCompletedMissingFileCreatesNew(t *testing.T) {
	s, database := testTasks(t)
	seedAccountDevice(t, database)
	now := time.Now().UTC()
	existing := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		ExternalVersionID: "v4", Status: "completed",
		OutputPath: filepath.Join(s.cfg.DecryptedDir, "missing.ipa"),
		Logs:       []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(existing); err != nil {
		t.Fatal(err)
	}

	got, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "us", ExternalVersionID: "v4", AppName: "App"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == existing.ID {
		t.Fatal("reused completed task whose file is gone")
	}
	if got.Status != "queued" {
		t.Fatalf("status = %s, want queued", got.Status)
	}
}

func TestRequestRetriesFailedWithEncryptedIPA(t *testing.T) {
	s, database := testTasks(t)
	seedAccountDevice(t, database)

	enc := filepath.Join(s.cfg.IPAsDir, "old.ipa")
	if err := os.WriteFile(enc, []byte("ipa"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	failed := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		ExternalVersionID: "v2", Status: "failed", EncryptedPath: enc,
		Error: "unfaird exit code 1", Logs: []string{"failed"},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(failed); err != nil {
		t.Fatal(err)
	}

	got, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "us", ExternalVersionID: "v2", AppName: "App"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != failed.ID {
		t.Fatalf("id = %s, want reused %s", got.ID, failed.ID)
	}
	if got.Status != "queued" {
		t.Fatalf("status = %s, want queued", got.Status)
	}
}

func TestRequestConcurrentSameVersionCreatesOnce(t *testing.T) {
	s, database := testTasks(t)
	now := time.Now().UTC()
	existing := db.Task{
		ID: uuid.NewString(), BundleID: "com.app", AppName: "App", Region: "us",
		Status: "queued", Logs: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.CreateTask(existing); err != nil {
		t.Fatal(err)
	}
	s.running.Store(existing.ID, struct{}{})
	t.Cleanup(func() { s.running.Delete(existing.ID) })

	const n = 8
	ids := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			got, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "us", AppName: "App"})
			errs[i] = err
			if err == nil {
				ids[i] = got.ID
			}
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}
	for i, id := range ids {
		if id != existing.ID {
			t.Fatalf("request %d id = %q, want %q", i, id, existing.ID)
		}
	}
}

func TestRequestNeedsAccountAndDevice(t *testing.T) {
	s, _ := testTasks(t)
	if _, err := s.Request(TaskRequestInput{BundleID: "com.app", Region: "us"}); err == nil {
		t.Fatal("expected no account error")
	}
}
