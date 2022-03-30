package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"

	"github.com/songgao/water"
)

type Tun struct {
	iface     *water.Interface
	ifacename string
	addr      string
	server    bool
}

type Option func(tun *Tun)

func WithCIDRAddr(addr string) Option {
	return func(tun *Tun) {
		tun.addr = addr
	}
}

func WithName(name string) Option {
	return func(tun *Tun) {
		tun.ifacename = name
	}
}

func WithServer(server bool) Option {
	return func(tun *Tun) {
		tun.server = server
	}
}

func NewTunInterface(opts ...Option) (*Tun, error) {
	var tun = new(Tun)

	for _, opt := range opts {
		opt(tun)
	}

	if tun.addr == "" {
		return nil, errors.New("tunAddr nil")
	}

	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: tun.ifacename,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create tun failed, err: %w", err)
	}

	tun.iface = ifce

	cmd := exec.Command("ip", "addr", "add", tun.addr, "dev", tun.ifacename)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ip set failed, err: %s", out)
	}

	cmd = exec.Command("ip", "link", "set", "dev", tun.ifacename, "up")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ip set failed, err: %s", out)
	}

	return tun, nil
}

func (tun *Tun) Read() chan *Conn {
	var dataChan = make(chan *Conn, 100)

	go func() {
		for {
			var tunBuf [4096]byte

			n, err := tun.iface.Read(tunBuf[:])
			if err != nil {
				log.Printf("tun read failed, err:%s", err)
			}

			conn := &Conn{
				data: append([]byte{}, tunBuf[:n]...),
			}
			dataChan <- conn
		}
	}()

	return dataChan
}

func (tun *Tun) Write(data <-chan *Conn) {
	for d := range data {
		_, err := tun.iface.Write(d.data)
		if err != nil {
			log.Println("tun write, err", err)
		}
	}
}
