package main

import "testing"

func TestDockerize(*testing.T) {
	const expected = "/c/users/Foo"
	var dockerized string
	dockerized = dockerize(`C:\users\Foo`)
	if dockerized != expected {
		t.Errorf("Expected '%s', got '%s'", expected, dockerized)
	}
}
