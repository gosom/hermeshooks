package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// NGROK_URL with yours
	NGROK_URL = "https://86bd-164-215-13-207.ngrok.io"
	// Replace with your API KEY
	APIKEY = "r_AUiush4KF6CX1RLiDTzcnR9ZszKAnKOXnRLF7Xl3o="
	// Replace with your server url
	ServerUrl = "http://localhost:8000"
)

func main() {
	http.HandleFunc("/", schedule)
	http.HandleFunc("/webhook", wh)

	if err := http.ListenAndServe("127.0.0.1:8090", nil); err != nil {
		panic(err)
	}
}

// schedule justs schedule a webhook to be triggered after 5 minutes
func schedule(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runAt := time.Now().UTC().Add(time.Minute * 5)
	postBody, err := json.Marshal(map[string]interface{}{
		"name":        "hello-world",    // a descriptive name
		"description": "just to say hi", // a small description
		"url":         NGROK_URL + "/webhook",
		"payload":     `{"msg": "hi there"}`,      // the payload to receive. Notice, it should be  a string
		"contentType": "application/json",         // the content  type
		"signature":   "",                         // TODO
		"runAt":       runAt.Format(time.RFC3339), // when to happen
		"retries":     1,                          // how many times to retry
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	r, err := http.NewRequestWithContext(
		req.Context(),
		http.MethodPost,
		ServerUrl+"/api/v1/scheduledJobs",
		bytes.NewBuffer(postBody),
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-KEY", APIKEY)

	resp, err := http.DefaultClient.Do(r)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		w.WriteHeader(resp.StatusCode)
		return
	}
	w.WriteHeader(http.StatusCreated)

}

func wh(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	var p map[string]string
	if err := json.Unmarshal(body, &p); err != nil {
		panic(err)
	}

	fmt.Println(p["msg"]) // it should print hi there
}
