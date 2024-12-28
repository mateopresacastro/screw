dev:
	ENV=dev concurrently \
		--names "frontend,api" \
		--prefix-colors "green,blue" \
		"cd frontend && npm run dev" "cd api && air"

install: compile
	cd frontend && npm i && cd ../api && go mod download && air init

compile:
	protoc *.proto \
		--go_out=api/proto/gen \
		--go-grpc_out=api/proto/gen \
 		--plugin=frontend/node_modules/.bin/protoc-gen-ts_proto \
		--ts_proto_out=frontend/src/proto \
		--ts_proto_opt=esModuleInterop=true,importSuffix=.js,outputClientImpl=grpc-web

build: compile
	docker build -t tagger .

start: build
	docker run -ti -p 3000:3000 tagger