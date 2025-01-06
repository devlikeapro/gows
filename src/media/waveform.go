package media

import (
	"bytes"
	"fmt"
	"gopkg.in/hraban/opus.v2"
	"io"
	"math"
	"mccoy.space/g/ogg"
)

func extractOpusFromOgg(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	decoder := ogg.NewDecoder(reader)

	var opusData []byte

	// Iterate through Ogg pages
	for {
		page, err := decoder.Decode()
		if err != nil {
			if err == io.EOF {
				// End of file
				break
			}
			return nil, fmt.Errorf("error reading Ogg page: %v", err)
		}

		// Append each packet's data to opusData
		for _, packet := range page.Packets {
			opusData = append(opusData, packet...)
		}
	}
	return opusData, nil
}
func readStreamPcm(stream *opus.Stream, buffersize int) ([]int16, error) {
	var pcm []int16
	pcmbuf := make([]int16, buffersize)
	for {
		n, err := stream.Read(pcmbuf)
		switch err {
		case io.EOF:
			return pcm, nil
		case nil:
			break
		default:
			return nil, fmt.Errorf("error while decoding opus file: %v", err)
		}
		if n == 0 {
			return nil, fmt.Errorf("nil-error Read() must not return 0")
		}
		pcm = append(pcm, pcmbuf[:n]...)
	}
}

// Waveform generates a waveform from the audio content
// 64 number from 0 to 100
func Waveform(content []byte) ([]byte, error) {
	const (
		sampleRate     = 48000 // Opus standard
		channels       = 1     // Mono
		waveformPoints = 64    // Number of points in the waveform
	)
	r := bytes.NewReader(content)
	stream, err := opus.NewStream(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus stream: %v", err)
	}
	pcm, err := readStreamPcm(stream, sampleRate*channels)
	if err != nil {
		return nil, fmt.Errorf("failed to read PCM data: %v", err)
	}

	n := len(pcm)
	// Calculate the RMS values for segments of the PCM data
	segmentSize := n / waveformPoints
	wf := make([]byte, waveformPoints)
	for i := 0; i < waveformPoints; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if end > n {
			end = n
		}

		// Calculate RMS for the segment
		var sum float64
		for j := start; j < end; j++ {
			sum += float64(pcm[j] * pcm[j])
		}
		rms := math.Sqrt(sum / float64(segmentSize))

		// Normalize RMS to range 0-100
		normalized := byte((rms / math.MaxInt16) * 100)
		if normalized > 100 {
			normalized = 100
		}
		wf[i] = normalized
	}
	return wf, nil
}
