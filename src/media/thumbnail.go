package media

import (
	"bytes"
	"fmt"
	"github.com/h2non/bimg"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"io"
	"os"
)

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

func VideoThumbnail(videoData []byte, time string, size struct{ Width, Height int }) ([]byte, error) {
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

	// Construct scale filter
	scaleFilter := fmt.Sprintf("%d:-1", size.Width)
	if size.Height != -1 {
		scaleFilter = fmt.Sprintf("%d:%d", size.Width, size.Height)
	}

	// Run ffmpeg process
	go func() {
		defer outputWriter.Close()
		err := ffmpeg.Input("pipe:", ffmpeg.KwArgs{"ss": time}).
			Filter("scale", ffmpeg.Args{scaleFilter}).
			Output("pipe:1", ffmpeg.KwArgs{
				"vframes": 1,
				"f":       "image2",
			}).
			WithInput(inputReader).
			WithOutput(outputWriter, os.Stdout).
			OverWriteOutput().
			Run()
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

	return buf.Bytes(), nil
}
