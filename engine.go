package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/coduno/app/models"
	"github.com/coduno/app/util"
	"github.com/coduno/piper"
	"github.com/m4rw3r/uuid"
)

var (
	// keep both home dir and docker home dir even if they are the same because
	// on windows we need to have a dockerHomeDir=/c/Users/$USER/tmp and a
	// homeDir=C:/Users/%USER%/tmp. On Linux they are homeDir=dockerHomeDir=$HOME/tmp
	homeDir       string
	dockerHomeDir string
	languages     = map[string]string{
		"python": "run_config/python/coduno.yaml",
		"c":      "run_config/c/coduno.yaml",
		"cpp":    "run_config/cpp/coduno.yaml",
		"java":   "run_config/java/coduno.yaml",
	}
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
	dockerTempDir := ""
	tempDir := ""
	switch codeData.Language {
	case "python":
		tempDir, dockerTempDir, err = prepareFilesForDockerRun("python", codeData.CodeBase)
	case "c":
		tempDir, dockerTempDir, err = prepareFilesForDockerRun("c", codeData.CodeBase)
	case "cpp":
		tempDir, dockerTempDir, err = prepareFilesForDockerRun("cpp", codeData.CodeBase)
	case "java":
		tempDir, dockerTempDir, err = prepareFilesForDockerRun("java", codeData.CodeBase)
	default:
		http.Error(w, "Language not available.", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "File preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	prepareAndSimpleRun(w, r, tempDir, dockerTempDir, codeData.CodeBase)
}

func prepareFilesForDockerRun(lang, codeBase string) (string, string, error) {
	tempDir, dockerTempDir, err := createTempDir()
	if err != nil {
		return "", "", err
	}
	err = createConfigurationFile(tempDir, lang)
	if err != nil {
		return "", "", err
	}
	err = createExecFile(tempDir, lang, codeBase)
	if err != nil {
		return "", "", err
	}
	return tempDir, dockerTempDir, nil
}

func prepareAndSimpleRun(w http.ResponseWriter, r *http.Request, tempDir, dockerTempDir, codeBase string) {
	key, build := piper.LogBuildStart("challengeId", codeBase, "user")

	cmdUser := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v",
		dockerTempDir+":/run",
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

func createExecFile(tmpDir, lang, codeBase string) error {
	f, err := os.Create(tmpDir + "/" + fileNames[lang])
	if err != nil {
		return err
	}
	f.WriteString(codeBase)
	f.Close()
	return nil
}

func createConfigurationFile(tempDir, lang string) error {
	if err := copyFileContents(languages[lang], tempDir, configFileName); err != nil {
		return err
	}
	return nil
}

func createTempDir() (string, string, error) {
	id, err := uuid.V4()
	if err != nil {
		return "", "", err
	}
	tempDir := homeDir + id.String()
	os.MkdirAll(tempDir, 0777)

	return tempDir, dockerHomeDir + id.String(), nil
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst, fileName string) (err error) {
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
	err = out.Sync()
	return
}

func main() {
	if runtime.GOOS == "windows" {
		u, err := user.Current()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		homeDir = u.HomeDir
		homeDir = strings.Replace(homeDir, "\\", "/", -1) + "/tmp/"
		if !strings.HasPrefix(u.HomeDir, "C:\\Users\\") {
			fmt.Println("Your home path should begin with C:\\Users\\ because you are on windows and that`s the only boot2docker shared folder")
			return
		}
		dockerHomeDir = "/" + homeDir
		dockerHomeDir = strings.Replace(dockerHomeDir, ":", "", 1)
		dockerHomeDir = strings.Replace(dockerHomeDir, "C", "c", 1)
		dockerHomeDir = strings.Replace(dockerHomeDir, "\\", "/", -1)
	} else {
		homeDir = os.Getenv("HOME") + "/tmp/"
		dockerHomeDir = homeDir
	}
	http.HandleFunc("/api/run/start/simple", startSimpleRun)
	http.ListenAndServe(":8081", nil)
}
