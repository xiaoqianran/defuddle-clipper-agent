//go:build !dcatest

// dcatest 标签会排除此 Wails 入口，以便 `go test -tags dcatest`
// 无需嵌入 frontend/dist 或链接 WebKit 即可运行。

package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// productionLogger 由 main 设置，供 in-process agent 写入同一日志。
var productionLogger *log.Logger

func main() {
	logger, logCloser := newDesktopLogger()
	if logCloser != nil {
		defer logCloser.Close()
	}
	productionLogger = logger

	hostLife := &AgentHost{
		Addr:      listenAddrFromEnv(),
		ClientURL: explicitAgentURLFromEnv(),
		Logger:    logger,
		Probe:     ProbeHealth,
		Start:     startInProcessAgent,
	}

	app := NewApp()
	clientURL, err := hostLife.Ensure(context.Background())
	if err != nil {
		logger.Printf("agent host: %v", err)
		app.initErr = err
	} else {
		client, clientErr := NewAgentClient(clientURL, agentTokenFromEnv())
		if clientErr != nil {
			logger.Printf("agent client: %v", clientErr)
			app.initErr = clientErr
		} else {
			app.client = client
		}
	}

	if err := wails.Run(&options.App{
		Title:     "Defuddle Browser Mirror",
		Width:     1500,
		Height:    950,
		MinWidth:  1000,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 20, A: 1},
		OnStartup:        app.startup,
		OnShutdown: func(ctx context.Context) {
			stopCtx, cancel := context.WithTimeout(context.Background(), ownedShutdownWait)
			defer cancel()
			if err := hostLife.Stop(stopCtx); err != nil {
				logger.Printf("stop agent: %v", err)
			}
		},
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		logger.Printf("Error: %s", err.Error())
		println("Error:", err.Error())
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), ownedShutdownWait)
	defer cancel()
	if err := hostLife.Stop(stopCtx); err != nil {
		logger.Printf("stop agent: %v", err)
	}
}
