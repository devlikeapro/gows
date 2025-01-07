package media

import (
	"bytes"
	"fmt"
	"github.com/h2non/bimg"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"io"
)

// ImageThumbnail generates a thumbnail image from an image.
func ImageThumbnail(image []byte) ([]byte, error) {
	img := bimg.NewImage(image)
	options := bimg.Options{
		Width:  72,
		Height: 72,
		Crop:   true,
	}
	thumbnail, err := img.Process(options)
	if err != nil {
		return nil, err
	}
	return thumbnail, nil
}

// VideoThumbnail generates a thumbnail image from a video at a specific frame.
func VideoThumbnail(videoData []byte, frameNum int, size struct{ Width int }) ([]byte, error) {
	// Create pipes for input and output
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()

	// Write the input video data to the input pipe
	go func() {
		defer inputWriter.Close()
		_, err := inputWriter.Write(videoData)
		if err != nil {
			inputWriter.CloseWithError(err)
		}
	}()

	// Run ffmpeg process
	go func() {
		defer outputWriter.Close()
		cmd := ffmpeg.
			Input("pipe:0").
			Filter("scale", ffmpeg.Args{fmt.Sprintf("%d:-1", size.Width)}).
			Filter("select", ffmpeg.Args{fmt.Sprintf("gte(n,%d)", frameNum)}).
			Output("pipe:", ffmpeg.KwArgs{"vframes": 1, "format": "image2"}).
			WithInput(inputReader).
			WithOutput(outputWriter).
			OverWriteOutput()
		err := cmd.Run()
		if err != nil {
			outputWriter.CloseWithError(err)
		}
	}()

	// Read the output into a buffer
	var buf bytes.Buffer
	_, err := buf.ReadFrom(outputReader)
	if err != nil {
		return nil, err
	}

	data := buf.Bytes()
	if len(data) == 0 {
		return nil, fmt.Errorf("no thumbnail data returned")
	}
	return buf.Bytes(), nil
}
