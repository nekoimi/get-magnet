FROM golang:1.22-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .
RUN go install cmd
RUN go build --ldflags "-extldflags -static" -o get-magnet cmd/main.go

FROM alpine:latest

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

ENV TZ=Asia/Shanghai

COPY --from=builder /build/get-magnet   /usr/bin/get-magnet

RUN apk add --no-cache tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /workspace

EXPOSE 8093

ENTRYPOINT ["get-magnet"]
