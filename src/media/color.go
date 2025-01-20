package media

import (
	"fmt"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"strconv"
	"strings"
)

func ParseColor(color string) (*uint32, error) {
	color = strings.TrimSpace(color)
	color = strings.TrimPrefix(color, "#")

	if len(color) <= 6 {
		color = "FF" + fmt.Sprintf("%06s", color)
	}

	parsed, err := strconv.ParseUint(color, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid color format: %v", err)
	}

	result := uint32(parsed)
	return &result, nil
}

func ParseFont(font uint32) *waE2E.ExtendedTextMessage_FontType {
	result := waE2E.ExtendedTextMessage_FontType(font)
	return &result
}
