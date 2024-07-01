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

RUN apk add tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

WORKDIR /workspace

CMD ["get-magnet"]
