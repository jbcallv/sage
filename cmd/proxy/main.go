package main

import (
	"log"
	"net/http"

	"github.com/jbcallv/agentmesh/internal/policy"
)

func main() {
	engine, err := policy.Load(mustReadPolicy())
	if err != nil {
		log.Fatal(err)
	}
	p := &Proxy{
		store:  NewStore(),
		engine: engine,
		key:    rootKey(),
		client: http.DefaultClient,
	}
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/bootstrap", p.handleBootstrap)
	http.HandleFunc("/", p.handleRequest)
	log.Println("proxy listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
