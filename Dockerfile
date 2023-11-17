FROM golang:1.20-alpine as builder
ARG VERSION
ARG CN
ENV VERSION=${VERSION}

# if CN is true, use goproxy.cn
RUN if [ "$CN" = "true" ] ; then \
    echo "using cn environment" \
    && go env -w GOPROXY=https://goproxy.cn,direct \
    && sed -i 's|deb.debian.org|mirrors.tuna.tsinghua.edu.cn|g' /etc/apt/sources.list \
    ;fi

WORKDIR /app
COPY . /app

RUN CGO_ENABLED=0 go build -o /derperer --ldflags "-w -extldflags '-static' -X git.yoshino-s.xyz/yoshino-s/derperer/cmd.Version=${VERSION:-dev}" .

# Final image.
FROM alpine:latest
COPY --from=builder /derperer /usr/local/bin/derperer

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/derperer", "server"]
