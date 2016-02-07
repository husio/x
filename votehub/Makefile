all: docker clean

bin:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o votehub cmd/votehub/votehub.go

docker: bin
	docker build -t "votehub:`git rev-list --count master`-`git rev-parse HEAD | cut -c1-4`" -f Dockerfile .
	rm votehub 2> /dev/null

deploy: docker
	docker save votehub | bzip2 | pv | ssh dark 'bunzip2 | docker load'

compose: docker
	docker-compose up

clean:
	rm votehub 2> /dev/null

.PHONY: bin docker all clean
