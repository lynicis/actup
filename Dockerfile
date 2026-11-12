FROM --platform=$BUILDPLATFORM alpine:3.21 AS certs
RUN apk add --no-cache ca-certificates=20261020-r0

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/actup /actup
USER 65532:65532
HEALTHCHECK CMD ["/actup", "--version"]
ENTRYPOINT ["/actup"]
