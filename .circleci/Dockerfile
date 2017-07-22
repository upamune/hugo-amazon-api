FROM golang:1.8-alpine

ENV GLIDE_VERSION 0.12.3

RUN apk add --no-cache --update curl ca-certificates git && \
    curl -fsSL -o glide.tar.gz https://github.com/Masterminds/glide/releases/download/v${GLIDE_VERSION}/glide-v${GLIDE_VERSION}-linux-amd64.tar.gz && \
    tar -xzf glide.tar.gz && \
    mv linux-amd64/glide /usr/bin && \
    rm -fr linux-amd64 glide.tar.gz && \
    go get -ldflags "-w -s" \
    github.com/mitchellh/gox \
    github.com/tcnksm/ghr

     
