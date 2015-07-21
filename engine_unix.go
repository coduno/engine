// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package main

import "io/ioutil"

func dockerize(path string) (result string, err error) {
	return path, nil
}

func volumeDir() (dir string, err error) {
	return ioutil.TempDir("", "coduno-volume")
}
