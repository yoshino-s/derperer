FROM golang:1.20-alpine as build-stage

ARG VERSION
ENV VERSION=${VERSION}

WORKDIR /src
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -o /main --ldflags "-w -extldflags '-static' -X cmd.Version=${VERSION:-dev}" .

# Final image.
FROM alpine:latest
COPY --from=build-stage /main /usr/local/bin/main

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/main", "server"]
