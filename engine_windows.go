package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"
)

var currentUser *user.User

// dockerize takes a Windows path (with volume and
// backslashes) and translates it into a Unix-like
// path that can be passed to the Docker CLI for
// volume mounting.
func dockerize(path string) (result string, err error) {
	if path[0] < 65 || path[0] > 122 || (path[0] > 90 && path[0] < 97) {
		err = fmt.Errorf("'%s' is not a valid disk designator", path[:1])
		return
	}
	if path[1] != ':' {
		err = fmt.Errorf("missing colon after disk designator")
		return
	}
	if path[2] != '\\' {
		err = fmt.Errorf("cannot deal with relative paths (backslash expected)")
		return
	}

	return "/" + strings.ToLower(path[:1]) + "/" + strings.Replace(path[3:], `\`, `/`, -1), nil
}

func volumeDir() (dir string, err error) {
	dir, err = ioutil.TempDir(path.Join(currentUser.HomeDir, "tmp"), "coduno-volume")
	if err != nil {
		return
	}
	return
}

func init() {
	var err error
	currentUser, err = user.Current()
	if err != nil {
		panic(err)
	}

	const expectedHomePrefix = "C:\\Users\\"

	if !strings.HasPrefix(currentUser.HomeDir, expectedHomePrefix) {
		fmt.Fprintf(os.Stderr, "Your home path does not begin with '%s' which is suspicious, because you appear to be running Windows and that is the only folder shared with boot2docker.", expectedHomePrefix)
	}
}
