FROM golang:1.23.2-alpine AS gobuild

WORKDIR /appbuild
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o flabergnomebot ./cmd/gnomotron/

FROM alpine
WORKDIR /app
COPY --from=gobuild /appbuild/ /app/
EXPOSE 8080
CMD ["./flabergnomebot"]
