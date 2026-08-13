package main

import (
	"context"
	"fmt"
)

type App struct {
	ctx     context.Context
	client  *AgentClient
	initErr error
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

func (a *App) agentReady() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.client == nil {
		return fmt.Errorf("local agent is not available")
	}
	return nil
}
