package validation

import (
	"fmt"
	"strings"
)

const (
	MaxURLCount    = 20
	MaxURLLength   = 2048
	MaxStringField = 255
)

type URLType int

const (
	ImageURL URLType = iota
	VideoURL
)

func ValidateURL(url string) error {
	if len(url) == 0 {
		return fmt.Errorf("url must not be empty")
	}
	if len(url) > MaxURLLength {
		return fmt.Errorf("url exceeds maximum length of %d characters", MaxURLLength)
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}

func ValidateURLs(urls []string, urlType URLType) error {
	if len(urls) > MaxURLCount {
		return fmt.Errorf("maximum %d %s URLs allowed per request", MaxURLCount, urlTypeNames(urlType))
	}
	seen := make(map[string]bool, len(urls))
	for _, u := range urls {
		if err := ValidateURL(u); err != nil {
			return fmt.Errorf("invalid %s URL %q: %w", urlTypeNames(urlType), u, err)
		}
		if seen[u] {
			return fmt.Errorf("duplicate %s URL: %s", urlTypeNames(urlType), u)
		}
		seen[u] = true
	}
	return nil
}

func urlTypeNames(t URLType) string {
	switch t {
	case ImageURL:
		return "image"
	case VideoURL:
		return "video"
	default:
		return "unknown"
	}
}

func ValidateStringField(value string, name string) error {
	if len(value) == 0 {
		return fmt.Errorf("%s must not be empty", name)
	}
	if len(value) > MaxStringField {
		return fmt.Errorf("%s exceeds maximum length of %d characters", name, MaxStringField)
	}
	return nil
}
