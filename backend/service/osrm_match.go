package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"locator/models"
)

// osrmMatchChunkSize — сколько точек в одном GET к /match/v1/… Публичный router.project-osrm.org
// принимает не больше ~10 координат (иначе TooBig); задайте ROUTING_MATCH_CHUNK_SIZE=10.
func osrmMatchChunkSize() int {
	const def = 80
	s := strings.TrimSpace(os.Getenv("ROUTING_MATCH_CHUNK_SIZE"))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 2 {
		return def
	}
	if n > 100 {
		return 100
	}
	return n
}

// osrmMatchRadiusMeters — опциональный радиус поиска (метры) на каждую точку в match; уменьшает NoMatch
// на «рваном» GPS. Публичный OSRM: попробуйте 15–25 (50 может дать TooBig). Пусто — без radiuses.
func osrmMatchRadiusMeters() (ok bool, meters int) {
	s := strings.TrimSpace(os.Getenv("ROUTING_MATCH_RADIUS"))
	if s == "" {
		return false, 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return false, 0
	}
	if n > 50 {
		n = 50
	}
	return true, n
}

// OSRMMatchResponse минимальная структура ответа /match/v1/... с geometries=geojson.
type osrmMatchResponse struct {
	Matchings []struct {
		Geometry struct {
			Type        string      `json:"type"`
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"matchings"`
	Code string `json:"code"`
}

// MatchLocationsToRoads строит линию по дорогам через OSRM match. baseURL без завершающего слэша
// (например https://router.project-osrm.org). Координаты в ответе: [lat, lng] для Leaflet.
func MatchLocationsToRoads(client *http.Client, baseURL string, locs []models.Location) ([][]float64, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("routing base URL is empty")
	}
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	if len(locs) == 0 {
		return nil, nil
	}
	if len(locs) == 1 {
		return [][]float64{{locs[0].Latitude, locs[0].Longitude}}, nil
	}

	chunkSz := osrmMatchChunkSize()
	var merged [][]float64
	for start := 0; start < len(locs); {
		end := start + chunkSz
		if end > len(locs) {
			end = len(locs)
		}
		chunk := locs[start:end]
		part, err := osrmMatchChunk(client, baseURL, chunk)
		if err != nil {
			return nil, err
		}
		if len(merged) > 0 && len(part) > 0 {
			// убрать дубликат стыка
			last := merged[len(merged)-1]
			first := part[0]
			if last[0] == first[0] && last[1] == first[1] {
				part = part[1:]
			}
		}
		merged = append(merged, part...)
		if end == len(locs) {
			break
		}
		// перекрытие в одну точку для непрерывности трека
		start = end - 1
	}
	return merged, nil
}

func osrmMatchChunk(client *http.Client, baseURL string, chunk []models.Location) ([][]float64, error) {
	var b strings.Builder
	for i, loc := range chunk {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(fmt.Sprintf("%f,%f", loc.Longitude, loc.Latitude))
	}
	u := fmt.Sprintf("%s/match/v1/driving/%s?overview=full&geometries=geojson&steps=false",
		baseURL, b.String())
	if useR, r := osrmMatchRadiusMeters(); useR {
		var rad strings.Builder
		for i := 0; i < len(chunk); i++ {
			if i > 0 {
				rad.WriteByte(';')
			}
			rad.WriteString(strconv.Itoa(r))
		}
		u += "&radiuses=" + rad.String()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSRM HTTP %d: %s", resp.StatusCode, string(body))
	}
	var parsed osrmMatchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("OSRM JSON: %w", err)
	}
	if parsed.Code != "Ok" && parsed.Code != "" {
		// некоторые инстансы не возвращают code=Ok
		if len(parsed.Matchings) == 0 {
			return nil, fmt.Errorf("OSRM: code=%s, no matchings", parsed.Code)
		}
	}
	if len(parsed.Matchings) == 0 {
		return nil, fmt.Errorf("OSRM: пустой matchings")
	}
	coords := parsed.Matchings[0].Geometry.Coordinates
	out := make([][]float64, 0, len(coords))
	for _, c := range coords {
		if len(c) < 2 {
			continue
		}
		// OSRM GeoJSON: lng, lat → Leaflet: lat, lng
		out = append(out, []float64{c[1], c[0]})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("OSRM: пустая геометрия")
	}
	return out, nil
}
