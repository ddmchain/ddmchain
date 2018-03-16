FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.8 https://github.com/ddmchain/go-ddmchain && \
  (cd go-ddmchain && make gddm) && \
  cp go-ddmchain/build/bin/gddm /gddm && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-ddmchain && rm -rf /var/cache/apk/*

EXPOSE 7545
EXPOSE 50303

ENTRYPOINT ["/gddm"]
