FROM node as jsbuilder
WORKDIR /workspace
COPY app/package.json /workspace
RUN yarn install
COPY app /workspace
RUN yarn build


FROM golang:1.11-alpine as builder
RUN apk update && apk add alpine-sdk git
ENV GO111MODULE=on
WORKDIR /go/src/github.com/gnur/quice/
COPY go.mod go.mod
COPY go.sum go.sum
RUN go get -u github.com/UnnoTed/fileb0x
COPY --from=jsbuilder /workspace/dist app/dist

COPY fileb0x.toml fileb0x.toml
COPY config config
COPY memdb memdb
COPY main.go .
RUN go generate
RUN go build

FROM alpine:latest  
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
CMD ["./quice"]
EXPOSE 8624
COPY --from=builder /go/src/github.com/gnur/quice/quice /
