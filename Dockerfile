FROM golang:1.12.7 AS builder
WORKDIR /go/src/github.com/CopiiDeco/statusok/
RUN go get github.com/codegangsta/cli && go get github.com/influxdata/influxdb1-client/v2  && go get cloud.google.com/go/logging && go get github.com/sirupsen/logrus && go get github.com/mailgun/mailgun-go 
COPY . .
RUN ls -la /go/src/github.com/CopiiDeco/
RUN ls -la /go/src/github.com/CopiiDeco/statusok/
RUN env GOOS=linux GOARCH=arm go build 

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/CopiiDeco/statusok/statusok .
VOLUME /config
COPY ./docker-entrypoint.sh /docker-entrypoint.sh
ENTRYPOINT /docker-entrypoint.sh
