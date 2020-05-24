FROM golang:1.14.3-alpine3.11 as builder

COPY . /app

WORKDIR /app

RUN apk add --no-cache git

RUN go get github.com/gin-contrib/gzip github.com/go-sql-driver/mysql

RUN go build ./LMQ.go
RUN go build ./cleanup.go

FROM alpine:3.11

COPY --from=builder /app/LMQ /app/LMQ
COPY --from=builder /app/cleanup /app/cleanup

WORKDIR /app

VOLUME /data

EXPOSE 3000

CMD ./LMQ /data/config.json
