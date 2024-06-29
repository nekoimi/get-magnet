FROM golang:1.20.14-alpine as builder

ARG BINARY_NAME="get-magnet"

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /build
COPY . .
RUN go install
RUN go build --ldflags "-extldflags -static" -o $BINARY_NAME main.go

FROM alpine:latest

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

ENV TZ=Asia/Shanghai

COPY --from=builder /build/$BINARY_NAME   /usr/bin/$BINARY_NAME

WORKDIR /workspace

ENTRYPOINT ["$BINARY_NAME"]
