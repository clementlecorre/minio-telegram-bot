FROM golang:alpine as builder

ARG VCS_REF="N/A"
COPY . /go/src/github.com/clementlecorre/minio-telegram-bot
WORKDIR /go/src/github.com/clementlecorre/minio-telegram-bot
RUN apk add --no-cache git gcc libc-dev ca-certificates

RUN GO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -tags netgo -installsuffix netgo -ldflags '-w' -ldflags "-X main.version=${VCS_REF}" -o minio-telegram-bot .

FROM scratch
ARG BUILD_DATE="N/A"
ARG VCS_REF="N/A"
ARG VCS_URL="N/A"

LABEL org.label-schema.build-date=$BUILD_DATE \
    org.label-schema.vcs-url=$VCS_URL \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.schema-version="1.0.0-rc1" \
    maintainer="clement@le-corre.eu"

WORKDIR /go/src/github.com/clementlecorre/minio-telegram-bot/
COPY --from=builder /go/src/github.com/clementlecorre/minio-telegram-bot .
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs/
ENTRYPOINT ["./minio-telegram-bot"]
