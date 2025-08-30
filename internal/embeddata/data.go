package embeddata

import (
	"embed"
	"io/fs"
)

//go:embed about.md bosskey.json motd.json sounds.zip
var embeddedFS embed.FS

// FS returns the embedded filesystem with access to files in this folder.
func FS() fs.FS {
	return embeddedFS
}

// ReadConfig returns the contents of about.md.
func ReadAboutMD() ([]byte, error) {
	return embeddedFS.ReadFile("about.md")
}

// ReadBoss returns the contents of bosskey.json.
func ReadBoss() ([]byte, error) {
	return embeddedFS.ReadFile("bosskey.json")
}

// ReadMOTD returns the contents of motd.json.
func ReadMOTD() ([]byte, error) {
	return embeddedFS.ReadFile("motd.json")
}

// ReadSoundsZip returns the contents of embedded sounds.zip.
func ReadSoundsZip() ([]byte, error) {
	return embeddedFS.ReadFile("sounds.zip")
}
