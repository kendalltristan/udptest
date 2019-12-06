package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	name string
	port int
	sentCount int = 0;
	receivedCount int = 0;
	logPath string

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
	flag.StringVar(&name, "host", "", "Host is the echo server to which we should connect.")
	flag.IntVar(&port, "port", 10100, "Port defines the UDP port to which we should connect.")
	flag.StringVar(&logPath, "log", "/var/log/udptest.log", "The file where output/errors will be logged.")
}


//
func receiver(ctx context.Context, conn io.Reader) {
	var cc int
	var err error
	c := make([]byte, 40)

	for ctx.Err() == nil {
		cc, err = conn.Read(c)
		if err != nil {
			Error.Printf("conn.Read() error: %s\n", err)
			continue
		}
		if cc != 37 {
			Error.Printf("ERROR: wrong bytes read: %d != %d", cc, 37)
		} else {
			receivedCount++
		}
	}
}


//
func cleanup(conn io.Reader) {

}


// And here we go...
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	flag.Parse()

	// Open the log file, redirect Stderr to it, and initialize our loggers.
    logFile, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
        fmt.Println("ERROR: Unable to open log file. Exiting.")
        os.Exit(1)
    }
    defer logFile.Close()
    redirectStderr(logFile)
    logInit(logFile)

	// Die if the host is not specified.
	if name == "" {
		Critical.Fatalln("host is a required parameter")
	}

	// Catch keyboard interrupts.
	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
		cancel()
        Info.Println("Total sent messages:", sentCount)
		Info.Println("Total received messages:", receivedCount)
        os.Exit(1)
    }()

	// Assemble the name and port, then open a UDP connection.
	nameport := name + ":" + strconv.Itoa(port)
	conn, err := net.Dial("udp", nameport)
	if err != nil {
		log.Fatal(err)
	}

	// Log the local and remote addresses.
	Info.Printf("Local address: %v\n", conn.LocalAddr())
	Info.Printf("Remote address: %v\n", conn.RemoteAddr())

	// Create a byte, launch the receiver, and sleep for a second.
	b := []byte("abcdefghijklmnopqrstuvwxyz0123456789\n")
	go receiver(ctx, conn)
	time.Sleep(time.Second)

	//
	for {
		sentCount++
		_, err = conn.Write(b)
		if err != nil {
			Error.Printf("conn.Write() error: %s\n", err)
		}
		time.Sleep(6 * time.Second)
	}

	// In case we somehow get broken out of the loop.
	cancel()
	Info.Println("Total sent messages:", sentCount)
	Info.Println("Total received messages:", receivedCount)
	if err := conn.Close(); err != nil {
		time.Sleep(time.Second)
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	os.Exit(0)
}
