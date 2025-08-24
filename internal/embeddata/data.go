package embeddata

import (
	"embed"
	"io/fs"
)

//go:embed about.md sounds.zip
var embeddedFS embed.FS

// FS returns the embedded filesystem with access to about.md amd sounds.zip.
func FS() fs.FS {
	return embeddedFS
}

// ReadConfig returns the contents of about.md.
func ReadAboutMD() ([]byte, error) {
	return embeddedFS.ReadFile("about.md")
}

// ReadSoundsZip returns the contents of embedded sounds.zip.
func ReadSoundsZip() ([]byte, error) {
	return embeddedFS.ReadFile("sounds.zip")
}
