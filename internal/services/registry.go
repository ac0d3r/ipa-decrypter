package services

import (
	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
)

// Registry wires all application services.
type Registry struct {
	Accounts *AccountService
	Devices  *DeviceService
	Tasks    *TaskService
	Versions *VersionService
	Auth     *AuthService
}

func NewRegistry(cfg config.Config, database *db.DB) *Registry {
	accounts := NewAccountService(cfg, database)
	devices := NewDeviceService(database)
	auth := NewAuthService(cfg, database)
	tasks := NewTaskService(cfg, database, accounts, devices, auth)
	versions := NewVersionService(accounts)
	return &Registry{
		Accounts: accounts,
		Devices:  devices,
		Tasks:    tasks,
		Versions: versions,
		Auth:     auth,
	}
}
