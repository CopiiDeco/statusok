FROM golang:1.12.7 AS builder
WORKDIR /go/src/github.com/CopiiDeco/statusok/
COPY statusok.go ./
COPY database ./database/
COPY go.mod ./
COPY notify ./notify/
COPY requests ./requests/
RUN env GO111MODULE=on go get
RUN env CGO_ENABLED=0 GOOS=linux go build 

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/CopiiDeco/statusok/statusok ./
VOLUME /config
COPY ./docker-entrypoint.sh /docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT /docker-entrypoint.sh
