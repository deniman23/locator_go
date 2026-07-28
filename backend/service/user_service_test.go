package service

import (
	"errors"
	"testing"
	"time"

	"locator/internal/testutil"
	"locator/models"
)

type fakeUserRepo struct {
	users  map[int]models.User
	nextID int
}

func newFakeUserRepo(users ...models.User) *fakeUserRepo {
	f := &fakeUserRepo{users: make(map[int]models.User), nextID: 1}
	for _, u := range users {
		if u.ID == 0 {
			u.ID = f.nextID
			f.nextID++
		}
		if u.ID >= f.nextID {
			f.nextID = u.ID + 1
		}
		f.users[u.ID] = u
	}
	return f
}

func (f *fakeUserRepo) Create(user *models.User) error {
	if user.ID == 0 {
		user.ID = f.nextID
		f.nextID++
	}
	f.users[user.ID] = *user
	return nil
}

func (f *fakeUserRepo) Update(user *models.User) error {
	f.users[user.ID] = *user
	return nil
}

func (f *fakeUserRepo) GetByID(id int) (*models.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := u
	return &cp, nil
}

func (f *fakeUserRepo) GetAll() ([]models.User, error) {
	out := make([]models.User, 0, len(f.users))
	for _, u := range f.users {
		out = append(out, u)
	}
	return out, nil
}

func TestAuthenticateUser_success(t *testing.T) {
	const plain = "test-api-key-admin-01"
	admin, err := testutil.UserWithAPIKey(1, "admin", plain, true)
	if err != nil {
		t.Fatal(err)
	}
	svc := &UserService{DAO: newFakeUserRepo(admin)}

	got, err := svc.AuthenticateUser(plain)
	if err != nil {
		t.Fatalf("AuthenticateUser: %v", err)
	}
	if got.ID != 1 || !got.IsAdmin {
		t.Fatalf("got %+v", got)
	}
}

func TestAuthenticateUser_wrongKey(t *testing.T) {
	admin, err := testutil.UserWithAPIKey(1, "admin", "correct-key-xxxxxxxx", true)
	if err != nil {
		t.Fatal(err)
	}
	svc := &UserService{DAO: newFakeUserRepo(admin)}

	_, err = svc.AuthenticateUser("wrong-key-yyyyyyyy")
	if err == nil {
		t.Fatal("expected error for wrong key")
	}
}

func TestAuthenticateUser_emptyKey(t *testing.T) {
	svc := &UserService{DAO: newFakeUserRepo()}
	_, err := svc.AuthenticateUser("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestAuthenticateUser_nonAdminFlagPreserved(t *testing.T) {
	const plain = "device-user-key-aaaa"
	user, err := testutil.UserWithAPIKey(2, "device", plain, false)
	if err != nil {
		t.Fatal(err)
	}
	svc := &UserService{DAO: newFakeUserRepo(user)}

	got, err := svc.AuthenticateUser(plain)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsAdmin {
		t.Fatal("expected non-admin user")
	}
}

func TestAuthenticateUser_cacheHitThenRotatedKeyRejected(t *testing.T) {
	const plain = "cache-key-aaaaaaaa"
	user, err := testutil.UserWithAPIKey(3, "device", plain, false)
	if err != nil {
		t.Fatal(err)
	}
	repo := newFakeUserRepo(user)
	svc := &UserService{DAO: repo}

	if _, err := svc.AuthenticateUser(plain); err != nil {
		t.Fatalf("warm cache: %v", err)
	}
	if _, err := svc.AuthenticateUser(plain); err != nil {
		t.Fatalf("cache hit: %v", err)
	}

	rotated, err := testutil.UserWithAPIKey(3, "device", "rotated-key-bbbbbbbb", false)
	if err != nil {
		t.Fatal(err)
	}
	_ = repo.Update(&rotated)

	if _, err := svc.AuthenticateUser(plain); err == nil {
		t.Fatal("expected old key rejected after rotate even if cache warmed")
	}
}

func TestRegenerateUserQR_invalidatesAuthCache(t *testing.T) {
	t.Setenv("BASE_URL", "http://localhost:8080")
	const plain = "regen-key-cccccccc"
	user, err := testutil.UserWithAPIKey(4, "device", plain, false)
	if err != nil {
		t.Fatal(err)
	}
	svc := &UserService{DAO: newFakeUserRepo(user)}
	tmp := t.TempDir()
	t.Chdir(tmp)

	if _, err := svc.AuthenticateUser(plain); err != nil {
		t.Fatalf("warm: %v", err)
	}
	if _, _, err := svc.RegenerateUserQR(4); err != nil {
		t.Fatalf("RegenerateUserQR: %v", err)
	}
	if _, err := svc.AuthenticateUser(plain); err == nil {
		t.Fatal("expected old key rejected after RegenerateUserQR")
	}
}

// TestAuthenticateUser_cacheHit verifies that a second call with the same key
// hits the cache (avoids bcrypt) and still returns the correct user.
func TestAuthenticateUser_cacheHit(t *testing.T) {
	const plain = "cache-hit-test-key-01"
	user, err := testutil.UserWithAPIKey(3, "cacheuser", plain, false)
	if err != nil {
		t.Fatal(err)
	}
	repo := newFakeUserRepo(user)
	svc := &UserService{DAO: repo}

	// First call: cache miss — full bcrypt scan.
	got1, err := svc.AuthenticateUser(plain)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if got1.ID != 3 {
		t.Fatalf("expected ID=3, got %d", got1.ID)
	}

	// Second call: cache hit — must still return the same user.
	got2, err := svc.AuthenticateUser(plain)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got2.ID != got1.ID {
		t.Fatalf("cache hit returned different user: %d vs %d", got2.ID, got1.ID)
	}
}

// TestAuthenticateUser_cacheExpiry verifies that an expired cache entry falls
// back to the full bcrypt path.
func TestAuthenticateUser_cacheExpiry(t *testing.T) {
	const plain = "cache-expiry-test-key"
	user, err := testutil.UserWithAPIKey(4, "expiry", plain, false)
	if err != nil {
		t.Fatal(err)
	}
	svc := &UserService{DAO: newFakeUserRepo(user)}

	// Warm the cache normally.
	if _, err := svc.AuthenticateUser(plain); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	// Manually expire the entry.
	keyHash := sha256KeyHex(plain)
	svc.authCache.Store(keyHash, &authCacheEntry{userID: 4, expiresAt: time.Now().Add(-time.Second)})

	// Call again — should fall through to full auth and refresh cache.
	got, err := svc.AuthenticateUser(plain)
	if err != nil {
		t.Fatalf("after expiry: %v", err)
	}
	if got.ID != 4 {
		t.Fatalf("expected ID=4, got %d", got.ID)
	}
}

// TestAuthenticateUser_deletedUserEvictsCache verifies that when a user is
// deleted from the DB, the next auth attempt evicts the stale cache entry
// and correctly returns an error.
func TestAuthenticateUser_deletedUserEvictsCache(t *testing.T) {
	const plain = "deleted-user-key-0001"
	user, err := testutil.UserWithAPIKey(5, "gone", plain, false)
	if err != nil {
		t.Fatal(err)
	}
	repo := newFakeUserRepo(user)
	svc := &UserService{DAO: repo}

	// Warm cache.
	if _, err := svc.AuthenticateUser(plain); err != nil {
		t.Fatalf("warmup: %v", err)
	}

	// Simulate deletion by removing from fake repo.
	delete(repo.users, 5)

	// Auth must now fail.
	_, err = svc.AuthenticateUser(plain)
	if err == nil {
		t.Fatal("expected error after user deletion, got nil")
	}
}
