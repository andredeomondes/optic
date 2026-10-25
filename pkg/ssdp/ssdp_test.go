package ssdp

import (
	"net"
	"testing"
)

func TestParseAndClassify(t *testing.T) {
	cases := []struct {
		name, raw, want string
		ok              bool
	}{
		{"roku", "HTTP/1.1 200 OK\r\nCache-Control: max-age=3600\r\nST: roku:ecp\r\nUSN: uuid:roku:ecp:YN00AB\r\nLocation: http://192.168.0.30:8060/\r\n\r\n", "roku", true},
		{"lg", "HTTP/1.1 200 OK\r\nst: urn:lge-com:service:webos-second-screen:1\r\nusn: uuid:abc::urn:lge-com:service:webos-second-screen:1\r\n\r\n", "webos", true},
		{"samsung-st", "HTTP/1.1 200 OK\r\nST: urn:samsung.com:device:RemoteControlReceiver:1\r\n\r\n", "tizen", true},
		{"samsung-server", "HTTP/1.1 200 OK\r\nST: upnp:rootdevice\r\nSERVER: Samsung-Linux/4.1, UPnP/1.0, Tizen\r\n\r\n", "tizen", true},
		{"router", "HTTP/1.1 200 OK\r\nST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\nSERVER: Linux UPnP/1.0 miniupnpd\r\n\r\n", "", false},
		{"garbage", "NOTIFY * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\n\r\n", "", false},
		{"empty", "", "", false},
	}
	for _, c := range cases {
		r, ok := Parse([]byte(c.raw), net.IPv4(192, 168, 0, 30))
		if !ok {
			if c.ok {
				t.Errorf("%s: Parse rejected valid response", c.name)
			}
			continue
		}
		got, ok := Classify(r)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: Classify = %q,%v want %q,%v", c.name, got, ok, c.want, c.ok)
		}
		if r.IP != "192.168.0.30" {
			t.Errorf("%s: IP = %q", c.name, r.IP)
		}
	}
}
