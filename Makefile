chatserver: chatserver.go
	go build chatserver.go
	
clean:
	rm -f chatserver

run: chatserver
	./chatserver