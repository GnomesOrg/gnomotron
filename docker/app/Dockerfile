FROM golang:1.23.2-alpine AS gobuild

WORKDIR /appbuild
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o flabergnomebot ./cmd/gnomotron/

FROM alpine AS certs
RUN apk add --no-cache ca-certificates

FROM alpine
WORKDIR /app

RUN apk add --no-cache tzdata ca-certificates && update-ca-certificates
ENV TZ="Asia/Yekaterinburg"

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=gobuild /appbuild/ /app/
EXPOSE 8080
CMD ["./flabergnomebot"]
