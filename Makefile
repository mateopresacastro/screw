dev:
	ENV=dev concurrently --names "client,server" --prefix-colors "green,blue" "cd client && npm run dev" "cd server && air"

install:
	cd client && npm i && cd ../server && go mod download && air init

build:
	docker build -t local_build .

start:
	docker run -ti -p 3000:3000 local_build