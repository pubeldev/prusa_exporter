# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

COPY *.go ./

RUN go build -v -o /prusa_exporter

FROM alpine:3.22.2

COPY --from=builder /prusa_exporter .

EXPOSE 10009

ENTRYPOINT ["/prusa_exporter"]
