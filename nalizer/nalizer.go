package nalizer

import (
	"bytes"
)

// Nalizer defines the state for an object which extracts NALUs from bytestreams
type Nalizer struct {
	buffer      bytes.Buffer
	NALTypeLong bool
}

// NAL defines a Network Access Layer Unit from the H.264 spec.
type NAL struct {
	// The raw bytes that represent the whole NALU
	Body []byte
	// The number of frames we found in this NALU
	FrameCount int
}

func (n *Nalizer) Nalize(input []byte) []NAL {
	result := make([]NAL, 0)
	// Append new input onto buffer
	n.buffer.Write(input)
	buffer := n.buffer.Bytes()
	bufferSize := len(buffer)

	// Keep track of where we find NALUs
	starts := make([]int, 0)

	// Look through those bytes for the start codes, store them
	for i, _ := range n.buffer.Bytes() {
		if i+4 > bufferSize-1 {
			break
		}
		if n.NALTypeLong {
			if buffer[i] == 0x00 && buffer[i+1] == 0x00 && buffer[i+2] == 0x00 && buffer[i+3] == 0x01 {
				starts = append(starts, i)
			}
		} else {
			if buffer[i] == 0x00 && buffer[i+1] == 0x00 && buffer[i+2] == 0x01 {
				starts = append(starts, i)
			}
		}
	}
	// If we don't have at least two start codes, we may only have part of a NALU
	if len(starts) < 2 {
		return result
	}
	// Look through our found start codes and find start/end indices of NALUs
	for i, _ := range starts {
		// If we're not on the last one
		if i+1 < len(starts) {
			// One NALU is the space between the starts
			nalLength := starts[i+1] - starts[i]
			// Copy those bytes out of our buffer so we can clean it up later
			nal := make([]byte, nalLength)
			copy(nal, n.buffer.Bytes()[starts[i]:starts[i+1]-1])
			// Try to estimate frames by detecting the presence of an Access Unit Delimiter: https://yumichan.net/video-processing/video-compression/introduction-to-h264-nal-unit/
			// TODO: I think there's a better way to do this - this results in a weird rollercoaster speed up and slow down result.
			frameCount := 0
			if n.NALTypeLong && nal[4]&0x1f == 9 {
				frameCount = 1
			}
			if !n.NALTypeLong && nal[3]&0x1f == 9 {
				frameCount = 1
			}
			// Put our newly found NALU in the list
			result = append(result, NAL{Body: nal, FrameCount: frameCount})
		}
	}
	// Dispose of the buffer where we've found NALUs, leaving only the leftovers
	n.buffer.Next(starts[len(starts)-1])
	return result
}
