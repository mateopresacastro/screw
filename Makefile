.PHONY: dev prod down clean

dev:
	docker compose -f compose.dev.yaml up --build

prod:
	docker compose -f compose.yaml up --pull always --build -d

down:
	docker compose -f compose.yaml down
	docker compose -f compose.dev.yaml down

clean:
	docker compose -f compose.yaml down -v
	docker compose -f compose.dev.yaml down -v
	docker images -q screw* | xargs -r docker rmi

push:
	docker buildx build --platform linux/amd64 -t ghcr.io/mateopresacastro/screw/api:latest ./api --push
	docker buildx build --platform linux/amd64 -t ghcr.io/mateopresacastro/screw/nextjs:latest ./nextjs --push
	docker buildx build --platform linux/amd64 -t ghcr.io/mateopresacastro/screw/proxy:latest ./nginx --push


test:
	cd api && go test ./...
