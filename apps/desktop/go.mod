module github.com/xiaoqianran/defuddle-clipper-agent/apps/desktop

go 1.22.0

require (
	github.com/wailsapp/wails/v2 v2.12.0
	github.com/xiaoqianran/defuddle-clipper-agent/apps/agent v0.0.0
)

replace github.com/xiaoqianran/defuddle-clipper-agent/apps/agent => ../agent
