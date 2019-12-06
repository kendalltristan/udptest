package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	name string
	port int
	messageCount int

	logPath string = "/var/log/udptest.log"

    Info *log.Logger
    Error *log.Logger
    Critical *log.Logger
)


// Redirect Stderr to a file.
func redirectStderr(f *os.File) {
    err := syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
    if err != nil {
        Critical.Fatalln("Failed to redirect stderr to file:", err)
    }
}


// Initializes our loggers.
func logInit(handle io.Writer) {
    Info = log.New(handle, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
    Error = log.New(handle, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
    Critical = log.New(handle, "CRITICAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}


// Assign name, port, and message count variables.
func init() {
	flag.StringVar(&name, "host", "", "host is the echo server to which we should connect")
	flag.IntVar(&port, "port", 10100, "port defines the UDP port to which we should connect")
	flag.IntVar(&messageCount, "count", 100, "count is the number of datagrams to send")
}


//
func receiver(ctx context.Context, conn io.Reader) {
	var cc int
	var count int
	var err error
	c := make([]byte, 40)

	for ctx.Err() == nil {
		if count == messageCount {
			break
		}
		cc, err = conn.Read(c)
		if err != nil {
			Error.Printf("conn.Read() error: %s\n", err)
			break
		}
		if cc != 37 {
			Error.Printf("ERROR: wrong bytes read: %d != %d", cc, 37)
		} else {
			count++
		}
	}
	Info.Println("total read messages:", count)
}


//
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Open the log file, redirect Stderr to it, and initialize our loggers.
    logFile, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
        fmt.Println("ERROR: Unable to open log file. Exiting.")
        os.Exit(1)
    }
    defer logFile.Close()
    redirectStderr(logFile)
    logInit(logFile)

	// Parse the flags and die if the host is not specified.
	flag.Parse()
	if name == "" {
		Critical.Fatalln("host is a required parameter")
	}

	// Assemble the name and port, then open a UDP connection.
	nameport := name + ":" + strconv.Itoa(port)
	conn, err := net.Dial("udp", nameport)
	if err != nil {
		log.Fatal(err)
	}

	// Log the local and remote addresses.
	Info.Printf("Local address: %v\n", conn.LocalAddr())
	Info.Printf("Remote address: %v\n", conn.RemoteAddr())

	b := []byte("abcdefghijklmnopqrstuvwxyz0123456789\n")

	go receiver(ctx, conn)

	time.Sleep(time.Second)

	for i := 0; i < messageCount; i++ {
		_, err = conn.Write(b)
		if err != nil {
			Error.Printf("conn.Write() error: %s\n", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	time.Sleep(time.Second)
	cancel()

	Info.Println("total sent messages:", messageCount)
	if err = conn.Close(); err != nil {
		time.Sleep(time.Second)
		log.Fatal(err)
	}
	time.Sleep(time.Second)
}
