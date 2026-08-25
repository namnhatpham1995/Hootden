# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Migrations are embedded via go:embed, so the final stage needs nothing but
# this static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && \
    adduser -D -H hootden
COPY --from=build /out/server /usr/local/bin/server
USER hootden
EXPOSE 8080
ENTRYPOINT ["server"]
