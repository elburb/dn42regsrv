FROM alpine:latest
WORKDIR /app
RUN apk add git
COPY dn42regsrv /app/
COPY StaticRoot /app/StaticRoot
ENTRYPOINT ["/app/dn42regsrv"]
