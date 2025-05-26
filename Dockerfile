FROM golang:1.24.3-bookworm AS base

WORKDIR /build

COPY src/go.mod src/go.sum ./

RUN go mod download

COPY src/. .

RUN go build -o e6-cache -ldflags "-X main.debugMode=false"

EXPOSE 8080

CMD ["/build/e6-cache"]
