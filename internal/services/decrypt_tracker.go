package services

import "sync"

// decryptTracker: one in-flight decrypt per user. Bind only when this
// user starts work (create or retry); reuse of an active task does not bind.
type decryptTracker struct {
	mu       sync.Mutex
	bindings map[string]string // userID -> taskID
}

func newDecryptTracker() *decryptTracker {
	return &decryptTracker{bindings: map[string]string{}}
}

func (t *decryptTracker) lock()   { t.mu.Lock() }
func (t *decryptTracker) unlock() { t.mu.Unlock() }

func (t *decryptTracker) boundTask(userID string) string {
	return t.bindings[userID]
}

func (t *decryptTracker) bind(userID, taskID string) {
	t.bindings[userID] = taskID
}

func (t *decryptTracker) drop(taskID string) {
	for uid, id := range t.bindings {
		if id == taskID {
			delete(t.bindings, uid)
			return
		}
	}
}

func (t *decryptTracker) finish(taskID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.drop(taskID)
}
