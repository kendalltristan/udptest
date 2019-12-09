package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"
)

var (
	bindInterface string
	bindPort int
	logPath string

    Info *log.Logger
    Error *log.Logger
    Critical *log.Logger
)


func redirectStderr(f *os.File) {
    err := syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
    if err != nil {
        Critical.Fatalln("Failed to redirect stderr to file:", err)
    }
}


func logInit(handle io.Writer) {
    Info = log.New(handle, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
    Error = log.New(handle, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
    Critical = log.New(handle, "CRITICAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}


func init() {
	flag.StringVar(&bindInterface, "i", "::", "i is the interface IP to which to bind")
	flag.IntVar(&bindPort, "p", 10100, "p is the UDP port to which to bind")
	flag.StringVar(&logPath, "log", "/var/log/udptest.log", "The file where output/errors will be logged.")
}


func main() {
	var err error
	flag.Parse()

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

	for {
		cc, remote, err := conn.ReadFromUDP(b)
		if err != nil {
			Error.Printf("net.ReadFromUDP() error: %s\n", err)
		}

		_, err = conn.WriteTo(b[0:cc], remote)
		if err != nil {
			Error.Printf("net.WriteTo() error: %s\n", err)
		}
	}
}
