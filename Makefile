start:
	ENV=dev concurrently --names "client,server" --prefix-colors "green,blue" "cd client && npm run dev" "cd server && air"

install:
	cd client && npm i && cd ../server && go mod download && air init