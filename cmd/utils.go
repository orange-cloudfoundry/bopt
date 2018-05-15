package cmd

import (
	"path/filepath"
	"strings"
	"os/user"
	"fmt"
)

func expandPath(path string) (string, error) {

	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("Getting current user home dir: %s", err.Error())
		}
		path = filepath.Join(usr.HomeDir, path[1:])
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("Getting absolute path: %s", err.Error())
	}

	return path, nil
}
