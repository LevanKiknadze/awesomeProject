package main

import (
	"os"
	"time"
	"os/signal"
	"log"
	"net/http"
	"awesomeProject/http_server"
	"golang.org/x/net/context"
)

func main() {
	// create channel for listening shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	logger := log.New(os.Stdout, "", 0)

	// get port from env variables
	addr := ":" + os.Getenv("PORT")
	if addr == ":" {
		addr = ":8080"
	}

	// create server
	s := http_server.New()
	h := &http.Server{Addr: addr, Handler: s}

	// execute listener
	go func() {
		logger.Printf("Listening on http://0.0.0.0%s\n", addr)

		if err := h.ListenAndServe(); err != nil {
			logger.Fatal(err)
		}
	}()


	// waiting on shutdown signal
	<-stop

	// shutdown the server
	logger.Println("\nShutting down the server...")

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	h.Shutdown(ctx)

	logger.Println("Server gracefully stopped")
}
