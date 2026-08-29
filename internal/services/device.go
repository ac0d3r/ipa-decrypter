package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/logger"
	"github.com/ac0d3r/ipa-decrypter/internal/unfaird"
	"github.com/google/uuid"
)

var deviceLog = logger.WithComponent("device")

type DeviceService struct {
	db *db.DB
}

func NewDeviceService(database *db.DB) *DeviceService {
	return &DeviceService{db: database}
}

type DeviceInput struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (s *DeviceService) List() ([]db.Device, error) {
	return s.db.ListDevices()
}

func (s *DeviceService) Get(id string) (db.Device, error) {
	return s.db.GetDevice(id)
}

func (s *DeviceService) Create(in DeviceInput) (db.Device, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Host = strings.TrimSpace(in.Host)
	if in.Host == "" {
		return db.Device{}, fmt.Errorf("host required")
	}
	if in.Port <= 0 {
		in.Port = 6347
	}
	if in.Name == "" {
		in.Name = in.Host
	}
	now := time.Now().UTC()
	d := db.Device{
		ID:        uuid.NewString(),
		Name:      in.Name,
		Host:      in.Host,
		Port:      in.Port,
		Status:    "unknown",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.CreateDevice(d); err != nil {
		return db.Device{}, err
	}
	deviceLog.WithFields(map[string]any{"device_id": d.ID, "host": d.Host, "port": d.Port}).Info("device created")
	return s.Health(d.ID)
}

func (s *DeviceService) Update(id string, in DeviceInput) (db.Device, error) {
	d, err := s.db.GetDevice(id)
	if err != nil {
		return db.Device{}, err
	}
	if v := strings.TrimSpace(in.Name); v != "" {
		d.Name = v
	}
	if v := strings.TrimSpace(in.Host); v != "" {
		d.Host = v
	}
	if in.Port > 0 {
		d.Port = in.Port
	}
	d.UpdatedAt = time.Now().UTC()
	if err := s.db.UpdateDevice(d); err != nil {
		return db.Device{}, err
	}
	return s.Health(id)
}

func (s *DeviceService) Delete(id string) error {
	return s.db.DeleteDevice(id)
}

func (s *DeviceService) Health(id string) (db.Device, error) {
	d, err := s.db.GetDevice(id)
	if err != nil {
		return db.Device{}, err
	}
	client := unfaird.New(d.BaseURL(), &http.Client{Timeout: 5 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d.UpdatedAt = time.Now().UTC()
	if err := client.Health(ctx); err != nil {
		d.Status = "offline"
		d.LastError = err.Error()
		deviceLog.WithFields(map[string]any{"device_id": id, "url": d.BaseURL(), "error": err.Error()}).Warn("device health check failed")
	} else {
		d.Status = "online"
		d.LastError = ""
		deviceLog.WithFields(map[string]any{"device_id": id, "url": d.BaseURL()}).Debug("device health check ok")
	}
	_ = s.db.UpdateDevice(d)
	return d, nil
}
