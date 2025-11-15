package swagger

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed swagger-ui/*
var content embed.FS

func GetHandler() (http.Handler, error) {
	subFS, err := fs.Sub(content, "swagger-ui")
	if err != nil {
		return nil, err
	}

	return http.FileServer(http.FS(subFS)), nil
}
