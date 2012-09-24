chatserver: chatserver.go
	go build chatserver.go
	
clean:
	rm -f chatserver

run: chatserver
	./chatserver

format:
	gofmt -w=true -tabs=false -tabwidth=2 chatserver.go

