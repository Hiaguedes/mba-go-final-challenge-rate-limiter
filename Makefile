.PHONY: run
run:
	go run cmd/server/main.go
stress-test:
	docker run -it --rm stress-test -u http://host.docker.internal:8080 -r 3 -c 2