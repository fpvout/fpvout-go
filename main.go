package main

import (
	"io"
	"log"
	"context"
	"time"
	"net/http"

	"github.com/google/gousb"
)

var fromGoggles gousb.InEndpoint
var toGoggles gousb.OutEndpoint

var MAGIC = []byte{0x52, 0x4d, 0x56, 0x54}

func main() {

	log.Println("go-fpv starting")
	// Setup USB
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(0x2ca3, 0x1f)
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}
	if dev == nil {
		log.Fatalf("couldn't find device")
	}
	defer dev.Close()
	cfg, err := dev.Config(1)
	if err != nil {
		log.Fatalf("Config(1): %v", err)
	}
	intf, err := cfg.Interface(3, 0)
	if err != nil {
		log.Fatalf("%s.Interface(3, 0): %v", cfg, err)
	}
	defer intf.Close()
	// Open endpoints
	fromGoggles, err := intf.InEndpoint(0x84)
	if err != nil {
		log.Fatalf("%s.InEndpoint(1): %v", intf, err)
	}

	toGoggles, err := intf.OutEndpoint(0x03)
	if err != nil {
		log.Fatalf("%s.OutEndpoint(0): %v", intf, err)
	}
	// Write magic
	log.Println("writing magic")
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	toGoggles.WriteContext(ctxTimeout, MAGIC)
	log.Println("writing magic complete")
	rs, err := fromGoggles.NewStream(10*fromGoggles.Desc.MaxPacketSize, 5)
	if err != nil {
		log.Fatalf("NewStream: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		b := make([]byte, 1024 * 512)
		w.WriteHeader(200)
		log.Println("beginning write loop")
		for {
			_, err = rs.Read(b)
			if err != io.EOF {
				w.Write(b)
			}
		}
	})
	log.Println("starting on 1234")
	log.Fatal(http.ListenAndServe(":1234", nil))
}
