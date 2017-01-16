package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

func Upload(url, file string) (string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fw, err := w.CreateFormFile("file", file)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return "", err
	}
	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
		return "", err
	}
	reqId := res.Header.Get("req_id")
	return string(reqId), nil
}

func main() {
	reqId, err := Upload("http://localhost:8000/requests", "test.txt")
	if err != nil {
		fmt.Println("error in post", err)
		return
	}

	newUrl := "http://localhost:8000/requests/" + reqId

	fmt.Println("RequestID:", reqId)
	fmt.Println("Request Loop at interval of 10 secs")
	for i := 0; i < 10; i++ {
		res, err := http.Get(newUrl)
		if err != nil {
			fmt.Println("error accessing ", newUrl)
			return
		}
		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			fmt.Println("error in reading response body", err)
			return
		}
		fmt.Println(string(body))
		time.Sleep(5 * time.Second)
	}
}
