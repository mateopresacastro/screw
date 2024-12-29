dev:
	ENV=dev concurrently \
		--names "frontend,api" \
		--prefix-colors "green,blue" \
		"cd frontend && npx next lint --fix && npm run dev" "cd api && air"

install: compile
	cd frontend && npm i && cd ../api && go mod download && air init

build:
	docker build -t tagger .

start: build
	docker run -ti -p 3000:3000 tagger