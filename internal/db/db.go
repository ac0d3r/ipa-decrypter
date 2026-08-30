package db

import (
	"encoding/json"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	gorm *gorm.DB
}

func Open(path string) (*DB, error) {
	g, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := g.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	d := &DB{gorm: g}
	if err := d.migrate(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error {
	sqlDB, err := d.gorm.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *DB) migrate() error {
	return d.gorm.AutoMigrate(
		&AccountRow{},
		&Device{},
		&Secret{},
		&Task{},
		&AdminUser{},
	)
}

func (d *DB) CreateAccount(a AccountRow) error {
	return d.gorm.Create(&a).Error
}

func (d *DB) UpdateAccount(a AccountRow) error {
	return d.gorm.Save(&a).Error
}

func (d *DB) GetAccount(id string) (AccountRow, error) {
	var a AccountRow
	err := d.gorm.First(&a, "id = ?", id).Error
	return a, err
}

func (d *DB) ListAccounts() ([]AccountRow, error) {
	var out []AccountRow
	err := d.gorm.Order("created_at DESC").Find(&out).Error
	return out, err
}

func (d *DB) DeleteAccount(id string) error {
	return d.gorm.Delete(&AccountRow{}, "id = ?", id).Error
}

func (d *DB) CreateDevice(dev Device) error {
	return d.gorm.Create(&dev).Error
}

func (d *DB) UpdateDevice(dev Device) error {
	return d.gorm.Save(&dev).Error
}

func (d *DB) GetDevice(id string) (Device, error) {
	var dev Device
	err := d.gorm.First(&dev, "id = ?", id).Error
	return dev, err
}

func (d *DB) ListDevices() ([]Device, error) {
	var out []Device
	err := d.gorm.Order("created_at DESC").Find(&out).Error
	return out, err
}

func (d *DB) DeleteDevice(id string) error {
	return d.gorm.Delete(&Device{}, "id = ?", id).Error
}

func (d *DB) SetSecret(key, value string) error {
	return d.gorm.Save(&Secret{Key: key, Value: value}).Error
}

func (d *DB) GetSecret(key string) (string, error) {
	var s Secret
	err := d.gorm.First(&s, "key = ?", key).Error
	return s.Value, err
}

func (d *DB) CountAdminUsers() (int64, error) {
	var n int64
	err := d.gorm.Model(&AdminUser{}).Count(&n).Error
	return n, err
}

func (d *DB) CreateAdminUser(u AdminUser) error {
	return d.gorm.Create(&u).Error
}

func (d *DB) UpdateAdminUser(u AdminUser) error {
	return d.gorm.Save(&u).Error
}

func (d *DB) GetAdminUserByUsername(username string) (AdminUser, error) {
	var u AdminUser
	err := d.gorm.First(&u, "username = ?", username).Error
	return u, err
}

func (d *DB) GetAdminUserBySessionToken(token string) (AdminUser, error) {
	var u AdminUser
	err := d.gorm.First(&u, "session_token = ?", token).Error
	return u, err
}

func MustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
