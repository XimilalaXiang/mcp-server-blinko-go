FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /blinko-mcp .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /blinko-mcp /usr/local/bin/blinko-mcp
ENTRYPOINT ["blinko-mcp"]
