FROM golang:1.22

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl tzdata ca-certificates openssl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .
RUN go build -o bot .
CMD ["./bot"]
