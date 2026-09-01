package db

import "errors"

const DefaultQuota = 5

var ErrQuotaExceeded = errors.New("quota exceeded")

func (u User) quotaLimit(qtype string) int {
	if u.IsAdmin {
		return -1
	}
	n := u.DecryptLimit
	if qtype == "download" {
		n = u.DownloadLimit
	}
	if n < 0 {
		return -1
	}
	if n == 0 {
		return DefaultQuota
	}
	return n
}

func (u User) quotaUsed(qtype string) int {
	if qtype == "download" {
		return u.Downloads
	}
	return u.Decrypts
}

func (u User) CheckQuota(qtype string) error {
	limit := u.quotaLimit(qtype)
	if limit >= 0 && u.quotaUsed(qtype) >= limit {
		return ErrQuotaExceeded
	}
	return nil
}

func (u *User) ConsumeQuota(qtype string) error {
	if err := u.CheckQuota(qtype); err != nil {
		return err
	}
	if qtype == "download" {
		u.Downloads++
	} else {
		u.Decrypts++
	}
	return nil
}

func (u User) Quota() map[string]map[string]int {
	return map[string]map[string]int{
		"decrypt":  {"used": u.quotaUsed("decrypt"), "limit": u.quotaLimit("decrypt")},
		"download": {"used": u.quotaUsed("download"), "limit": u.quotaLimit("download")},
	}
}
