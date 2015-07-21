package main

import "testing"

func TestDockerize(*testing.T) {
	const expected = "/c/users/Foo"

	var dockerized string
	dockerized, err := dockerize(`C:\users\Foo`)

	if err != nil {
		t.Error("Unexpected error: ", err)
	}

	if dockerized != expected {
		t.Error("Expected '", expected, "', got '", dockerized, "'")
	}
}
