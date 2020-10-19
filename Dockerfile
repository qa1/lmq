FROM golang:1.15.3-alpine3.12 as builder

COPY . /app

WORKDIR /app

RUN go build

FROM alpine:3.12

COPY --from=builder /app/lmq /app/lmq

WORKDIR /app

VOLUME /data

EXPOSE 3000

CMD ./lmq /data/config.json
