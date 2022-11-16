FROM golang:latest AS build

ADD go.mod \
    go.sum \
    lint.go \
    /build/

WORKDIR /build
RUN go build -o /usr/bin/lint lint.go

FROM debian:stable-slim

COPY --from=build /usr/bin/lint /usr/bin/lint

RUN apt-get update && apt-get upgrade -y

ENTRYPOINT ["/usr/bin/lint", "conf/manifest.yaml"]