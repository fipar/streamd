/*
Serves a stream on a tcp port
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	CHANNEL_BUFFER_SIZE = 100
)

var (
	port             int                   // port to listen on 
	fn               string                // file name to read 
	verbose          bool                  // verbose flag
	clients          map[int]chan (string) // every connected client will have a channel here to receive lines from the stream
	lock             *sync.Mutex           // protects access to the clients and connectedClients variables
	connectedClients int                   // count of connected clients
)

func main() {

	flag.IntVar(&port, "port", 1234, "The TCP port to listen on. Defaults to 1234")
	flag.BoolVar(&verbose, "debug", false, "Be verbose")
	flag.StringVar(&fn, "fn", "", "The file or stream to read from. Defaults to stdin")
	flag.Parse()

	connectedClients = 0
	clients = map[int]chan (string){}
	lock = new(sync.Mutex)

	var (
		fd   *os.File
		err  error
		con  net.Conn
		sock net.Listener
	)

	if fn == "" {
		fd = os.Stdin
	} else {
		fd, err = os.Open(fn)
		if err != nil {
			die(err.Error())
		}
	}

	go readFile(fd)

	sock, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		die(err.Error())
	}
	for {
		con, err = sock.Accept()
		if err != nil {
			debug(err.Error())
		}
		go handleConnection(con)
	}
}

// this is the reading goroutine. 
// reads fd until EOF and if there are clients connected, writes the lines to their channels
// TODO: continue reading like tail -f
func readFile(fd *os.File) {
	reader := bufio.NewReader(fd)
	defer fd.Close()
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			return
		}
		lock.Lock() // begin critical section
		if connectedClients > 0 {
			debug(strconv.Itoa(connectedClients) + " connected clients, will send to their channels ")
			for i := 0; i < connectedClients; i++ {
				debug("sending to channel " + strconv.Itoa(i))
				//				s[i] <- line

				go func() {
					select {
					case clients[i] <- line:
					default:
					}
				}()

				debug("sent to channel " + strconv.Itoa(i))
			}
		} else {
			debug("no clients are connected, discarding line")
		}
		lock.Unlock() // end critical section
	}
}

// handles a tcp client
// every new client increments connectedClients and creates a channel, adding it to the clients map
func handleConnection(con net.Conn) {
	defer func() {
		debug("client disconnecting when connectedClients is " + strconv.Itoa(connectedClients))
		if err := recover(); err != nil {
			debug(err.(error).Error())
		}
		lock.Lock()
		connectedClients--
		lock.Unlock()
		con.Close()
	}()

	lock.Lock()
	clients[connectedClients] = make(chan (string), CHANNEL_BUFFER_SIZE)
	id := connectedClients
	debug("client " + strconv.Itoa(id) + " connecting")
	connectedClients++
	lock.Unlock()
	for {
		var line string
		debug("client " + strconv.Itoa(id) + " will read from channel")
		line = <-clients[id]
		debug("client " + strconv.Itoa(id) + " has read from channel")
		con.Write([]byte(line))
	}
}

// so that I can do the same as panic() but without printing the stack trace to the user
func die(message string) {
	fmt.Println(message)
	os.Exit(1)
}

// helper function to print debug messages
func debug(message string) {
	if verbose {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05 MST") + " : " + message)
	}
}
