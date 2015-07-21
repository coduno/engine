package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"

	"github.com/coduno/app/models"
	"github.com/coduno/app/util"
	"github.com/coduno/piper"
)

var (
	fileNames = map[string]string{
		"python": "app.py",
		"c":      "app.c",
		"cpp":    "app.cpp",
		"java":   "Application.java",
	}
)

const configFileName string = "coduno.yaml"

func startSimpleRun(w http.ResponseWriter, r *http.Request) {
	if !util.CheckMethod(w, r, "POST") {
		return
	}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Error reading: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var codeData models.CodeData
	err = json.Unmarshal(body, &codeData)

	if err != nil {
		http.Error(w, "Cannot unmarshal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for availableLanguage := range fileNames {
		if codeData.Language == availableLanguage {
			goto LANGUAGE_AVAILABLE
		}
	}
	http.Error(w, "Language not available.", http.StatusBadRequest)

LANGUAGE_AVAILABLE:
	tempDir, err := prepareFilesForDockerRun(codeData.Language, codeData.CodeBase)

	if err != nil {
		http.Error(w, "File preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	prepareAndSimpleRun(w, r, tempDir, codeData.CodeBase)
}

func prepareFilesForDockerRun(lang, codeBase string) (tempDir string, err error) {
	tempDir, err = volumeDir()
	if err != nil {
		return
	}
	err = createConfigurationFile(tempDir, lang)
	if err != nil {
		return
	}
	err = createExecFile(tempDir, lang, codeBase)
	if err != nil {
		return
	}
	return tempDir, nil
}

func prepareAndSimpleRun(w http.ResponseWriter, r *http.Request, tempDir, codeBase string) {
	key, build := piper.LogBuildStart("challengeId", codeBase, "user")

	volume, err := dockerize(tempDir)

	if err != nil {
		log.Fatal(err)
	}

	cmdUser := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v",
		volume+":/run",
		"coduno_all")

	outUser, err := cmdUser.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	errUser, err := cmdUser.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	_, err = cmdUser.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	var runOutput, runErr bytes.Buffer
	cmdUser.Start()
	var wg sync.WaitGroup
	wg.Add(2)

	go piper.PipeOutput(&wg, outUser, os.Stdout, &runOutput)
	go piper.PipeOutput(&wg, errUser, os.Stdout, &runErr)

	exitErr := cmdUser.Wait()
	wg.Wait()
	prepLog, err := ioutil.ReadFile(tempDir + "/prepare.log")
	if err != nil {
		log.Fatal(err)
	}

	var stats syscall.Rusage
	statsData, err := ioutil.ReadFile(tempDir + "/stats.log")
	if err != nil {
		log.Print(err)
	} else {
		err = json.Unmarshal(statsData, &stats)
		if err != nil {
			log.Fatal(err)
		}
	}

	piper.LogRunComplete(key, build, "", runOutput.String(), "", exitErr, string(prepLog), stats)

	var toSend = make(map[string]string)
	toSend["run"] = runOutput.String()
	toSend["err"] = runErr.String()

	json, err := json.Marshal(toSend)
	if err != nil {
		http.Error(w, "Json marshal err: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(json)
}

func createExecFile(tmpDir, lang, codeBase string) (err error) {
	f, err := os.Create(path.Join(tmpDir, fileNames[lang]))
	if err != nil {
		return
	}
	f.WriteString(codeBase)
	f.Close()
	return
}

func createConfigurationFile(tempDir, lang string) error {
	src := path.Join(".", "run_config", lang, "coduno.yaml")
	return copyFileContents(tempDir, src, configFileName)
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(dst, src, fileName string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	dst = dst + "/" + fileName
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	return
}

func main() {
	http.HandleFunc("/api/run/start/simple", startSimpleRun)
	http.ListenAndServe(":8081", nil)
}
