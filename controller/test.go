package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"

	"io/ioutil"

	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"
)

// LogData holds all data that is stored for a single run of coduno
type LogData struct {
	Challenge string
	User      string
	Commit    string
	Status    string
	StartTime time.Time
	EndTime   time.Time
	InLog     string
	OutLog    string
	ExtraLog  string
}

var signal = make(chan int)
var secret []byte
var config *jwt.Config
var ctx context.Context

func init() {
	err := error(nil)
	secret, err = ioutil.ReadFile("../config/secret.json")
	if err != nil {
		panic(err)
	}
	config, err = google.JWTConfigFromJSON(secret, datastore.ScopeDatastore, datastore.ScopeUserEmail)
	if err != nil {
		panic(err)
	}

	ctx = cloud.NewContext("coduno", config.Client(oauth2.NoContext))
}

// LogBuildStart sends info to the datastore, informing that a new build
// started
func LogBuildStart(repo string, commit string, user string) {
	key := datastore.NewIncompleteKey(ctx, "testrun", nil)
	_, err := datastore.Put(ctx, key, &LogData{
		Commit:    commit,
		Challenge: repo,
		User:      user,
	})
	if err != nil {
		log.Panic(err)
	}
}

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
	LogBuildStart("test", commit, tmpdir)
	/*
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
		log.Print(commit)*/
}
