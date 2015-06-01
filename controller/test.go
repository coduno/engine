package main

import (
	"bytes"
	"io"
	"log"
	"os"

	"io/ioutil"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/datastore/v1beta2"
)

var signal = make(chan int)
var secret []byte
var config *jwt.Config
var service *datastore.Service

func init() {
	err := error(nil)
	secret, err = ioutil.ReadFile("../config/Coduno-6b4d6d5a0f06.json")
	if err != nil {
		panic(err)
	}
	config, err = google.JWTConfigFromJSON(secret, datastore.DatastoreScope)
	if err != nil {
		panic(err)
	}
	client := config.Client(context.Background())
	service, err = datastore.New(client)
	if err != nil {
		panic(err)
	}
}

// LogBuildStart sends info to the datastore, informing that a new build
// started
func LogBuildStart(repo string, commit string, user string) {
	insert := new(datastore.CommitRequest)
	insert.Mode = "NON_TRANSACTIONAL"
	path := make([]*datastore.KeyPathElement, 1)
	path[0] = new(datastore.KeyPathElement)
	path[0].Kind = "testrun"
	mutation := new(datastore.Mutation)
	mutation.InsertAutoId = make([]*datastore.Entity, 1)
	mutation.InsertAutoId[0] = new(datastore.Entity)
	mutation.InsertAutoId[0].Properties = make(map[string]datastore.Property)
	mutation.InsertAutoId[0].Properties["repo"] = datastore.Property{StringValue: repo}
	mutation.InsertAutoId[0].Properties["commit"] = datastore.Property{StringValue: commit}
	mutation.InsertAutoId[0].Properties["user"] = datastore.Property{StringValue: user}
	mutation.InsertAutoId[0].Key = new(datastore.Key)
	mutation.InsertAutoId[0].Key.Path = path

	req := service.Datasets.Commit("coduno", insert)
	ret, err := req.Do()
	if err != nil {
		log.Panic(err)
	}
	log.Print(ret.Header)
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
