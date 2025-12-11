package aria2_downloader

import (
	"path/filepath"

	"github.com/siku2/arigo"
)

func friendly(status arigo.Status) string {
	if len(status.Files) == 0 {
		return "unknow"
	}

	var maxFile arigo.File
	var maxSize uint

	statusFiles := status.Files
	for i := range statusFiles {
		length := statusFiles[i].Length
		if length > maxSize {
			maxSize = length
			maxFile = statusFiles[i]
		}
	}

	name := filepath.Base(maxFile.Path)
	if name == "." {
		return "【GID(" + status.GID + ")】"
	}
	return "【GID(" + status.GID + "): " + name + "】"
}
