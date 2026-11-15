.PHONY: run
run:
	go run cmd/server/main.go
stress-test:
	docker run -it --rm stress-test -u http://host.docker.internal:8080 -r 300 -c 15
docker-up:
	 docker compose up -d