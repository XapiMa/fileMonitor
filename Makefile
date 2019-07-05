all: fileMonitor fileMonitor.exe

fileMonitor:
	GOOS=linux go build -ldflags '-w -s -extldflags "-static"' -o $@ ./cmd/fileMonitor

fileMonitor.exe:
	GOOS=windows go build -ldflags '-w -s -extldflags "-static"' -o $@ ./cmd/fileMonitor
