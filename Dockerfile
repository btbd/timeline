FROM golang:1.9 as builder
COPY main.go src/
RUN GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 go build \
    -ldflags "-w -extldflags -static" \
    -tags netgo -installsuffix netgo \
    -o /timeline ./src/main.go

FROM scratch
COPY --from=builder /timeline /timeline
COPY index.html /
ENTRYPOINT [ "/timeline" ]