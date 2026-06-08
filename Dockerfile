FROM alpine:3.21
ARG TARGETARCH
RUN apk add --no-cache ca-certificates
COPY linux/$TARGETARCH/actup /usr/local/bin/actup
ENTRYPOINT ["actup"]
