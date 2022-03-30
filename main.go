package main

import (
	"flag"
	"log"
)

var port int
var tunAddr string
var serverAddr string

func main() {
	flag.StringVar(&serverAddr, "s", "", "server addr")
	flag.StringVar(&tunAddr, "t", "", "tun addr")
	flag.IntVar(&port, "p", 55555, "port to listen on or to connect")
	flag.Parse()

	if tunAddr == "" {
		log.Fatalln("tunAddr", tunAddr)
	}

	NewVpn(serverAddr == "", tunAddr, serverAddr).Dispatch()
}
