package controller

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/image/webp"
)

const (
	managedWebPDataPrefix = "data:image/webp;base64,"
	maxManagedWebPPixels  = 4096 * 4096
)

func validateManagedWebPDataURL(value string, maxBytes int) error {
	if !strings.HasPrefix(value, managedWebPDataPrefix) {
		return errors.New("必须是 WebP 格式")
	}

	imageBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, managedWebPDataPrefix))
	if err != nil {
		return errors.New("数据无效")
	}
	if len(imageBytes) == 0 || len(imageBytes) > maxBytes {
		return fmt.Errorf("不能超过 %d KB", maxBytes/1024)
	}

	imageConfig, err := webp.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return errors.New("不是有效的 WebP 图片")
	}
	if imageConfig.Width <= 0 ||
		imageConfig.Height <= 0 ||
		imageConfig.Width > maxManagedWebPPixels/imageConfig.Height {
		return errors.New("尺寸过大")
	}

	return nil
}
