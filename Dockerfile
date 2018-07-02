FROM node as jsbuilder
WORKDIR /app
COPY app /app
RUN cd /app && npm run build


FROM golang:1.9.4-alpine3.7 as builder
WORKDIR /go/src/github.com/gnur/quice/
RUN apk add --no-cache git
RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
COPY --from=jsbuilder /app/dist app/dist

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
