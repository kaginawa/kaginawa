package main

import (
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func updateChecker() {
	if checkAndUpdate() {
		return
	}
	for range time.Tick(24 * time.Hour) {
		if checkAndUpdate() {
			return
		}
	}
}

func checkAndUpdate() (finished bool) {
	if newVer, newest := latest(); !newest {
		log.Printf("starging version up process: %s -> %s", ver, newVer)
		if tempFileName, err := download(); err != nil {
			log.Printf("failed to download version %s: %v", newVer, err)
		} else {
			if err := replace(tempFileName); err != nil {

			} else {
				if len(config.UpdateCommand) > 0 {
					log.Print("download complete. now executing restart...")
					restart()
				}
				return true
			}
		}
	}
	return false
}

func latest() (string, bool) {
	resp, err := http.Get(config.UpdateCheckURL)
	if err != nil {
		return ver, true // may offline
	}
	defer safeClose(resp.Body, "update check body")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ver, true
	}
	latest := strings.TrimSpace(string(body))
	currentVer := ver
	i := strings.Index(ver, "-")
	if i > 0 {
		currentVer = ver[:i] // trim commit number (ex. v0.0.1-18-g2c63e8b -> v0.0.1)
	}
	return latest, currentVer == latest
}

func download() (string, error) {
	url := ""
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		url = strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.linux-x64.bz2", 1)
	} else if runtime.GOOS == "linux" && runtime.GOARCH == "arm" {
		url = strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.linux-arm.bz2", 1)
	} else if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
		url = strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.macos.bz2", 1)
	} else if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		url = strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.exe.zip", 1)
	} else {
		return "", fmt.Errorf("unsupporte os: %v", runtime.GOOS)
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer safeClose(resp.Body, "download link")
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tempFile, err := ioutil.TempFile("", "kgnw")
	if err != nil {
		return "", err
	}
	defer safeClose(tempFile, "temp file")

	// extract to temp file
	if runtime.GOOS == "windows" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		r, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return "", nil
		}
		var exeFile io.ReadCloser
		for _, zf := range r.File {
			if zf.Name == "kaginawa.exe" {
				rc, err := zf.Open()
				if err != nil {
					return "", err
				}
				exeFile = rc
				break
			}
		}
		if exeFile == nil {
			return "", errors.New("kaginawa.exe not found in fetched zip file")
		}
		defer safeClose(exeFile, "zip file entry")
		if _, err := io.Copy(tempFile, exeFile); err != nil {
			return "", err
		}
	} else {
		if _, err := io.Copy(tempFile, bzip2.NewReader(resp.Body)); err != nil {
			return "", err
		}
	}
	return tempFile.Name(), nil
}

func replace(tempFileName string) error {
	if runtime.GOOS == "windows" {
		// tmp -> kaginawa.new
		if err := os.Rename(tempFileName, os.Args[0]+".new"); err != nil {
			return fmt.Errorf("failed to move file: %v", err)
		}
		log.Printf("downloaded " + os.Args[0] + ".new")
		log.Print("Please rename to actual file name after program stop manually.")
	} else {
		// kaginawa -> kaginawa.old
		if err := os.Rename(os.Args[0], os.Args[0]+".old"); err != nil {
			return fmt.Errorf("failed to move file: %v", err)
		}
		log.Printf("current binary has been moved to " + os.Args[0] + ".old")

		// tmp -> kaginawa
		if err := os.Rename(tempFileName, os.Args[0]); err != nil {
			return fmt.Errorf("failed to move file: %v", err)
		}

		// make executable
		if err := os.Chmod(os.Args[0], 0775); err != nil {
			log.Printf("failed to chmod: %s", os.Args[0])
		}
	}
	return nil
}

func restart() {
	split := strings.Split(config.UpdateCommand, " ")
	attrs := make([]string, len(split)-1)
	if len(split) > 0 {
		attrs = split[1:]
	}
	res, err := exec.Command(split[0], attrs...).Output()
	if err != nil {
		log.Printf("%s: %v", config.UpdateCommand, err)
	} else {
		log.Printf("%s: %s", config.UpdateCommand, res)
	}
}
