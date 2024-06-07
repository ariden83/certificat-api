default: build-app
build: build-app

build-app:
	go build -o bin/p2pnetwork -v .

run: build-app
	./bin/p2pnetwork .

install:
	sudo go install ./
