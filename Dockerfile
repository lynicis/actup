FROM golang:1.26-alpine AS builder
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /actup .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /actup /usr/local/bin/actup
ENTRYPOINT ["actup"]
