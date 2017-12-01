package confreader

import (
	"os/user"
	"path/filepath"
)

func cleanFilename(filename string) string {
	usr, _ := user.Current()
	if filename[:2] == "~/" {
		filename = filepath.Join(usr.HomeDir, filename[2:])
	}
	return filename
}
