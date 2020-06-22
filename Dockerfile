FROM golang:alpine AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o main .

FROM alpine
COPY --from=builder /build/main /datamon
COPY files /files
COPY config.yaml /
ENTRYPOINT ["/datamon"]