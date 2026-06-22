# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/cms ./cmd/cms && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/discovery ./cmd/discovery

# --- runtime stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/cms /out/discovery /app/
USER app
EXPOSE 8080 8081
# The binary to run + config path are provided per service in docker-compose.
CMD ["/app/cms", "-config", "/app/config.yaml"]
