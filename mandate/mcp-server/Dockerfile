FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mcp-server .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/mcp-server .
EXPOSE 8090
ENTRYPOINT ["./mcp-server", "-addr", ":8090"]
