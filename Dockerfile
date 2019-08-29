FROM golang:1.7.3 AS builder
WORKDIR /go/src/github.com/CopiiDeco/statusok/
RUN go get -d -v github.com/codegangsta/cli   
COPY statuook.go    .
RUN env GOOS=linux GOARCH=arm go build 

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/alexellis/href-counter/app .
VOLUME /config
COPY ./docker-entrypoint.sh /docker-entrypoint.sh
ENTRYPOINT /docker-entrypoint.sh
