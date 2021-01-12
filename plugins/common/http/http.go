package http

import (
	"fmt"
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
