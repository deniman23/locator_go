package main

import (
	"encoding/json"
	"fmt"
	"os"

	"locator/service"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: apkinfo <file.apk>")
		os.Exit(1)
	}
	meta, err := service.ReadAPKMeta(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(meta)
}
