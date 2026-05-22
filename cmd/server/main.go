package main

import (
	"log"
	"net/http"

	"backend-assignment/internal/catalog"
	"backend-assignment/internal/handlers"
	"backend-assignment/internal/ratelimiter"
)

func main() {
	rateLimiter := ratelimiter.NewRateLimiter()
	productStore := catalog.NewProductStore()

	rateHandler := handlers.NewRateLimiterHandler(rateLimiter)
	productHandler := handlers.NewProductHandler(productStore)

	mux := http.NewServeMux()
	mux.Handle("/request", rateHandler)
	mux.Handle("/stats", rateHandler)
	mux.Handle("/products/", productHandler)
	mux.Handle("/products", productHandler)

	addr := ":8080"
	log.Printf("server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
