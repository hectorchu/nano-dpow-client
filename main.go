package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	port        = flag.Int("p", 7076, "Listen port")
	user        = flag.String("u", "", "User")
	apiKey      = flag.String("k", "", "API key")
	fallbackURL = flag.String("f", "", "Fallback RPC URL")
)

func main() {
	flag.Parse()
	if *user == "" || *apiKey == "" {
		fmt.Println("User and API key are required")
		os.Exit(1)
	}
	c := newClient()
	if err := c.connect(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError := func(err error) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(err)
			return
		}
		var v struct{ Action, Hash, Difficulty string }
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&v); err != nil {
			writeError(err)
			return
		}
		if v.Action != "work_generate" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "Action is not supported"})
			return
		}
		if work, done, err := process(r.Context(), c, v.Hash, v.Difficulty); err == nil {
			json.NewEncoder(w).Encode(map[string]string{"work": work})
		} else if done {
			return
		} else if *fallbackURL != "" {
			resp, err := http.Post(*fallbackURL, "application/json", bytes.NewReader(body))
			if err != nil {
				writeError(err)
				return
			}
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			resp.Body.Close()
		} else {
			writeError(err)
		}
	})
	if err := http.ListenAndServe(fmt.Sprint(":", *port), nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func process(ctx context.Context, c *client, hash, difficulty string) (work string, done bool, err error) {
	ch := make(chan *response, 1)
	if err = c.request(hash, difficulty, ch); err != nil {
		return
	}
	select {
	case v := <-ch:
		if v.Error != "" {
			return "", false, errors.New(v.Error)
		}
		return v.Work, false, nil
	case <-ctx.Done():
		return "", true, ctx.Err()
	}
}
