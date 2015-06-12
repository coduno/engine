package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
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
	InLog      string `datastore:",noindex"`
	OutLog     string `datastore:",noindex"`
	ExtraLog   string `datastore:",noindex"`
	PrepareLog string `datastore:",noindex"`
	SysUsage   syscall.Rusage
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
	out string, extra string, exit error, prepLog string, stats syscall.Rusage) {
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
		InLog:      in,
		OutLog:     out,
		ExtraLog:   extra,
		PrepareLog: prepLog,
		SysUsage:   stats,
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

func pipeOutput(out io.ReadCloser, dest io.Writer, logBuf *bytes.Buffer) {
	tempBuf := make([]byte, 1024)
	writeErr := error(nil)
	r, readErr := int(0), error(nil)

	defer out.Close()
	defer sendSig()

	for readErr == nil {
		r, readErr = out.Read(tempBuf)

		if logBuf != nil {
			logBuf.Write(tempBuf[0:r])
		}

		if dest != nil && r != 0 && writeErr == nil {
			_, writeErr := dest.Write(tempBuf[0:r])
			if writeErr != nil {
				log.Print(writeErr)
			}
		}
	}
}

func main() {
	if len(os.Args) != 6 {
		log.Fatal(os.Args)
	}

	username := os.Args[1]
	repo := os.Args[2]
	commit := os.Args[3]
	tmpdir := os.Args[4]
	testdir := os.Args[5]

	key, build := LogBuildStart(repo, commit, username)
	cmdUser := exec.Command(
		"sudo",
		"docker",
		"run",
		"--rm",
		"-v",
		tmpdir+":/run",
		"coduno/base")
	cmdTest := exec.Command(
		"sudo",
		"docker",
		"run",
		"--rm",
		"-v",
		testdir+":/run",
		"coduno/base")
	outUser, err := cmdUser.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	errUser, err := cmdUser.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	inUser, err := cmdUser.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	outTest, err := cmdTest.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	errTest, err := cmdTest.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	inTest, err := cmdTest.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	var userToTest bytes.Buffer
	var testToUser bytes.Buffer
	var extraBuf bytes.Buffer
	cmdUser.Start()
	cmdTest.Start()
	go pipeOutput(outUser, inTest, &userToTest)
	go pipeOutput(outTest, inUser, &testToUser)
	go pipeOutput(errUser, os.Stderr, nil)
	go pipeOutput(errTest, ioutil.Discard, &extraBuf)

	<-signal
	<-signal
	<-signal
	<-signal
	exitErr := cmdUser.Wait()

	prepLog, err := ioutil.ReadFile(tmpdir + "/prepare.log")
	if err != nil { // This file should always exist, so an error here should never happen
		log.Fatal(err)
	}

	var stats syscall.Rusage
	statsData, err := ioutil.ReadFile(tmpdir + "/stats.log")
	if err != nil {
		log.Print(err)
	} else {
		err = json.Unmarshal(statsData, &stats)
		if err != nil {
			log.Fatal(err)
		}
	}

	LogRunComplete(key, build, testToUser.String(), userToTest.String(), extraBuf.String(), exitErr, string(prepLog), stats)
}
