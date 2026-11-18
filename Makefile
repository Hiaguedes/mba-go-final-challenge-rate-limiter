.PHONY: run test test-integration stress-test docker-up docker-down

run:
	go run cmd/server/main.go

# Testes unitários. Os testes de integração com Redis são pulados se REDIS_ADDR não estiver definido.
test:
	go test -count=1 -v ./...

# Sobe o Redis via Docker e roda todos os testes, incluindo os de integração.
test-integration: docker-redis
	REDIS_ADDR=localhost:6379 go test -count=1 -v ./...

docker-redis:
	docker compose up -d redis

stress-test:
	docker run -it --rm stress-test -u http://host.docker.internal:8080 -r 300 -c 15 -H "API_KEY: token_de_acesso"

docker-up:
	docker compose up -d

docker-down:
	docker compose down
