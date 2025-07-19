FROM golang:1.23.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o flabergnomebot ./cmd/gnomotron/

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata curl openssl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/flabergnomebot .

ENV TZ=Asia/Yekaterinburg

EXPOSE 8080

CMD ["./flabergnomebot"]
