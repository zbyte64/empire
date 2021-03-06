FROM golang:1.7.0
MAINTAINER Eric Holmes <eric@remind101.com>

LABEL version 0.11.0

ADD . /go/src/github.com/remind101/empire
WORKDIR /go/src/github.com/remind101/empire
RUN go install ./cmd/empire

ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080
