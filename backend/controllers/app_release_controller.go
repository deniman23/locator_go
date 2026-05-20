package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"locator/service"

	"github.com/gin-gonic/gin"
)

// AppReleaseController — публикация APK для OTA (app_update).
type AppReleaseController struct {
	ManifestPath string
	ReleasesDir  string
	BaseURL      string
}

func NewAppReleaseController(manifestPath, releasesDir, baseURL string) *AppReleaseController {
	return &AppReleaseController{
		ManifestPath: manifestPath,
		ReleasesDir:  releasesDir,
		BaseURL:      baseURL,
	}
}

func (rc *AppReleaseController) loadManifest() (map[string]interface{}, error) {
	data, err := os.ReadFile(rc.ManifestPath)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	filename, _ := m["filename"].(string)
	if filename != "" {
		m["url"] = rc.BaseURL + "/static/releases/" + filename
	}
	return m, nil
}

// GetLatestRelease — GET /api/app/release/latest (без авторизации).
func (rc *AppReleaseController) GetLatestRelease(ctx *gin.Context) {
	m, err := rc.loadManifest()
	if err != nil {
		if os.IsNotExist(err) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Релиз не опубликован"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка чтения манифеста"})
		return
	}
	ctx.JSON(http.StatusOK, m)
}

// SyncManifestFromReleaseAPK пересчитывает sha256 и версию из APK в manifest.json.
func (rc *AppReleaseController) SyncManifestFromReleaseAPK() (map[string]interface{}, error) {
	m, err := rc.loadManifest()
	if err != nil {
		return nil, err
	}
	filename, _ := m["filename"].(string)
	if filename == "" {
		return nil, fmt.Errorf("manifest: filename пуст")
	}
	apkPath := filepath.Join(rc.ReleasesDir, filename)
	return service.WriteReleaseManifest(apkPath, rc.ManifestPath, rc.BaseURL, "")
}

// ManifestForAppUpdate возвращает payload для команды app_update.
func (rc *AppReleaseController) ManifestForAppUpdate() (map[string]interface{}, error) {
	m, err := rc.loadManifest()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"url":          m["url"],
		"version":      m["version_name"],
		"version_code": m["version_code"],
		"sha256":       m["sha256"],
		"force":        m["force"],
		"changelog":    m["changelog"],
		"filename":     m["filename"],
	}, nil
}

// PostSyncReleaseManifest — POST /api/admin/releases/sync-manifest
// Перечитывает versionCode/versionName из APK в releases и обновляет manifest.json.
func (rc *AppReleaseController) PostSyncReleaseManifest(ctx *gin.Context) {
	currentUser, ok := getCurrentUserFromContext(ctx)
	if !ok {
		return
	}
	if !currentUser.IsAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Требуются права администратора"})
		return
	}

	var body struct {
		Filename  string `json:"filename"`
		Changelog string `json:"changelog"`
	}
	_ = ctx.ShouldBindJSON(&body)

	apkPath := filepath.Join(rc.ReleasesDir, body.Filename)
	if body.Filename == "" {
		m, err := rc.loadManifest()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Укажите filename или опубликуйте manifest"})
			return
		}
		fn, _ := m["filename"].(string)
		apkPath = filepath.Join(rc.ReleasesDir, fn)
	}

	if _, err := os.Stat(apkPath); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "APK не найден", "path": apkPath})
		return
	}

	manifest, err := service.WriteReleaseManifest(apkPath, rc.ManifestPath, rc.BaseURL, body.Changelog)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":  "manifest обновлён из APK",
		"manifest": manifest,
	})
}
