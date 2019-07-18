all: fmonitor fmonitor.exe

fmonitor:
	GOOS=linux go build -ldflags '-w -s -extldflags "-static"' -o $@ ./cmd/fmonitor

fmonitor.exe:
	GOOS=windows go build -ldflags '-w -s -extldflags "-static"' -o $@ ./cmd/fmonitor
