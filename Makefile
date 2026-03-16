.PHONY: build run test clean

build:
	go build -o bin/distributed-task-scheduler .

run: build
	./bin/distributed-task-scheduler

test:
	go test ./...

clean:
	rm -rf bin/
