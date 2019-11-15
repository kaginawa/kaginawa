package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func measureRoundTripTimeMills() (int64, error) {
	begin := time.Now()
	resp, err := http.Get("http://" + config.Server + "/measure/10240") // Use http to reduce overhead
	if err != nil {
		return -1, err
	}
	defer safeClose(resp.Body, "measure body")
	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		return -1, err
	}
	elapsed := time.Since(begin).Milliseconds()
	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return elapsed, nil
}

func measureThroughput(kb int) (int64, int64, error) {
	downloadBegin := time.Now()
	dr, err := http.Get("http://" + config.Server + "/measure/" + strconv.Itoa(kb)) // Use http to reduce overhead
	if err != nil {
		return -1, -1, err
	}
	defer safeClose(dr.Body, "measure body")
	if _, err := ioutil.ReadAll(dr.Body); err != nil {
		return -1, -1, err
	}
	downloadSec := time.Since(downloadBegin).Seconds()
	if dr.StatusCode != http.StatusOK {
		return -1, -1, fmt.Errorf("HTTP %d", dr.StatusCode)
	}
	body := bytes.NewBuffer(make([]byte, kb*1024))
	uploadBegin := time.Now()
	ur, err := http.Post("http://"+config.Server+"/measure/"+strconv.Itoa(kb), "application/octet-stream", body)
	if err != nil {
		return -1, -1, err
	}
	defer safeClose(ur.Body, "measure body")
	if _, err := ioutil.ReadAll(ur.Body); err != nil {
		return -1, -1, err
	}
	uploadSec := time.Since(uploadBegin).Seconds()
	if ur.StatusCode != http.StatusOK {
		return -1, -1, fmt.Errorf("HTTP %d", ur.StatusCode)
	}
	return int64(float64(kb*8) / downloadSec), int64(float64(kb*8) / uploadSec), nil
}
