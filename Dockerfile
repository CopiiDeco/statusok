# Do not use this Dockerfile.This is not ready yet.

FROM askmike/golang-raspbian

RUN apt-get update && apt-get install -y --no-install-recommends git && apt-get clean

ENV GOPATH /gopath
WORKDIR /gopath/src/github.com/dlaize/statusok

RUN mkdir -p /gopath/src/github.com/dlaize/statusok
ADD . /gopath/src/github.com/dlaize/statusok

RUN cd /gopath/src/github.com/dlaize/statusok
RUN go get github.com/urfave/cli
RUN go get github.com/Sirupsen/logrus
RUN go get github.com/influxdata/influxdb/client/v2
RUN go get github.com/mailgun/mailgun-go
RUN go build -o goapp

ENTRYPOINT ./goapp

VOLUME ["/usr/local/config.json"]

EXPOSE 7321