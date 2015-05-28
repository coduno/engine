package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
)

var signal = make(chan int)

func sendSig() {
	signal <- 1
}

func pipeOutput(out io.ReadCloser, dest io.WriteCloser, logBuf *bytes.Buffer) {
	tempBuf := make([]byte, 1024)
	writeErr := error(nil)
	r, readErr := int(0), error(nil)

	defer out.Close()
	defer dest.Close()
	defer sendSig()

	for readErr == nil {
		r, readErr = out.Read(tempBuf)
		logBuf.Write(tempBuf[0:r])

		if r != 0 && writeErr == nil {
			_, writeErr := dest.Write(tempBuf[0:r])
			if writeErr != nil {
				log.Print(writeErr)
			}
		}
	}
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Invalid number of arguments. This should never have happend")
	}

	commit := os.Args[1]
	tmpdir := os.Args[2]

	cmd := exec.Command("sudo", "docker", "run", "--rm", "-v", tmpdir+":/app", "coduno/base")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	var b1 bytes.Buffer
	var b2 bytes.Buffer
	cmd.Start()
	go pipeOutput(stdout, stdin, &b1)
	go pipeOutput(stderr, stdin, &b2)

	<-signal
	<-signal
	log.Print(b1.String())
	log.Print(b2.String())
	log.Print(commit)
}
