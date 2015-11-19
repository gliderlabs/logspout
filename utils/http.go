package utils

import (
	"net/http"
	"io/ioutil"
)

func HttpGet(url string) (string, error) {
	resp, err := http.Get(url)
    	if err != nil {
        	// handle error
		return "", err
    	}

    	defer resp.Body.Close()
    	body, err := ioutil.ReadAll(resp.Body)
    	if err != nil {
        	// handle error
		return "", err
    	}
	return string(body), err
}
