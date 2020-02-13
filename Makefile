run: build
	chmod +x ./out/*
	./out/service.admin

build:
	mkdir -p out && cd out && \
	go build -o service.admin ../cmd/app/* && \
	go build -o worker.report ../cmd/reportworker/*
clean:
	rm -rf out
	
tidy:
	go fmt ./...
	go mod tidy