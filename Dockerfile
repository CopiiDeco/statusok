FROM golang:1.12.7 AS builder
WORKDIR /go/src/github.com/CopiiDeco/statusok/
RUN go get google.golang.org/api/option && go get github.com/codegangsta/cli && go get github.com/influxdata/influxdb1-client/v2  && go get cloud.google.com/go/logging && go get github.com/sirupsen/logrus && go get github.com/mailgun/mailgun-go 
COPY statusok.go ./
COPY database ./database/
COPY notify ./notify/
COPY requests ./requests/
RUN env CGO_ENABLED=0 GOOS=linux GOARCH=arm go build  -a -installsuffix cgo 

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/CopiiDeco/statusok/statusok ./
VOLUME /config
COPY ./docker-entrypoint.sh /docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT /docker-entrypoint.sh
