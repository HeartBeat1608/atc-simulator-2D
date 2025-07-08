.PHONY: run build clean

build: clean
	go build -o bin/atc-sim-client ./cmd/client/main.go

clean: 
	rm -f ./bin/atc-sim-client || true

run: build
	./bin/atc-sim-client
