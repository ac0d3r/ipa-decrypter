package db

import (
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
		&Task{},
		&User{},
		&Voucher{},
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

func (d *DB) CreateUser(u *User) error {
	return d.gorm.Create(u).Error
}

func (d *DB) UpdateUser(u User) error {
	return d.gorm.Save(&u).Error
}

func (d *DB) GetUser(id string) (User, error) {
	var u User
	err := d.gorm.First(&u, "id = ?", id).Error
	return u, err
}

func (d *DB) GetUserByUsername(username string) (User, error) {
	var u User
	err := d.gorm.First(&u, "username = ?", username).Error
	return u, err
}

func (d *DB) GetUserBySessionToken(token string) (User, error) {
	if token == "" {
		return User{}, gorm.ErrRecordNotFound
	}
	var u User
	err := d.gorm.First(&u, "session_token = ?", token).Error
	return u, err
}

func (d *DB) ListUsers() ([]User, error) {
	var out []User
	err := d.gorm.Order("created_at ASC").Find(&out).Error
	return out, err
}

func (d *DB) GetUsersByIDs(ids []string) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []User
	err := d.gorm.Where("id IN ?", ids).Find(&out).Error
	return out, err
}

func (d *DB) DeleteUser(id string) error {
	return d.gorm.Delete(&User{}, "id = ?", id).Error
}

func (d *DB) CountAdmins() (int64, error) {
	var n int64
	err := d.gorm.Model(&User{}).Where("is_admin = ?", true).Count(&n).Error
	return n, err
}
