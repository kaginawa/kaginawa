package main

import (
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"crypto/sha256"
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
	newVer, newest := latest()
	if newest {
		return false
	}
	log.Printf("starging version up process: %s -> %s", ver, newVer)
	url := binaryURL()
	if len(url) == 0 {
		log.Printf("automatic update disabled due to unsupported machine: %s %s", runtime.GOOS, runtime.GOARCH)
		return true
	}
	archive, err := download(url)
	if err != nil {
		log.Printf("failed to download version %s: %v", newVer, err)
		return false
	}
	checksum, err := download(url + ".sha256")
	if err != nil {
		log.Printf("failed to download checksum: %v", err)
		return false
	}
	if !validate(archive, checksum) {
		log.Print("checksum error")
		return false
	}
	tempFileName, err := extract(archive)
	if err != nil {
		log.Printf("failed to extract version %s: %v", newVer, err)
		return false
	}
	if err := replace(tempFileName); err != nil {
		log.Printf("automatic update disabled due to binary replacement failed: %v", err)
		return true
	}
	if len(config.UpdateCommand) > 0 {
		log.Print("download complete. now executing restart...")
		restart()
		return true
	}
	log.Printf("download complete. please restart process manually.")
	return true
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

func binaryURL() string {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		return strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.linux-x64.bz2", 1)
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm" {
		if machine, err := exec.Command("uname", "-m").Output(); err != nil {
			if strings.HasPrefix(string(machine), "armv6") {
				return strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.linux-arm6.bz2", 1)
			}
		}
		return strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.linux-arm.bz2", 1)
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
		return strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.macos-x64.bz2", 1)
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		return strings.Replace(config.UpdateCheckURL, "LATEST", "kaginawa.exe.zip", 1)
	}
	return ""
}

func download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer safeClose(resp.Body, "download link")
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("empty body")
	}
	return body, nil
}

func validate(content []byte, checksum []byte) bool {
	hash := sha256.Sum256(content)
	log.Printf("expected %s", string(checksum))
	log.Printf("actual   %s", fmt.Sprintf("%x", hash))
	return strings.HasPrefix(string(checksum), fmt.Sprintf("%x", hash))
}

func extract(content []byte) (string, error) {
	// Create temp file
	tempFile, err := ioutil.TempFile("", "kgnw")
	if err != nil {
		return "", err
	}
	defer safeClose(tempFile, "temp file")

	// extract to temp file
	if runtime.GOOS == "windows" {
		r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
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
		if _, err := io.Copy(tempFile, bzip2.NewReader(bytes.NewReader(content))); err != nil {
			return "", err
		}
	}
	stat, err := tempFile.Stat()
	if err != nil {
		return "", fmt.Errorf("cannot stat downloaded file: %v", err)
	}
	if stat.Size() == 0 {
		safeRemove(tempFile.Name())
		return "", fmt.Errorf("empty body: %s", tempFile.Name())
	}
	return tempFile.Name(), nil
}

func replace(tempFileName string) error {
	// kaginawa -> kaginawa.old
	if err := os.Rename(os.Args[0], os.Args[0]+".old"); err != nil {
		return fmt.Errorf("failed to move file: %v", err)
	}
	log.Printf("current binary has been moved to " + os.Args[0] + ".old")

	// tmp -> kaginawa
	if err := os.Rename(tempFileName, os.Args[0]); err != nil {
		if err := os.Rename(os.Args[0]+".old", os.Args[0]); err != nil {
			return fmt.Errorf("failed to recover file: %v", err)
		}
		log.Printf("binary recovered using old file: %s.old", os.Args[0])
		return fmt.Errorf("failed to move file: %v", err)
	}

	// make executable
	if runtime.GOOS != "windows" {
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

func safeRemove(name string) {
	if err := os.Remove(name); err != nil {
		log.Printf("failed to remove %s: %v", name, err)
	}
}
