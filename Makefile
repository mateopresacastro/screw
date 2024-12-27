dev:
	ENV=dev concurrently --names "client,server" --prefix-colors "green,blue" "cd client && npm run dev" "cd server && air"

install:
	cd client && npm i && cd ../server && go mod download && air init

compile:
	protoc server/proto/*.proto \
		--go_out=server/proto/gen \
		--go-grpc_out=server/proto/gen \

build:
	docker build -t tagger .

start: build
	docker run -ti -p 3000:3000 tagger