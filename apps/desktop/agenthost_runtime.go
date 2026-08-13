//go:build !dcatest

package main

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/xiaoqianran/defuddle-clipper-agent/apps/agent/host"
)

func startInProcessAgent(ctx context.Context, addr string) (func(context.Context) error, string, error) {
	srv, err := host.New(productionLogger)
	if err != nil {
		return nil, "", err
	}
	listenAddr := addr
	if listenAddr == "" {
		listenAddr = srv.Addr
	}
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, "", err
	}
	srv.HTTP.Addr = ln.Addr().String()

	if productionLogger != nil {
		productionLogger.Printf("data dir: %s", srv.DataDir)
		if srv.Token == "" {
			productionLogger.Printf("warning: DCA_TOKEN is empty; loopback-only mode is relying on local network isolation")
		}
	}

	go func() {
		if err := srv.HTTP.Serve(ln); err != nil && err != http.ErrServerClosed && productionLogger != nil {
			productionLogger.Printf("agent http: %v", err)
		}
	}()

	readyCtx := ctx
	if readyCtx == nil {
		readyCtx = context.Background()
	}
	waitCtx, cancel := context.WithTimeout(readyCtx, 2*time.Second)
	defer cancel()
	if err := waitHealthy(waitCtx, httpBaseURL(ln.Addr().String()), ProbeHealth); err != nil {
		_ = srv.HTTP.Close()
		return nil, "", err
	}
	return srv.HTTP.Shutdown, ln.Addr().String(), nil
}
