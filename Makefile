dev:
	ENV=dev concurrently \
		--names "frontend,server" \
		--prefix-colors "green,blue" \
		"cd frontend && npm run dev" "cd server && air"

install: compile
	cd frontend && npm i && cd ../server && go mod download && air init

compile:
	protoc *.proto \
		--go_out=server/proto/gen \
		--go-grpc_out=server/proto/gen \
 		--plugin=frontend/node_modules/.bin/protoc-gen-ts_proto \
		--ts_proto_out=frontend/src/proto \
		--ts_proto_opt=esModuleInterop=true,importSuffix=.js,outputClientImpl=grpc-web

build: compile
	docker build -t tagger .

start: build
	docker run -ti -p 3000:3000 tagger