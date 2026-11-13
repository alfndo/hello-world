package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hidup jokowi v3"))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}