package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world, ini adalah test"))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}