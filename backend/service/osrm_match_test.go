package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"locator/models"
)

func TestMatchLocationsToRoads_singlePoint(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	out, err := MatchLocationsToRoads(client, "http://unused", []models.Location{
		{Latitude: 53.9, Longitude: 27.57},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0][0] != 53.9 || out[0][1] != 27.57 {
		t.Fatalf("unexpected %v", out)
	}
}

func TestMatchLocationsToRoads_httpMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method %s", r.Method)
		}
		resp := map[string]interface{}{
			"code": "Ok",
			"matchings": []interface{}{
				map[string]interface{}{
					"geometry": map[string]interface{}{
						"type":        "LineString",
						"coordinates": [][]float64{{27.5, 53.9}, {27.51, 53.91}},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	locs := []models.Location{
		{Latitude: 53.9, Longitude: 27.5},
		{Latitude: 53.91, Longitude: 27.51},
	}
	out, err := MatchLocationsToRoads(http.DefaultClient, srv.URL, locs)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 2 {
		t.Fatalf("expected at least 2 coords, got %v", out)
	}
	// Leaflet order lat, lng
	if out[0][0] != 53.9 || out[0][1] != 27.5 {
		t.Fatalf("first point lat/lng: %v", out[0])
	}
}
