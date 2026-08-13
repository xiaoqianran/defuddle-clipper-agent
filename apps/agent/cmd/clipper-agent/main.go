package main

import (
	"log"
	"net/http"
	"os"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/host"
)

func main() {
	logger := log.New(os.Stderr, "dca: ", log.LstdFlags|log.LUTC)

	srv, err := host.New(logger)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Printf("listening on http://%s", srv.Addr)
	logger.Printf("data dir: %s", srv.DataDir)
	if srv.Token == "" {
		logger.Printf("warning: DCA_TOKEN is empty; loopback-only mode is relying on local network isolation")
	}
	if err := srv.HTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal(err)
	}
}
