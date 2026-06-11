package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8010", "HTTP listen address")
	dir := flag.String("dir", "../dashboard", "dashboard directory")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(*dir)))

	log.Printf("Mandate dashboard  addr=%s  dir=%s", *addr, *dir)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("dashboard: %v", err)
	}
}
