copy: build
	scp 8x8 root@jeff.co.tz:/usr/local/bin

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build