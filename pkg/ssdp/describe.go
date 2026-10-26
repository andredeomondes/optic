package ssdp

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxDescriptionBytes caps a UPnP device description.
const maxDescriptionBytes = 1 << 20

type deviceDescription struct {
	Device struct {
		FriendlyName string `xml:"friendlyName"`
		ModelName    string `xml:"modelName"`
	} `xml:"device"`
}

// ParseDescription reads friendlyName and modelName from a UPnP device
// description document.
func ParseDescription(body []byte) (friendlyName, model string, err error) {
	var d deviceDescription
	if err := xml.Unmarshal(body, &d); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(d.Device.FriendlyName), strings.TrimSpace(d.Device.ModelName), nil
}

// Describe fetches the device description at location and returns the
// device's own name and model. The body is capped at 1 MiB.
func Describe(ctx context.Context, client *http.Client, location string) (friendlyName, model string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("descrição UPnP: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDescriptionBytes+1))
	if err != nil {
		return "", "", err
	}
	if len(body) > maxDescriptionBytes {
		return "", "", errors.New("descrição UPnP grande demais")
	}
	return ParseDescription(body)
}
