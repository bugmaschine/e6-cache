FROM golang:1.24.3-bookworm AS build

WORKDIR /build

COPY src/go.mod src/go.sum ./

RUN go mod download

COPY src/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o e6-cache -ldflags "-s -w -X main.debugMode=false"

# copy to simpler image
FROM alpine:latest
COPY --from=build /build/e6-cache /build/e6-cache
CMD ["/build/e6-cache"]

EXPOSE 8080
