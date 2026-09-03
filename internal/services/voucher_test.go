package services

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
)

func testAuth(t *testing.T) *AuthService {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return NewAuthService(config.Config{
		AdminUsername: "admin",
		AdminPassword: "secret",
	}, database)
}

func TestVoucherRegisterAndRedeem(t *testing.T) {
	s := testAuth(t)
	if err := s.EnsureBootstrapAdmin(); err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateVouchers(VoucherCreateInput{Count: 2, Quota: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Codes) != 2 {
		t.Fatalf("codes = %d, want 2", len(created.Codes))
	}

	reg, err := s.Register(RegisterInput{Username: "alice", Password: "pw", Code: created.Codes[0]})
	if err != nil {
		t.Fatal(err)
	}
	if reg.IsAdmin {
		t.Fatal("registered user should not be admin")
	}
	if reg.Quota["decrypt"]["limit"] != 8 {
		t.Fatalf("decrypt limit = %v, want 8", reg.Quota["decrypt"]["limit"])
	}

	if _, err := s.Register(RegisterInput{Username: "bob", Password: "pw", Code: created.Codes[0]}); !errors.Is(err, db.ErrVoucherUsed) {
		t.Fatalf("reuse code: %v, want %v", err, db.ErrVoucherUsed)
	}

	alice, err := s.db.GetUserByUsername("alice")
	if err != nil {
		t.Fatal(err)
	}
	u, added, err := s.RedeemVoucher(alice.ID, created.Codes[1])
	if err != nil {
		t.Fatal(err)
	}
	if added != 8 {
		t.Fatalf("added = %d, want 8", added)
	}
	if u.DecryptLimit != 16 || u.DownloadLimit != 16 {
		t.Fatalf("after redeem limits = %d/%d, want 16/16", u.DecryptLimit, u.DownloadLimit)
	}

	if _, _, err := s.RedeemVoucher(alice.ID, created.Codes[1]); !errors.Is(err, db.ErrVoucherUsed) {
		t.Fatalf("second redeem: %v, want %v", err, db.ErrVoucherUsed)
	}

	admin, err := s.db.GetUserByUsername("admin")
	if err != nil {
		t.Fatal(err)
	}
	more, err := s.CreateVouchers(VoucherCreateInput{Count: 1, Quota: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.RedeemVoucher(admin.ID, more.Codes[0]); err == nil {
		t.Fatal("admin should not redeem quota vouchers")
	}
	list, err := s.ListVouchers(VoucherListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	unused := 0
	for _, v := range list.Items {
		if !v.Redeemed {
			unused++
		}
	}
	if unused != 1 {
		t.Fatalf("unused vouchers = %d, want 1 (admin redeem must not consume)", unused)
	}

	alice2, err := s.UpdateUser(alice.ID, UserUpdateInput{Limit: intPtr(-1)})
	if err != nil {
		t.Fatal(err)
	}
	if alice2.DecryptLimit != -1 {
		t.Fatalf("limit = %d, want -1", alice2.DecryptLimit)
	}
	if _, _, err := s.RedeemVoucher(alice.ID, more.Codes[0]); err == nil {
		t.Fatal("unlimited user should not redeem quota vouchers")
	}
	unused2, err := s.ListVouchers(VoucherListQuery{Status: "unused"})
	if err != nil {
		t.Fatal(err)
	}
	if unused2.Total != 1 {
		t.Fatalf("unused after unlimited redeem = %d, want 1", unused2.Total)
	}

	usedList, err := s.ListVouchers(VoucherListQuery{Status: "used"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteVoucher(usedList.Items[0].ID); !errors.Is(err, db.ErrVoucherUsed) {
		t.Fatalf("delete used: %v, want %v", err, db.ErrVoucherUsed)
	}
	if err := s.DeleteVoucher(unused2.Items[0].ID); err != nil {
		t.Fatal(err)
	}
}

func intPtr(n int) *int { return &n }

func TestVoucherListFilterAndPage(t *testing.T) {
	s := testAuth(t)
	if err := s.EnsureBootstrapAdmin(); err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateVouchers(VoucherCreateInput{Count: 3, Quota: 4})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Register(RegisterInput{Username: "alice", Password: "pw", Code: created.Codes[0]}); err != nil {
		t.Fatal(err)
	}

	unused, err := s.ListVouchers(VoucherListQuery{Status: "unused"})
	if err != nil {
		t.Fatal(err)
	}
	if unused.Total != 2 || len(unused.Items) != 2 {
		t.Fatalf("unused total=%d items=%d, want 2", unused.Total, len(unused.Items))
	}

	used, err := s.ListVouchers(VoucherListQuery{Status: "used"})
	if err != nil {
		t.Fatal(err)
	}
	if used.Total != 1 || used.Items[0].RedeemedUsername != "alice" {
		t.Fatalf("used = %+v", used)
	}

	alice, err := s.db.GetUserByUsername("alice")
	if err != nil {
		t.Fatal(err)
	}
	byUser, err := s.ListVouchers(VoucherListQuery{Status: "user", UserID: alice.ID})
	if err != nil {
		t.Fatal(err)
	}
	if byUser.Total != 1 || byUser.Items[0].RedeemedUsername != "alice" {
		t.Fatalf("user filter = %+v", byUser)
	}
	if len(byUser.Users) != 1 || byUser.Users[0].Username != "alice" {
		t.Fatalf("redeemers = %+v", byUser.Users)
	}

	emptyUser, err := s.ListVouchers(VoucherListQuery{Status: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if emptyUser.Total != 0 || len(emptyUser.Items) != 0 {
		t.Fatalf("empty user filter = %+v", emptyUser)
	}
	if len(emptyUser.Users) != 1 || emptyUser.Users[0].ID != alice.ID {
		t.Fatalf("empty user redeemers = %+v", emptyUser.Users)
	}

	page, err := s.ListVouchers(VoucherListQuery{Status: "unused", Offset: 0, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Limit != 1 {
		t.Fatalf("page = %+v", page)
	}
	page2, err := s.ListVouchers(VoucherListQuery{Status: "unused", Offset: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 1 || page2.Items[0].ID == page.Items[0].ID {
		t.Fatalf("page2 overlap: %+v vs %+v", page2.Items, page.Items)
	}
}

func TestRegisterDuplicateUsernameDoesNotConsumeVoucher(t *testing.T) {
	s := testAuth(t)
	if err := s.EnsureBootstrapAdmin(); err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateVouchers(VoucherCreateInput{Count: 2, Quota: 4})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Register(RegisterInput{Username: "alice", Password: "pw", Code: created.Codes[0]}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Register(RegisterInput{Username: "alice", Password: "pw", Code: created.Codes[1]}); err == nil {
		t.Fatal("duplicate username should fail")
	}
	unused, err := s.ListVouchers(VoucherListQuery{Status: "unused"})
	if err != nil {
		t.Fatal(err)
	}
	if unused.Total != 1 || unused.Items[0].Code != created.Codes[1] {
		t.Fatalf("second voucher should remain unused: %+v", unused)
	}
}
