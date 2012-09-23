package main

import (
	"fmt"
	"net"
	"runtime"
)

type Connection struct {
	id					uint32					// ID of this connection
	connection	net.Conn				// TCP connection
	output			chan string			// Channel for sending output to client
	open				bool						// Is connection open?
	server			*Server					// Back-reference to server
}

const (
	kNewConnection			= iota
	kConnectionClosed		= iota
)

type ConnectionEvent struct {
	eventType		int
	connection	*Connection
}

type ConnectionMessage struct {
	message			string
	connection	*Connection
}

type Server struct {
	
	host								string
	port								string
	
	// Channel for receiving connection events (new client, client disconnected)
	events							chan ConnectionEvent
	
	// Channel for receiving messages from clients
	// (this and previous could possibly be rolled into a single channel?)
	messages						chan ConnectionMessage
	
	// Map of active connections, connection_id => connection
	connections					map[uint32]*Connection

}

func (server *Server)init(host string, port string) {
	server.host = host
	server.port = port
	server.events = make(chan ConnectionEvent, 128)
	server.messages = make(chan ConnectionMessage, 128)
	server.connections = make(map[uint32]*Connection)
}

func (server *Server)remote() string {
	return server.host + ":" + server.port
}

func (server *Server)listen() {
	
	listener, error := net.Listen("tcp", server.remote())
	if error != nil {
		fmt.Printf("Error creating listener: %s\n", error)
		return
	}
	
	var nextID uint32 = 0
	for {
		conn, error := listener.Accept()
		
		if error != nil {
			fmt.Println("error while accepting connection")
		} else {
			
			nextID++
			
			connection 					 	:= new(Connection)
			connection.id					= nextID
			connection.connection = conn
			connection.output			= make(chan string)
			connection.open				= true
			connection.server			= server
			
			fmt.Printf("New connection accepted, ID = %d\n", connection.id)
			server.events <- ConnectionEvent{eventType: kNewConnection, connection: connection}
			
		}
	}
}

func (server *Server)mainHandler() {
	for {
		select {
			
			// channel event
			// client has either connected or disconnected
			// just handle appropriately...
			case evt := <-server.events:
				conn := evt.connection
				switch (evt.eventType) {
					
					case kNewConnection:
						server.connections[conn.id] = conn
						go conn.handleInput()
						go conn.handleOutput()
					
					case kConnectionClosed:
						delete(server.connections, conn.id)
					
					default:
				}
				
			// message has come in from a client
			// relay message to all clients except source
			case msg := <-server.messages:
				srcConn := msg.connection
				for id, conn := range(server.connections) {
					if id != srcConn.id {
						conn.output <- msg.message
					}
				}
			
			default:
				runtime.Gosched()
		
		}
	}
}

//
// Connection

func (conn *Connection)close() {
	if (conn.open) {
		conn.server.events <- ConnectionEvent{eventType: kConnectionClosed, connection: conn}
		conn.connection.Close()
		close(conn.output)
		conn.open = false
	}
}

func (conn *Connection)handleInput() {
	buffer := make([]byte, 1024)
	// my understanding is that this loop format is safe because
	// if the call to connection.Read() blocks the Go scheduler
	// will be invoked
	for {
		read, error := conn.connection.Read(buffer)
		if error != nil {
			break
		} else {
			conn.server.messages <- ConnectionMessage{connection: conn, message: string(buffer[0:read])}
		}
	}
	conn.close()
}

func (conn *Connection)handleOutput() {
	buffer := make([]byte, 1024)
	// Ref: http://stackoverflow.com/questions/12413510/why-is-this-golang-code-blocking
	// using range ensures we yield to the scheduler correctly and that channel close 
	// is handled correctly
	for message := range conn.output {
		bytesWritten := copy(buffer, message)
		_, error := conn.connection.Write(buffer[0:bytesWritten])
		if (error != nil) {
			fmt.Printf("error writing to connection ID %d\n", conn.id)
			break
		}
	}
	conn.close()
}

func main() {
	
	server := Server{}
	server.init("localhost", "3000")
	
	fmt.Println("Listening on: " + server.remote())
	
	go server.mainHandler()
	go server.listen()
	
	// hack to make program wait indefinitely
	// wg := new(sync.WaitGroup)
	// wg.Add(1)
	// wg.Wait()
	
	// Alternatively,
	select {}
	
}