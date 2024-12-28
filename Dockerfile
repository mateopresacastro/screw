FROM node:22 AS frontend

WORKDIR /build/frontend
COPY frontend/package*.json .
RUN npm install
COPY frontend/ .
RUN npm run build

FROM golang:1.23 AS server
WORKDIR /build/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o app .

FROM scratch
COPY --from=frontend /build/frontend/out /frontend/out
COPY --from=server /build/server/app /usr/bin/app
ENV ENV=prod
ENTRYPOINT ["/usr/bin/app"]
EXPOSE 3000
