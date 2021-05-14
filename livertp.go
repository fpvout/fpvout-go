package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/google/gousb"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"

	"nalizer"
)

var MAGIC = []byte{0x52, 0x4d, 0x56, 0x54}

var httpPort int

var ip string
var port int
var bufferSize int
var mtu int
var packetType int
var clockRate uint
var frameRate uint

var usbVid int
var usbPid int

func init() {
	// HTTP (for SDP)
	flag.IntVar(&httpPort, "http_port", 8080, "port on which to serve SDP")

	// RTP
	flag.IntVar(&port, "port", 16384, "destination port for the RTP stream")
	flag.StringVar(&ip, "ip", "224.0.190.128", "destination ip address for the RTP stream (can be multicast)")
	flag.IntVar(&bufferSize, "buffer", 2048, "size of USB read buffer")
	flag.IntVar(&mtu, "mtu", 1400, "max packet size over the transport")
	flag.IntVar(&packetType, "type", 96, "RTP packet type field (must be <= 127)")
	flag.UintVar(&clockRate, "clock", 90000, "RTP clock rate")

	// USB
	flag.IntVar(&usbVid, "usb_vid", 0x2ca3, "USB Vendor ID")
	flag.IntVar(&usbPid, "usb_pid", 0x1f, "USB Product ID")
	flag.UintVar(&frameRate, "usb_framerate", 60, "Goggle exported framerate")

	flag.Parse()
}

func main() {
	// Initialize UDP output
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Fatalf("udp: %v", err)
	}
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("udp: %v", err)
	}

	// Create our packetizer
	p := rtp.NewPacketizer(mtu, uint8(packetType), 0xDFDF1000 /* arbitrary source ID */, &codecs.H264Payloader{}, rtp.NewRandomSequencer(), uint32(clockRate))

	// Setup USB
	ctx := gousb.NewContext()
	defer ctx.Close()

	var dev *gousb.Device
	err = errors.New("no error")
	dev, err = ctx.OpenDeviceWithVIDPID(gousb.ID(usbVid), gousb.ID(usbPid))
	if err != nil {
		log.Fatalf("Error opening device: %v", err)
	}
	for dev == nil {
		dev, err = ctx.OpenDeviceWithVIDPID(gousb.ID(usbVid), gousb.ID(usbPid))
		if err != nil {
			log.Fatalf("Error opening device: %v", err)
		}
		log.Printf("Waiting for device...")
		time.Sleep(5 * time.Second)
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
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	toGoggles.WriteContext(ctxTimeout, MAGIC)
	rs, err := fromGoggles.NewStream(fromGoggles.Desc.MaxPacketSize, 5 /* 5 read transactions may be in-flight at any time */)
	if err != nil {
		log.Fatalf("NewStream: %v", err)
	}

	go reader(rs, p, c)
	http.HandleFunc("/stream.sdp", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("v=0\n"))
		w.Write([]byte(fmt.Sprintf("o=fpv 0 0 IN IP4 %s\n", ip)))
		w.Write([]byte("s=FPV Feed\n"))
		w.Write([]byte(fmt.Sprintf("c=IN IP4 %s\n", ip)))
		w.Write([]byte("t=0 0\n"))
		w.Write([]byte(fmt.Sprintf("m=video %d RTP/AVP 96\n", port)))
		w.Write([]byte(fmt.Sprintf("a=rtpmap:96 H264/%d\n", clockRate)))
	})
	printAllAddresses()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil))

}

// Print to stdout all the ways one can retrieve the SDP file from our embedded webserver
func printAllAddresses() {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Error enumerating interfaces: %v", err)
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Fatalf("Error enumerating addresses: %v", err)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				log.Printf("http://%s:%d/stream.sdp", ip.String(), httpPort)
			}
		}
	}
}

// Take in our packets over USB, NALize them, packetize them, transmit them out the io.Writer
func reader(in io.Reader, p rtp.Packetizer, w io.Writer) {

	nz := nalizer.Nalizer{NALTypeLong: false}

	packetsWritten := 0
	lastPacketReport := 0

	b := make([]byte, bufferSize)
	for {
		n, err := in.Read(b)
		if err == io.EOF {
			fmt.Println("EOF")
			break
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		if n > 0 {
			// Create NALS
			nals := nz.Nalize(b[:n])
			for _, nal := range nals {
				// To update our timestamp, we need to understand how many samples a given RTP packet contains
				// Our best guess, given that we don't have a display timestamp from the export, is to estimate this
				// using the clock rate and the framerate, both of which are per second, then only using that if
				// the NALU contained a frame.
				var sampleCount uint32 = uint32(clockRate/frameRate) * uint32(nal.FrameCount)
				packets := p.Packetize(nal.Body, sampleCount)
				for _, packet := range packets {
					packetsWritten++
					pBytes, err := packet.Marshal()
					if err != nil {
						log.Printf("packet: %v", err)
					}
					w.Write(pBytes)
				}
			}
			if packetsWritten > lastPacketReport+1000 {
				lastPacketReport = packetsWritten
				log.Printf("wrote %d packets\nbuffer: %d bytes", packetsWritten, len(nal.Body))
			}
		}
	}
}
