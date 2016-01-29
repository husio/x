all: docker clean

bin:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o paste cmd/paste/paste.go
	#go build -o paste cmd/paste/paste.go

docker: bin
	docker build -t paste -f Dockerfile .
	rm paste 2> /dev/null

run: docker
	docker-compose up

clean:
	rm paste 2> /dev/null

.PHONY: bin docker all clean
