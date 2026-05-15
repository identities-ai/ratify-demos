FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o vendor-mcp-server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/vendor-mcp-server .
EXPOSE 8091
ENTRYPOINT ["./vendor-mcp-server", "-addr", ":8091"]
