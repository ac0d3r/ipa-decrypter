package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrVoucherInvalid = errors.New("invalid voucher")
	ErrVoucherUsed    = errors.New("voucher already used")
)

func (d *DB) CreateVouchers(items []Voucher) error {
	if len(items) == 0 {
		return nil
	}
	return d.gorm.Create(&items).Error
}

type VoucherListFilter struct {
	Unused  bool
	Used    bool
	UserIDs []string
	Offset  int
	Limit   int
}

func (d *DB) ListVouchersPage(f VoucherListFilter) ([]Voucher, int64, error) {
	q := d.gorm.Model(&Voucher{})
	if f.Unused {
		q = q.Where("redeemed_at IS NULL")
	} else if f.Used {
		q = q.Where("redeemed_at IS NOT NULL")
	}
	if len(f.UserIDs) > 0 {
		q = q.Where("user_id IN ?", f.UserIDs)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	var out []Voucher
	err := q.Order("created_at DESC").Offset(f.Offset).Limit(f.Limit).Find(&out).Error
	if out == nil {
		out = []Voucher{}
	}
	return out, total, err
}

func (d *DB) ListVoucherRedeemers() ([]User, error) {
	sub := d.gorm.Model(&Voucher{}).
		Select("user_id").
		Where("redeemed_at IS NOT NULL AND user_id != ''")
	var users []User
	err := d.gorm.Where("id IN (?)", sub).Order("username ASC").Find(&users).Error
	if users == nil {
		users = []User{}
	}
	return users, err
}

func (d *DB) DeleteUnusedVoucher(id string) error {
	if id == "" {
		return gorm.ErrRecordNotFound
	}
	res := d.gorm.Where("id = ? AND redeemed_at IS NULL", id).Delete(&Voucher{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 1 {
		return nil
	}
	var v Voucher
	if err := d.gorm.First(&v, "id = ?", id).Error; err != nil {
		return err
	}
	return ErrVoucherUsed
}

func (d *DB) RedeemVoucherForNewUser(code string, u *User) error {
	return d.gorm.Transaction(func(tx *gorm.DB) error {
		v, err := takeUnusedVoucher(tx, code)
		if err != nil {
			return err
		}
		u.IsAdmin = false
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		return markVoucherRedeemed(tx, v, u.ID)
	})
}

func takeUnusedVoucher(tx *gorm.DB, code string) (*Voucher, error) {
	var v Voucher
	err := tx.Where("code = ?", code).First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVoucherInvalid
		}
		return nil, err
	}
	if v.Used() {
		return nil, ErrVoucherUsed
	}
	return &v, nil
}

func markVoucherRedeemed(tx *gorm.DB, v *Voucher, userID string) error {
	now := time.Now()
	res := tx.Model(&Voucher{}).
		Where("id = ? AND redeemed_at IS NULL", v.ID).
		Updates(map[string]any{
			"user_id":     userID,
			"redeemed_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrVoucherUsed
	}
	v.UserID = userID
	v.RedeemedAt = &now
	return nil
}
