FROM golang:1.9

EXPOSE 8080
WORKDIR $GOPATH/src/github.com/Azure/mirrorcat

ADD . $GOPATH/src/github.com/Azure/mirrorcat
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure
RUN go install ./mirrorcat/

ENTRYPOINT ["mirrorcat", "start"]