package main

import (
	"bytes"
	"io"
	"log"
	"os/exec"
)

var signal = make(chan int)

func send_sig() {
	signal <- 1
}

func pipe_output(out io.ReadCloser, dest io.WriteCloser, logBuf *bytes.Buffer) {
	tempBuf := make([]byte, 1024)
	writeErr := error(nil)
	r, readErr := int(0), error(nil)

	defer out.Close()
	defer dest.Close()
	defer send_sig()

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
	cmd := exec.Command("timeout", "0.02", "./test.sh")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	var b bytes.Buffer
	cmd.Start()
	go pipe_output(stdout, stdin, &b)

	<-signal
	log.Print(b.String())
}
