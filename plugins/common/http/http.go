package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type HttpGetResponse struct {
	Body    []byte
	Headers http.Header
}

func HttpGet(client *http.Client, url string) (response HttpGetResponse, err error) {
	r, err := client.Get(url)
	if err != nil {
		return
	}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	response = HttpGetResponse{body, r.Header}

	if r.StatusCode >= 300 {
		err = fmt.Errorf(string(response.Body))
		return
	}

	err = nil
	return
}

func HttpPost(client *http.Client, url string, reqBody io.Reader) (response HttpGetResponse, err error) {
	req, _ := http.NewRequest("POST", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	r, err := client.Do(req)
	if err != nil {
		return
	}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	response = HttpGetResponse{body, r.Header}

	if r.StatusCode >= 300 {
		err = fmt.Errorf(string(response.Body))
		return
	}

	//log.Trace("HTTP POST %s: %s %s", url, r.Status, string(response.Body))

	err = nil
	return
}
