package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"locator/internal/testutil"
	"locator/middleware"
	"locator/models"
	"locator/service"

	"github.com/gin-gonic/gin"
)

type fakeUserRepo struct {
	users map[int]models.User
}

func (f *fakeUserRepo) Create(user *models.User) error {
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
		return nil, http.ErrNoCookie
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

func newTestUserService(t *testing.T, plain string, isAdmin bool) *service.UserService {
	t.Helper()
	u, err := testutil.UserWithAPIKey(1, "test", plain, isAdmin)
	if err != nil {
		t.Fatal(err)
	}
	svc := &service.UserService{DAO: &fakeUserRepo{users: map[int]models.User{1: u}}}
	return svc
}

func TestBasicAuthMiddleware_missingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestUserService(t, "valid-key-12345678", true)
	r := gin.New()
	r.GET("/me", middleware.BasicAuthMiddleware(svc), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestBasicAuthMiddleware_invalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestUserService(t, "valid-key-12345678", true)
	r := gin.New()
	r.GET("/me", middleware.BasicAuthMiddleware(svc), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-API-Key", "wrong-key-00000000")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestBasicAuthMiddleware_success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const key = "valid-key-12345678"
	svc := newTestUserService(t, key, false)
	r := gin.New()
	r.GET("/me", middleware.BasicAuthMiddleware(svc), func(c *gin.Context) {
		u, _ := c.Get("user")
		c.JSON(200, u)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-API-Key", key)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got models.User
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 || got.IsAdmin {
		t.Fatalf("got %+v", got)
	}
}

func TestAPIKeyAuthMiddleware_nonAdminForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const key = "device-key-abcdefgh"
	svc := newTestUserService(t, key, false)
	r := gin.New()
	r.GET("/admin", middleware.APIKeyAuthMiddleware(svc), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("X-API-Key", key)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAPIKeyAuthMiddleware_adminOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const key = "admin-key-abcdefgh"
	svc := newTestUserService(t, key, true)
	r := gin.New()
	r.GET("/admin", middleware.APIKeyAuthMiddleware(svc), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("X-API-Key", key)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
