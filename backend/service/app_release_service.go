package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteReleaseManifest обновляет manifest.json по метаданным из APK.
func WriteReleaseManifest(apkPath, manifestPath, baseURL, changelog string) (map[string]interface{}, error) {
	meta, err := ReadAPKMeta(apkPath)
	if err != nil {
		return nil, err
	}

	sum, err := sha256File(apkPath)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(apkPath)
	if changelog == "" {
		changelog = fmt.Sprintf("Release %s (build %d)", meta.VersionName, meta.VersionCode)
	}

	m := map[string]interface{}{
		"version_name": meta.VersionName,
		"version_code": meta.VersionCode,
		"package_name": meta.PackageName,
		"filename":     filename,
		"sha256":       sum,
		"force":        false,
		"changelog":    changelog,
		"url":          baseURL + "/static/releases/" + filename,
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	return m, nil
}

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
