package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			fmt.Print(err.Error())
		} else {
			fmt.Print(string(requestDump))
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"created":"ok"}`)
	})

	http.ListenAndServe("localhost:8080", nil)
}
