all: timeline

timeline: main.go
	GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 go build \
	-ldflags "-w -extldflags -static" \
	-tags netgo -installsuffix netgo \
	-o timeline main.go
