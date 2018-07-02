FROM busybox as busybox
run echo "sleeping for 10 seconds"
RUN sleep 10
RUN ls -l /src/
RUN find /src/ -maxdepth 2
RUN du -sh /src/
RUN du -sh /src/*

FROM node as jsbuilder
WORKDIR /workspace
COPY app /workspace
RUN cd /workspace && npm run build


FROM golang:1.9.4-alpine3.7 as builder
WORKDIR /go/src/github.com/gnur/quice/
RUN apk add --no-cache git
RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
COPY --from=jsbuilder /workspace/dist app/dist

RUN go-bindata-assetfs -prefix app app/dist/...
COPY vendor vendor
COPY config config
COPY memdb memdb
COPY main.go .
RUN go build -o quice *.go

FROM alpine:latest  
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
CMD ["./quice"]
EXPOSE 8624
COPY --from=builder /go/src/github.com/gnur/quice/quice /
