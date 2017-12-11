FROM golang:1.9-alpine as builder

WORKDIR /go/src/github.com/Azure/mirrorcat

RUN apk add --update git
RUN go get -u github.com/golang/dep/cmd/dep

ADD . .
RUN dep ensure && \
    cd mirrorcat && \
    go build -ldflags "-X github.com/Azure/mirrorcat/mirrorcat/cmd.commit=$(git rev-parse HEAD)"

FROM alpine
RUN apk add --update git
WORKDIR /root/
COPY --from=builder /go/src/github.com/Azure/mirrorcat/mirrorcat/mirrorcat .
EXPOSE 8080
ENTRYPOINT [ "./mirrorcat", "start" ]
