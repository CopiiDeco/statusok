# Donot use this Dockerfile.This is not ready yet.

# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/dlaize/statusok

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)

# cli is a simple, fast, and fun package for building command line apps in Go
RUN go get github.com/urfave/cli
# Run influxdb with docker
#RUN go get github.com/influxdb/influxdb
#RUN go get github.com/mailgun/mailgun-go
RUN go install github.com/dlaize/statusok

#RUN wget http://influxdb.s3.amazonaws.com/influxdb_0.9.3_amd64.deb
#RUN dpkg -i influxdb_0.9.3_amd64.deb
#RUN /etc/init.d/influxdb start

#RUN wget https://s3-us-west-2.amazonaws.com/grafana-releases/release/grafana_4.3.1_amd64.deb 
#RUN apt-get update
#RUN apt-get install -y adduser libfontconfig
#RUN dpkg -i grafana_4.3.1_amd64.deb
#RUN service grafana-server start

#how to connect to localhost inside ?? http://stackoverflow.com/questions/24319662/from-inside-of-a-docker-container-how-do-i-connect-to-the-localhost-of-the-mach

ENTRYPOINT /go/bin/statusok --config /go/src/github.com/dlaize/statusok/config.json

# Document that the service listens 
#8086 influxdb
#3000 grafana
#7231 default statusOK port
#EXPOSE 80 8083 8086 7321 3000
EXPOSE 7321