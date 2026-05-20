package service

import (
	"fmt"
	"os"

	"github.com/shogo82148/androidbinary/apk"
)

// APKMeta — versionCode/versionName из AndroidManifest внутри APK.
type APKMeta struct {
	PackageName string
	VersionName string
	VersionCode uint32
}

// ReadAPKMeta читает versionCode/versionName из APK (не из имени файла).
func ReadAPKMeta(path string) (*APKMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open apk: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	zip, err := apk.OpenZipReader(f, stat.Size())
	if err != nil {
		return nil, fmt.Errorf("open apk zip: %w", err)
	}
	defer zip.Close()

	manifest := zip.Manifest()

	code, err := manifest.VersionCode.Int32()
	if err != nil {
		return nil, fmt.Errorf("versionCode: %w", err)
	}
	name, err := manifest.VersionName.String()
	if err != nil {
		return nil, fmt.Errorf("versionName: %w", err)
	}
	pkg, err := manifest.Package.String()
	if err != nil {
		pkg = ""
	}

	return &APKMeta{
		PackageName: pkg,
		VersionName: name,
		VersionCode: uint32(code),
	}, nil
}
