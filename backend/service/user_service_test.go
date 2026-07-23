package service

import (
	"errors"
	"testing"

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
