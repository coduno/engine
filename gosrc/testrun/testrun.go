package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"io/ioutil"

	"google.golang.org/cloud"
	"google.golang.org/cloud/datastore"
)

// BuildData is used to represent general data for a single invokation of
// a coduno build
type BuildData struct {
	Challenge string
	User      string
	Commit    string
	Status    string
	StartTime time.Time
	EndTime   time.Time
}

// LogData is used to represent accumulated log data of a single invokation of
// a coduno build
type LogData struct {
	InLog    string
	OutLog   string
	ExtraLog string
}

var signal = make(chan int)
var ctx context.Context

func init() {
	err := error(nil)
	home := os.Getenv("HOME")
	secret, err := ioutil.ReadFile(home + "/config/secret.json")
	if err != nil {
		log.Panic(err)
	}
	config, err := google.JWTConfigFromJSON(secret, datastore.ScopeDatastore,
		datastore.ScopeUserEmail)
	if err != nil {
		log.Panic(err)
	}

	ctx = cloud.NewContext("coduno", config.Client(oauth2.NoContext))
}

// LogBuildStart sends info to the datastore, informing that a new build
// started
func LogBuildStart(repo string, commit string, user string) (*datastore.Key,
	*BuildData) {
	key := datastore.NewIncompleteKey(ctx, "testrun", nil)
	build := &BuildData{
		Commit:    commit,
		Challenge: repo,
		User:      user,
		StartTime: time.Now(),
		Status:    "Started",
	}
	key, err := datastore.Put(ctx, key, build)
	if err != nil {
		log.Panic(err)
	}
	return key, build
}

// LogRunComplete logs the end of a completed (failed of finished) run of
// a coduno testrun
func LogRunComplete(pKey *datastore.Key, build *BuildData, in string,
	out string, extra string, exit error) {
	tx, err := datastore.NewTransaction(ctx)
	if err != nil {
		log.Panic(err)
	}
	build.EndTime = time.Now()
	if exit != nil {
		build.Status = "FAILED"
	} else {
		build.Status = "DONE"
	}
	_, err = tx.Put(pKey, build)
	if err != nil {
		log.Panic(err)
		err = tx.Rollback()
		if err != nil {
			log.Panic(err)
		}
	}
	data := &LogData{
		InLog:    in,
		OutLog:   out,
		ExtraLog: extra,
	}
	k := datastore.NewIncompleteKey(ctx, "testrun", pKey)
	_, err = tx.Put(k, data)
	if err != nil {
		log.Panic(err)
		err = tx.Rollback()
		if err != nil {
			log.Panic(err)
		}
	}
	_, err = tx.Commit()
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
	if len(os.Args) != 5 {
		log.Fatal(os.Args)
	}

	username := os.Args[1]
	repo := os.Args[2]
	commit := os.Args[3]
	tmpdir := os.Args[4]

	key, build := LogBuildStart(repo, commit, username)
	cmd := exec.Command(
		"sudo",
		"docker",
		"run",
		"--rm",
		"-v",
		tmpdir+":/run",
		"coduno/base")
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
	err = cmd.Wait()
	LogRunComplete(key, build, "", b1.String(), b2.String(), err)
}
