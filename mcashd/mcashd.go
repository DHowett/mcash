package main

import (
	"bytes"
	"flag"
	"io"
	"launchpad.net/goyaml"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Executable struct {
	Path string
	Args []string ",flow"
}

type Config struct {
	Java Executable
	Jar  Executable
	Env  map[string]string
}

func loadConfig(filename string) (config *Config, err error) {
	file, err := os.Open(filename)
	config = nil
	if err != nil {
		return
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, file)
	if err != nil {
		return
	}

	file.Close()

	err = goyaml.Unmarshal(buf.Bytes(), &config)
	return
}

func main() {
	sockdirPtr := flag.String("sockdir", ".", "socket directory")
	tagPtr := flag.String("tag", "minecraft", "socket name")
	cfgPtr := flag.String("config", "mcash.yml", "config filename")
	flag.Parse()

	var activeConfiguration *Config
	activeConfiguration, err := loadConfig(*cfgPtr)
	if err != nil {
		log.Fatalln("Error loading configuration " + *cfgPtr + ": " + err.Error())
	}

	stdinPath := filepath.Join(*sockdirPtr, "_"+*tagPtr+"_stdin.fifo")
	stdoutPath := filepath.Join(*sockdirPtr, "_"+*tagPtr+"_stdout.fifo")
	syscall.Unlink(stdinPath)
	syscall.Unlink(stdoutPath)
	err = syscall.Mkfifo(stdinPath, 0770)
	if err != nil {
		log.Fatalln("Error making fifo " + stdinPath + ": " + err.Error())
	}

	err = syscall.Mkfifo(stdoutPath, 0770)
	if err != nil {
		log.Fatalln("Error making fifo " + stdoutPath + ": " + err.Error())
	}

	log.Println("Opening " + stdinPath + " O_RDONLY")
	stdinFifoFd, err := syscall.Open(stdinPath, syscall.O_RDWR, 0)
	if err != nil {
		log.Fatalln("Error opening " + stdinPath + ": " + err.Error())
	}

	log.Println("Opening " + stdoutPath + " O_WRONLY")
	stdoutFifoFd, err := syscall.Open(stdoutPath, syscall.O_RDWR, 0)
	if err != nil {
		log.Fatalln("Error opening " + stdoutPath + ": " + err.Error())
	}

	syscall.Dup2(int(stdinFifoFd), syscall.Stdin)
	syscall.Dup2(int(stdoutFifoFd), syscall.Stdout)
	syscall.Dup2(int(stdoutFifoFd), syscall.Stderr)

	syscall.Close(stdinFifoFd)
	syscall.Close(stdoutFifoFd)

	var args []string
	java, err := exec.LookPath(activeConfiguration.Java.Path)
	if err != nil {
		log.Fatalln("java (" + java + ") not found.")
	}

	args = append([]string{java}, activeConfiguration.Java.Args...)
	args = append(args, "-jar", activeConfiguration.Jar.Path)
	args = append(args, activeConfiguration.Jar.Args...)

	env := make([]string, len(activeConfiguration.Env))
	i := 0
	for k, v := range activeConfiguration.Env {
		env[i] = k + "=" + v
		i++
	}
	syscall.Exec(args[0], args, env)
}
