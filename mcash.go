package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
)

func main() {
	sockdirPtr := flag.String("sockdir", ".", "socket directory")
	tagPtr := flag.String("tag", "minecraft", "socket name")
	flag.Parse()

	stdinPath := filepath.Join(*sockdirPtr, "_"+*tagPtr+"_stdin.fifo")
	stdoutPath := filepath.Join(*sockdirPtr, "_"+*tagPtr+"_stdout.fifo")

	stdinFifo, err := os.OpenFile(stdinPath, os.O_WRONLY, 0)
	if err != nil {
		log.Fatalln("Error opening " + stdinPath + ": " + err.Error())
	}

	stdoutFifo, err := os.OpenFile(stdoutPath, os.O_RDONLY, 0)
	if err != nil {
		log.Fatalln("Error opening " + stdoutPath + ": " + err.Error())
	}

	deathChan := make(chan string)
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)

	log.Println("Connected to " + *tagPtr)

	go func() {
		buf := make([]byte, 512)
		for {
			if n, err := stdoutFifo.Read(buf); err == nil {
				os.Stdout.Write(buf[:n])
			} else {
				deathChan <- "stdout fifo died: " + err.Error()
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 512)
		for {
			if n, err := os.Stdin.Read(buf); err == nil {
				stdinFifo.Write(buf[:n])
			} else {
				deathChan <- "stdin closed"
				break
			}
		}
	}()

	go func() {
		for {
			select {
			case <-sigChan:
				deathChan <- "received signal"
				break
			}
		}
	}()

	reason := <-deathChan
	log.Println("Disconnecting from " + *tagPtr + ": " + reason)
	stdinFifo.Close()
	stdoutFifo.Close()
}
