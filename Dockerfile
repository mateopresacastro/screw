FROM node:22-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package*.json .
RUN npm install
COPY frontend/ .
RUN npm run build

FROM golang:latest AS api
WORKDIR /build/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN GOOS=linux go build -trimpath -ldflags="-s -w" -o app .

FROM debian:stable-slim
RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*
COPY --from=frontend /build/frontend/out /frontend/out
COPY --from=api /build/api/app /usr/bin/app
ENV ENV=prod
ENTRYPOINT ["/usr/bin/app"]
EXPOSE 3000