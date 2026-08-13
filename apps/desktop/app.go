package main

import "context"

type App struct {
	ctx     context.Context
	client  *AgentClient
	initErr error
}

func NewApp() *App {
	client, err := NewAgentClientFromEnv()
	return &App{client: client, initErr: err}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetSnapshot(limit int) (Snapshot, error) {
	if a.initErr != nil {
		return Snapshot{}, a.initErr
	}
	return a.client.Snapshot(limit)
}

func (a *App) ReadCapture(captureID string) (CaptureView, error) {
	if a.initErr != nil {
		return CaptureView{}, a.initErr
	}
	return a.client.ReadCapture(captureID)
}
