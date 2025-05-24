FROM golang:1.21-alpine

WORKDIR /app

# Install dependencies, including librdkafka-dev for Alpine/musl
# and build-base which includes common build tools that might be needed by CGO.
RUN apk add --no-cache tzdata git gcc libc-dev librdkafka-dev build-base

COPY go.mod go.sum ./
# It's a good practice to download modules before copying the rest of the source code
# to leverage Docker's layer caching more effectively.
RUN go mod download

COPY . .

# Build the application with CGO enabled and musl tag for librdkafka
# GOOS=linux is good practice if your build environment isn't Linux
# The -tags musl is crucial for confluent-kafka-go on Alpine
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -tags musl -o avax-extractor ./cmd/main.go

EXPOSE 8081
EXPOSE 8080

CMD ["./avax-extractor"]