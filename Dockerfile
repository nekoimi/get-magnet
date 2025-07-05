FROM golang:1.22-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .
RUN go install cmd
RUN go build --ldflags "-extldflags -static" -o get-magnet cmd/main.go

FROM ghcr.io/nekoimi/get-magnet-runtime:latest

LABEL maintainer="nekoimi <nekoimime@gmail.com>"

COPY --from=builder /build/get-magnet   /usr/bin/get-magnet

EXPOSE 8093

ENTRYPOINT ["get-magnet"]
