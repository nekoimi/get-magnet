#!/usr/bin/env bash

protoc -I. --go_out=../ --go-grpc_out=../ fetch.proto

