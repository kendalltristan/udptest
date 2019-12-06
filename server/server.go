package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

var (
	bindInterface string
	bindPort int
	logPath string

    Info *log.Logger
    Error *log.Logger
    Critical *log.Logger
)


/**
 * Log related functions.
 */

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


/**
 * SourceCounter
 */

// SourceCounter keeps track of the number of received messages from each
// source until the next Report (at which point, the counters are cleared)
type SourceCounter struct {
	list map[string]*int64
	mu   sync.Mutex
}


// NewSourceCounter returns a new instantiated SourceCounter
func NewSourceCounter() *SourceCounter {
	return &SourceCounter{
		list: make(map[string]*int64),
	}
}


func (s *SourceCounter) Add(src string) {
	cnt, ok := s.list[src]
	if !ok {
		s.mu.Lock()
		cnt = new(int64)
		s.list[src] = cnt
		s.mu.Unlock()
	}
	*cnt = *cnt + 1
}


func (s *SourceCounter) Report() {
	s.mu.Lock()

	for ip, count := range s.list {
		Info.Printf("%s: %d", ip, *count)
	}

	s.list = make(map[string]*int64)
	s.mu.Unlock()
}


/**
 * Main/init functions.
 */

func init() {
	flag.StringVar(&bindInterface, "i", "::", "i is the interface IP to which to bind")
	flag.IntVar(&bindPort, "p", 10100, "p is the UDP port to which to bind")
	flag.StringVar(&logPath, "log", "/var/log/udptest.log", "The file where output/errors will be logged.")
}


func main() {
	var err error

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

	addr := net.UDPAddr{Port: bindPort, IP: net.ParseIP(bindInterface)}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		Critical.Fatal(err)
	}

	b := make([]byte, 2048)

	sc := NewSourceCounter()
	go reporter(sc)
	go inputReporter(sc)

	for {
		cc, remote, err := conn.ReadFromUDP(b)
		if err != nil {
			Error.Printf("net.ReadFromUDP() error: %s\n", err)
		}

		sc.Add(remote.String())

		_, err = conn.WriteTo(b[0:cc], remote)
		if err != nil {
			Error.Printf("net.WriteTo() error: %s\n", err)
		}
	}
}


/**
 * Utilities
 */

func reporter(sc *SourceCounter) {
	t := time.NewTicker(5 * time.Minute)

	for {
		<-t.C
		sc.Report()
	}
}


func inputReporter(sc *SourceCounter) {
	r := bufio.NewReader(os.Stdin)
	for {
		_, _ = r.ReadString('\n')
		sc.Report()
	}
}
