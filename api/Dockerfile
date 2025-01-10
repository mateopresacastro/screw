FROM golang:latest AS builder
WORKDIR /build/api
COPY go.mod go.sum ./
RUN go mod download

FROM builder AS development
WORKDIR /app
RUN apt-get update && apt-get install -y ffmpeg sqlite3 && rm -rf /var/lib/apt/lists/*
RUN go install github.com/air-verse/air@latest
COPY . .
CMD ["air"]

FROM builder AS production
COPY . .
RUN GOOS=linux go build -trimpath -ldflags="-s -w" -o app .

FROM debian:stable-slim
RUN apt-get update && apt-get install -y ffmpeg sqlite3 && rm -rf /var/lib/apt/lists/*
COPY --from=production /build/api/app /usr/bin/app
COPY audio /app/audio
ENV ENV=prod
ENV IR_PATH=/app/audio/ir.wav
RUN mkdir -p /app/data
ENTRYPOINT ["/usr/bin/app"]
EXPOSE 3000