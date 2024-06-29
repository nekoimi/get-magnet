FROM golang:1.20.14-alpine as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /build
COPY . .
RUN go install
RUN go build --ldflags "-extldflags -static" -o get-magnet main.go

FROM alpine:latest

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

ENV TZ=Asia/Shanghai

COPY --from=builder /build/get-magnet   /usr/bin/get-magnet

WORKDIR /workspace

ENTRYPOINT ["get-magnet"]
