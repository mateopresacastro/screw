.PHONY: dev prod down clean

dev:
	docker compose -f compose.dev.yaml up --build

prod:
	docker compose -f compose.yaml up --build -d

down:
	docker compose -f compose.yaml down
	docker compose -f compose.dev.yaml down

clean:
	docker compose -f compose.yaml down -v
	docker compose -f compose.dev.yaml down -v
	docker images -q screw* | xargs -r docker rmi