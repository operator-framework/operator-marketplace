FROM alpine:3.6

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN adduser -D marketplace-operator
USER marketplace-operator

ADD tmp/_output/bin/marketplace-operator /usr/local/bin/marketplace-operator
