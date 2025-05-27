FROM golang:1.24.3-bookworm AS build

WORKDIR /build

COPY src/go.mod src/go.sum ./

RUN go mod download

COPY src/. .

RUN go build -o e6-cache -ldflags "-X main.debugMode=false"

# copy to simpler image
FROM scratch
COPY --from=build /bin/e6-cache /bin/e6-cache
CMD ["/bin/e6-cache"]

EXPOSE 8080
