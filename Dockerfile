FROM node:22 AS client

WORKDIR /build/client
COPY client/package*.json .
RUN npm install
COPY client/ .
RUN npm run build

FROM golang:1.23 AS server
WORKDIR /build/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o app .

FROM scratch
COPY --from=client /build/client/dist /client/dist
COPY --from=server /build/server/app /usr/bin/app
ENV ENV=prod
ENTRYPOINT ["/usr/bin/app"]
EXPOSE 3000
