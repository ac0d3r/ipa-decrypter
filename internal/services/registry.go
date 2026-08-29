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
}

func NewRegistry(cfg config.Config, database *db.DB) *Registry {
	accounts := NewAccountService(cfg, database)
	devices := NewDeviceService(database)
	tasks := NewTaskService(cfg, database, accounts, devices)
	versions := NewVersionService(accounts)
	return &Registry{
		Accounts: accounts,
		Devices:  devices,
		Tasks:    tasks,
		Versions: versions,
	}
}
