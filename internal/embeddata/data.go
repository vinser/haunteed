package embeddata

import (
	"embed"
	"io/fs"
)

//go:embed config.yml sounds.zip
var embeddedFS embed.FS

// FS returns the embedded filesystem with access to config.yml amd sounds.zip.
func FS() fs.FS {
	return embeddedFS
}

// ReadConfig returns the contents of embedded config.yml.
func ReadConfig() ([]byte, error) {
	return embeddedFS.ReadFile("config.yml")
}

// ReadSoundsZip returns the contents of embedded sounds.zip.
func ReadSoundsZip() ([]byte, error) {
	return embeddedFS.ReadFile("sounds.zip")
}
