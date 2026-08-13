.PHONY: build build-agent build-extension build-desktop test test-agent typecheck clean

build: build-agent build-extension build-desktop

build-agent:
	cd apps/agent && go build ./cmd/clipper-agent

build-extension:
	npm run build

# Windows 桌面只需 Go 的 production 标签，不要走 wails CLI / gcc。
build-desktop:
	cd apps/desktop && CGO_ENABLED=0 go build -tags desktop,production .

test: test-agent typecheck

test-agent:
	cd apps/agent && go test ./...

typecheck:
	npm run typecheck

clean:
	cd apps/agent && go clean
	npm run clean
