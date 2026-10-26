package tizen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var infoTimeout = 3 * time.Second

const (
	infoPort     = "8001"
	maxInfoBytes = 1 << 20
)

// Info is what the TV reports on its REST endpoint.
type Info struct {
	Name       string
	Model      string
	WifiMAC    string
	PowerState string // "on" when the TV does not report it (older models)
	OS         string
	TokenAuth  bool
	ModelCode  string // e.g. "18_KANTM2_QTV"
	Year       string // "2018", from the model code prefix; empty when unknown
	ImeSynced  bool   // the TV reports its text-field state (ImeSyncedSupport)
}

// GetInfo reads GET {base}/api/v2/. base is "http://host:8001", or just the
// host (then http and port 8001 are assumed). The client should have a
// timeout; GetInfo caps it at 3s, never follows redirects (on a copy of client) and
// reads at most 1 MiB.
func GetInfo(ctx context.Context, client *http.Client, base string) (Info, error) {
	if !strings.Contains(base, "://") {
		base = "http://" + net.JoinHostPort(base, infoPort)
	}
	if client == nil {
		client = &http.Client{}
	}
	cl := *client
	cl.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if cl.Timeout <= 0 || cl.Timeout > infoTimeout {
		cl.Timeout = infoTimeout
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v2/", nil)
	if err != nil {
		return Info{}, errors.New("tizen: endereço inválido")
	}
	resp, err := cl.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return Info{}, fmt.Errorf("tizen: consultar a TV: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("tizen: a TV respondeu %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxInfoBytes+1))
	if err != nil {
		return Info{}, fmt.Errorf("tizen: ler resposta da TV: %w", err)
	}
	if len(body) > maxInfoBytes {
		return Info{}, errors.New("tizen: resposta da TV grande demais")
	}
	var raw struct {
		Device struct {
			Name       string          `json:"name"`
			ModelName  string          `json:"modelName"`
			Model      string          `json:"model"`
			ImeSynced  json.RawMessage `json:"ImeSyncedSupport"`
			WifiMAC    string          `json:"wifiMac"`
			PowerState string          `json:"PowerState"`
			OS         string          `json:"OS"`
			TokenAuth  json.RawMessage `json:"TokenAuthSupport"`
		} `json:"device"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Info{}, errors.New("tizen: resposta da TV inválida")
	}
	d := raw.Device
	info := Info{
		Name:       d.Name,
		Model:      d.ModelName,
		WifiMAC:    d.WifiMAC,
		PowerState: d.PowerState,
		OS:         d.OS,
		TokenAuth:  rawTrue(d.TokenAuth),
		ModelCode:  d.Model,
		Year:       modelYear(d.Model),
		ImeSynced:  rawTrue(d.ImeSynced),
	}
	if info.PowerState == "" {
		info.PowerState = "on"
	}
	return info, nil
}

// rawTrue reads a JSON "true" or true.
func rawTrue(r json.RawMessage) bool {
	return strings.EqualFold(strings.Trim(string(r), `"`), "true")
}

// modelYear turns a code like "18_KANTM2_QTV" into "2018".
func modelYear(code string) string {
	if len(code) >= 3 && code[2] == '_' && code[0] >= '0' && code[0] <= '9' && code[1] >= '0' && code[1] <= '9' {
		return "20" + code[:2]
	}
	return ""
}
