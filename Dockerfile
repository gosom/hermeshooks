FROM golang:1.18.3-alpine as builder

LABEL maintainer="Georgios Komninos"
LABEL website="https://blog.gkomninos.com"

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go install ./...

FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /usr/bin

COPY --from=builder /go/bin .
