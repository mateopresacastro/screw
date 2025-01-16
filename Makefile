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
	@echo "Pushing images to GitHub Container Registry..."
	docker build -t ghcr.io/mateopresacastro/screw/api:latest ./api
	docker build -t ghcr.io/mateopresacastro/screw/nextjs:latest ./nextjs
	docker build -t ghcr.io/mateopresacastro/screw/proxy:latest ./nginx
	docker push ghcr.io/mateopresacastro/screw/api:latest
	docker push ghcr.io/mateopresacastro/screw/nextjs:latest
	docker push ghcr.io/mateopresacastro/screw/proxy:latest
	@echo "Images successfully pushed to GHCR."

test:
	cd api && go test ./...
