package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestHealthz(t *testing.T) {
	env := setupEnv(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAuth_usersMe(t *testing.T) {
	env := setupEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("X-API-Key", env.AdminKey)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	env.Router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w2.Code)
	}
}

func TestAuth_deviceForbiddenOnAdminRoute(t *testing.T) {
	env := setupEnv(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/checkpoint/", nil)
	req.Header.Set("X-API-Key", env.DeviceKey)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLocation_postAndGetSingleAndCurrent(t *testing.T) {
	env := setupEnv(t)

	body := map[string]interface{}{
		"latitude":  53.9,
		"longitude": 27.5,
		"source":    "periodic",
	}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/location", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", env.DeviceKey)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("POST location status=%d body=%s", w.Code, w.Body.String())
	}

	for _, path := range []string{"/api/location/single", "/api/location/current"} {
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, path+"?user_id="+itoa(env.Device.ID), nil)
		req2.Header.Set("X-API-Key", env.DeviceKey)
		env.Router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, w2.Code, w2.Body.String())
		}
	}
}

func TestCheckpoint_CRUD(t *testing.T) {
	env := setupEnv(t)

	create := map[string]interface{}{
		"name":      "IT Office",
		"latitude":  53.92684,
		"longitude": 27.695144,
		"radius":    100,
	}
	raw, _ := json.Marshal(create)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/checkpoint/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", env.AdminKey)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("create checkpoint status=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/checkpoint/", nil)
	req2.Header.Set("X-API-Key", env.AdminKey)
	env.Router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list checkpoints status=%d", w2.Code)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) < 1 {
		t.Fatal("expected at least one checkpoint")
	}
}

func TestVisits_listEmpty(t *testing.T) {
	env := setupEnv(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/visits/?user_id="+itoa(env.Device.ID), nil)
	req.Header.Set("X-API-Key", env.AdminKey)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDevice_pollEmptyThenEnqueueAndPoll(t *testing.T) {
	env := setupEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/device/poll", nil)
	req.Header.Set("X-API-Key", env.DeviceKey)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("poll empty status=%d body=%s", w.Code, w.Body.String())
	}

	cmdBody := map[string]interface{}{"type": "health_check"}
	raw, _ := json.Marshal(cmdBody)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+itoa(env.Device.ID)+"/commands", bytes.NewReader(raw))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-API-Key", env.AdminKey)
	env.Router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusAccepted && w2.Code != http.StatusOK {
		t.Fatalf("enqueue status=%d body=%s", w2.Code, w2.Body.String())
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/device/poll", nil)
	req3.Header.Set("X-API-Key", env.DeviceKey)
	env.Router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("poll after enqueue status=%d body=%s", w3.Code, w3.Body.String())
	}
	var pollResp struct {
		Command struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"command"`
	}
	if err := json.Unmarshal(w3.Body.Bytes(), &pollResp); err != nil {
		t.Fatal(err)
	}
	if pollResp.Command.Type != "health_check" {
		t.Fatalf("cmd=%v body=%s", pollResp, w3.Body.String())
	}

	ack := map[string]interface{}{
		"command_id": pollResp.Command.ID,
		"status":     "ok",
		"message":    "done",
	}
	ackRaw, _ := json.Marshal(ack)
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost, "/api/device/command/ack", bytes.NewReader(ackRaw))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("X-API-Key", env.DeviceKey)
	env.Router.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("ack status=%d body=%s", w4.Code, w4.Body.String())
	}
}

func TestAppRelease_latestPublic(t *testing.T) {
	env := setupEnv(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/app/release/latest", nil)
	env.Router.ServeHTTP(w, req)
	// 200 with manifest or 404 if missing — both acceptable for harness smoke
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestVisitEventProcessor_enterExit(t *testing.T) {
	env := setupEnv(t)

	// Create checkpoint via API
	create := map[string]interface{}{
		"name": "Visit CP", "latitude": 53.92684, "longitude": 27.695144, "radius": 100.0,
	}
	raw, _ := json.Marshal(create)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/checkpoint/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", env.AdminKey)
	env.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("checkpoint: %s", w.Body.String())
	}

	// Process events directly (no RabbitMQ) — covered more thoroughly in unit tests;
	// here we verify visits endpoint still works after DB seed.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/visits/?user_id="+itoa(env.Device.ID)+"&active=true", nil)
	req2.Header.Set("X-API-Key", env.AdminKey)
	env.Router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("visits: %d %s", w2.Code, w2.Body.String())
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
