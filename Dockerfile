FROM golang:1.21 as builder

WORKDIR /app

COPY go.mod .
RUN cat go.mod
RUN go mod download || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o loadbalancer .

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/loadbalancer .
COPY config.json .
COPY backend1.html .
COPY backend2.html .

EXPOSE 8080
CMD ["./loadbalancer", "--config", "config.json"]