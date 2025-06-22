FROM golang:1.22-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .
RUN go install -v -x cmd
RUN go build --ldflags "-extldflags -static" -o get-magnet cmd/main.go

FROM alpine:latest

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

COPY --from=builder /build/get-magnet   /usr/bin/get-magnet

RUN apk add tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

WORKDIR /workspace

EXPOSE 8093

CMD ["get-magnet"]
