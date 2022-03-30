package main

import (
	"fmt"
	"log"
	"net"

	"github.com/songgao/water/waterutil"
)

type VPN struct {
	conn   *net.UDPConn
	tun    *Tun
	server bool
	client *net.UDPAddr
}

var cache = make(map[string]*net.UDPAddr)

type Conn struct {
	*net.UDPConn
	data       []byte
	//remoteAddr *net.UDPAddr
}

func NewVpn(server bool, tunAddr string, serverAddr string) *VPN {
	var vpn = &VPN{}

	tun, err := NewTunInterface(
		WithName("tun0"),
		WithCIDRAddr(tunAddr),
		WithServer(server),
	)
	if err != nil {
		log.Fatalln(err)
	}

	vpn.tun = tun

	localAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))

	if server {
		conn, err := net.ListenUDP("udp", localAddr)
		if err != nil {
			log.Fatalln(err)
		}

		vpn.conn = conn

		vpn.server = true

		return vpn
	}

	remoteAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", serverAddr, port))
	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		log.Fatalln(err)
	}

	vpn.conn = conn

	return vpn
}

func (vpn *VPN) Dispatch() {
	go vpn.tun.Write(vpn.Read())
	go vpn.Write(vpn.tun.Read())

	sig := make(chan struct{})
	<-sig
}

func (vpn *VPN) Read() <-chan *Conn {
	var dataChan = make(chan *Conn, 100)

	go func() {
		for {
			var innerData [4096]byte

			n, remoteAddr, err := vpn.conn.ReadFromUDP(innerData[:])
			if err != nil {
				log.Fatalln(err)
			}

			// todo
			if n < 20 {
				continue
			}

			srcIp := waterutil.IPv4Source(innerData[:n])
			dstIp := waterutil.IPv4Destination(innerData[:n])

			// todo
			if srcIp.String() == "0.0.0.0" {
				continue
			}

			log.Printf("received Len: %d from %s --> %s\n", n, remoteAddr.String(), vpn.conn.LocalAddr().String())
			log.Printf("\tinner from: %s ---> %s\n", srcIp, dstIp)

			c := &Conn{
				UDPConn: vpn.conn,
				data:    innerData[:n],
			}

			if vpn.server {
				cache[srcIp.String()] = remoteAddr
			}

			dataChan <- c
		}
	}()

	return dataChan
}

func (vpn *VPN) Write(tunChan <-chan *Conn) {
	for d := range tunChan {

		if vpn.server {

			// todo
			remoteAddr, ok := cache["10.53.0.2"]
			if !ok {
				continue
			}

			_, err := vpn.conn.WriteToUDP(d.data, remoteAddr)
			if err != nil {
				log.Println("vpn server write error", err)
			}
			//fmt.Printf("%s ---> %s, len: %d\n", vpn.conn.LocalAddr(), remoteAddr, n)
		} else {
			_, err := vpn.conn.Write(d.data)
			if err != nil {
				log.Println("vpn client write error", err)
			}
			//fmt.Printf("%s ---> %s, len: %d\n", vpn.conn.LocalAddr(), vpn.conn.RemoteAddr(), n)
		}

	}
}
