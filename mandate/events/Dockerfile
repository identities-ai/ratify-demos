FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -o event-relay .

FROM alpine:3.20
COPY --from=builder /app/event-relay /event-relay
EXPOSE 8099
ENTRYPOINT ["/event-relay"]
