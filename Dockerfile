FROM node as jsbuilder
WORKDIR /workspace
COPY app /workspace
RUN npm install
RUN npm run build


FROM golang:1.11-alpine as builder
RUN apk update && apk add alpine-sdk git
ENV GO111MODULE=on
WORKDIR /go/src/github.com/gnur/quice/
COPY go.mod go.mod
COPY go.sum go.sum
COPY --from=jsbuilder /workspace/dist app/dist
RUN go get -u github.com/UnnoTed/fileb0x

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
