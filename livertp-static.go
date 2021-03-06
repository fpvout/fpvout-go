package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/karalabe/gousb/usb"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"

	"nalizer"
)

var MAGIC = []byte{0x52, 0x4d, 0x56, 0x54}

type flagDef struct {
	http struct {
		port int
	}
	rtp struct {
		ip                 string
		port               int
		mtu                int
		packetType         int
		clockRate          uint
		frameRate          uint
		sampleMethodStatic bool
	}
	usb struct {
		vid        int
		pid        int
		bufferSize int
		bufferTxes int
	}
}

var flags flagDef
var COMPILE_VERSION string
var COMPILE_HOSTNAME string
var COMPILE_TIMESTAMP string
var COMPILE_USER string

func init() {
	// HTTP (for SDP)
	flag.IntVar(&flags.http.port, "http_port", 8080, "port on which to serve SDP files")

	// RTP
	flag.UintVar(&flags.rtp.clockRate, "rtp_clockrate", 90000, "RTP clock rate")
	flag.UintVar(&flags.rtp.frameRate, "rtp_framerate", 60, "RTP's assumption of goggle exported framerate")
	flag.IntVar(&flags.rtp.mtu, "rtp_mtu", 1400, "max packet size over the transport")
	flag.StringVar(&flags.rtp.ip, "rtp_ip", "224.0.190.128", "destination ip for the RTP stream (can be multicast)")
	flag.IntVar(&flags.rtp.port, "rtp_port", 16384, "destination port for the RTP stream")
	flag.BoolVar(&flags.rtp.sampleMethodStatic, "rtp_samples_static", false, "if true, RTP timestamp tracks a monotonic clock rather than frame count")
	flag.IntVar(&flags.rtp.packetType, "rtp_type", 96, "RTP packet type field (must be <= 127)")

	// USB
	flag.IntVar(&flags.usb.bufferSize, "usb_buffer_size", 2048, "size of buffer we stage reads into")
	flag.IntVar(&flags.usb.bufferTxes, "usb_buffer_txes", 20, "how many bulk reads we'll prefetch and keep in flight simultaneously")
	flag.IntVar(&flags.usb.pid, "usb_pid", 0x1f, "USB Product ID")
	flag.IntVar(&flags.usb.vid, "usb_vid", 0x2ca3, "USB Vendor ID")

	flag.Parse()
}

func main() {
	log.Printf("[init]: %s version %s (%s) | built at %s by %s@%s\n\n", os.Args[0], COMPILE_VERSION, runtime.Version(), COMPILE_TIMESTAMP, COMPILE_USER, COMPILE_HOSTNAME)
	// Build our important channels
	c := setupUDP()
	rs := setupUSB()
	p := setupRTP()
	setupHTTP()

	// Spin off passing data around into a goroutine
	go reader(rs, p, c)

	log.Println("[sdp]: the following streams are available:")
	printAllAddresses()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", flags.http.port), nil))

}

func setupRTP() rtp.Packetizer {
	return rtp.NewPacketizer(flags.rtp.mtu, uint8(flags.rtp.packetType), 0xDFDF1000, /* arbitrary source ID */
		&codecs.H264Payloader{}, rtp.NewRandomSequencer(), uint32(flags.rtp.clockRate))

}

func setupUDP() io.Writer {
	// Initialize UDP output
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", flags.rtp.ip, flags.rtp.port))
	if err != nil {
		log.Fatalf("udp: %v", err)
	}
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("udp: %v", err)
	}
	return c
}

func setupUSB() io.Reader {
	// Setup USB
	ctx, err := usb.NewContext()
	if err != nil {
		log.Fatalf("usb context: %v", err)
	}
	//defer ctx.Close()

	var dev *usb.Device

	if os.Geteuid() != 0 {
		log.Println("[usb] you may need to run this program as root")
	}

	dev, err = ctx.OpenDeviceWithVidPid(flags.usb.vid, flags.usb.pid)
	if err != nil {
		log.Fatalf("[usb] error opening device: %v", err)
	}
	for dev == nil {
		dev, err = ctx.OpenDeviceWithVidPid(flags.usb.vid, flags.usb.pid)
		if err != nil {
			log.Fatalf("[usb] error opening device: %v", err)
		}
		log.Printf("[usb] waiting for device...")
		time.Sleep(5 * time.Second)
	}
	// Detach kernel drivers
	//dev.SetAutoDetach(true)
	//defer dev.Close()
	err = dev.SetConfig(1)
	if err != nil {
		log.Fatalf("[usb] config(1): %v", err)
	}
	// Open endpoints
	fromGoggles, err := dev.OpenEndpoint(1, 3, 0, 0x84)
	if err != nil {
		log.Fatalf("[usb] endpoint: %v", err)
	}

	toGoggles, err := dev.OpenEndpoint(1, 3, 0, 0x03)
	if err != nil {
		log.Fatalf("[usb] endpoint: %v", err)
	}

	// Write magic
	toGoggles.Write(MAGIC)
	rs := fromGoggles
	return rs
}

func setupHTTP() {
	http.HandleFunc("/stream.sdp", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("v=0\n"))
		w.Write([]byte(fmt.Sprintf("o=fpv 0 0 IN IP4 %s\n", flags.rtp.ip)))
		w.Write([]byte("s=FPV Feed\n"))
		w.Write([]byte(fmt.Sprintf("c=IN IP4 %s\n", flags.rtp.ip)))
		w.Write([]byte("t=0 0\n"))
		w.Write([]byte(fmt.Sprintf("m=video %d RTP/AVP %d\n", flags.rtp.port, flags.rtp.packetType)))
		w.Write([]byte(fmt.Sprintf("a=rtpmap:%d H264/%d\n", flags.rtp.packetType, flags.rtp.clockRate)))
	})
}

// Print to stdout all the ways one can retrieve the SDP file from our embedded webserver
func printAllAddresses() {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("[sdp] error enumerating interfaces: %v", err)
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Fatalf("[sdp] error enumerating addresses: %v", err)
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
				fmt.Printf("\thttp://%s:%d/stream.sdp\n", ip.String(), flags.http.port)
			}
		}
	}
}

// Take in our packets over USB, NALize them, packetize them, transmit them out the io.Writer
func reader(in io.Reader, p rtp.Packetizer, w io.Writer) {

	nz := nalizer.Nalizer{NALTypeLong: false}

	packetsWritten := 0
	lastPacketReport := 0

	lastPacketTimestamp := time.Now()

	b := make([]byte, flags.usb.bufferSize)
	for {
		n, err := in.Read(b)
		if err == io.EOF {
			log.Println("EOF")
			break
		}
		if err != nil {
			log.Println(err)
			continue
		}
		if n > 0 {
			// Create NALS
			nals := nz.Nalize(b[:n])
			for _, nal := range nals {
				// To update our timestamp, we need to understand how many samples a given RTP packet contains
				// given that we don't have a display timestamp from the export, is to estimate this
				// using one of two two methods, switched by a flag
				// TODO: Which is actually better?
				var sampleCount uint32 = 0
				if flags.rtp.sampleMethodStatic {
					// We know how long it's been since the last packet, so increment the timestamp that many samples
					// ignoring the actual frame data
					t := time.Now()
					nsPerSecond := uint32(1 * 1000 * 1000 * 1000)
					nsPerClock := uint32(nsPerSecond / uint32(flags.rtp.clockRate))
					nsElapsed := uint32(t.Sub(lastPacketTimestamp).Nanoseconds())
					sampleCount = nsElapsed / nsPerClock * uint32(nal.FrameCount)
					lastPacketTimestamp = t
				} else {
					// Calculate our best guess, using the clock rate and the framerate,
					// both of which are per second, then only using that if the NALU contained a frame.
					sampleCount = uint32(flags.rtp.clockRate/flags.rtp.frameRate) * uint32(nal.FrameCount)
				}
				packets := p.Packetize(nal.Body, sampleCount)
				for _, packet := range packets {
					packetsWritten++
					pBytes, err := packet.Marshal()
					if err != nil {
						log.Printf("[reader] packet: %v", err)
					}
					w.Write(pBytes)
				}
			}
			if packetsWritten > lastPacketReport+1000 {
				lastPacketReport = packetsWritten
				log.Printf("[reader] wrote %d packets", packetsWritten)
			}
		}
	}
}
