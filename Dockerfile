FROM node:22 AS frontend

WORKDIR /build/frontend
COPY frontend/package*.json .
RUN npm install
COPY frontend/ .
RUN npm run build

FROM golang:1.23 AS api
WORKDIR /build/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o app .

FROM scratch
COPY --from=frontend /build/frontend/out /frontend/out
COPY --from=api /build/api/app /usr/bin/app
ENV ENV=prod
ENTRYPOINT ["/usr/bin/app"]
EXPOSE 3000
