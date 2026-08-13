//go:build !dcatest

// dcatest 标签会排除此 Wails 入口，以便 `go test -tags dcatest`
// 无需嵌入 frontend/dist 或链接 WebKit 即可运行。

package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
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
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		println("Error:", err.Error())
	}
}
