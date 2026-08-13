package main

import (
	"context"
	"fmt"
	"time"
)

type App struct {
	ctx     context.Context
	client  *AgentClient
	initErr error
	emit    func(ctx context.Context, name string, data ...interface{})
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.watchEvents()
}

func (a *App) watchEvents() {
	if a.client == nil || a.ctx == nil {
		return
	}
	for a.ctx.Err() == nil {
		err := a.client.WatchEvents(a.ctx, func(ev ChangeEvent) {
			if a.emit != nil && a.ctx != nil {
				a.emit(a.ctx, "dca:changed", ev)
			}
		})
		if a.ctx.Err() != nil {
			return
		}
		if err != nil {
			time.Sleep(time.Second)
		}
	}
}

func (a *App) GetSnapshot(limit int) (Snapshot, error) {
	if err := a.agentReady(); err != nil {
		return Snapshot{}, err
	}
	return a.client.Snapshot(limit)
}

func (a *App) ReadCapture(captureID string) (CaptureView, error) {
	if err := a.agentReady(); err != nil {
		return CaptureView{}, err
	}
	return a.client.ReadCapture(captureID)
}

func (a *App) ReprocessCapture(captureID string) (ReprocessResult, error) {
	if err := a.agentReady(); err != nil {
		return ReprocessResult{}, err
	}
	return a.client.ReprocessCapture(captureID)
}

func (a *App) GetPolicy() (Policy, error) {
	if err := a.agentReady(); err != nil {
		return Policy{}, err
	}
	return a.client.GetPolicy()
}

func (a *App) PutPolicy(doc Policy) (Policy, error) {
	if err := a.agentReady(); err != nil {
		return Policy{}, err
	}
	return a.client.PutPolicy(doc)
}

func (a *App) agentReady() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.client == nil {
		return fmt.Errorf("local agent is not available")
	}
	return nil
}
