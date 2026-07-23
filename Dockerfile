FROM alpine:3.21 AS certs
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/actup /actup
ENTRYPOINT ["/actup"]
