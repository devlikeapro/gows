package media

import (
	"github.com/h2non/bimg"
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
