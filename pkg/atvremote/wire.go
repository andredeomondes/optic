package atvremote

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

// Every message on both ports is a protobuf prefixed by its varint length.
const maxFrame = 64 << 10

func writeFrame(w io.Writer, msg []byte) error {
	buf := protowire.AppendVarint(nil, uint64(len(msg)))
	_, err := w.Write(append(buf, msg...))
	return err
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	if n > maxFrame {
		return nil, fmt.Errorf("frame too large: %d bytes", n)
	}
	msg := make([]byte, n)
	_, err = io.ReadFull(r, msg)
	return msg, err
}

// fields is a minimal decoded protobuf message: field number -> raw values.
// Varints are stored as uint64, length-delimited fields as []byte.
type fields map[protowire.Number][]any

func parseFields(b []byte) (fields, error) {
	out := fields{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		b = b[n:]
		switch typ {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			out[num] = append(out[num], v)
			b = b[n:]
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			out[num] = append(out[num], v)
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return out, nil
}

func (f fields) uint(num protowire.Number) (uint64, bool) {
	for _, v := range f[num] {
		if u, ok := v.(uint64); ok {
			return u, true
		}
	}
	return 0, false
}

func (f fields) bytes(num protowire.Number) ([]byte, bool) {
	for _, v := range f[num] {
		if b, ok := v.([]byte); ok {
			return b, true
		}
	}
	return nil, false
}

// DumpRaw renders a protobuf without its schema, like `protoc --decode_raw`.
// Handy for reverse engineering messages we have no .proto for.
func DumpRaw(b []byte) string {
	var sb strings.Builder
	dumpRaw(&sb, b, 0)
	return sb.String()
}

func dumpRaw(sb *strings.Builder, b []byte, depth int) {
	f, err := parseFields(b)
	if err != nil {
		fmt.Fprintf(sb, "%s<raw %x>\n", strings.Repeat("  ", depth), b)
		return
	}
	ind := strings.Repeat("  ", depth)
	for num := protowire.Number(1); num < 1<<12; num++ {
		for _, v := range f[num] {
			switch v := v.(type) {
			case uint64:
				fmt.Fprintf(sb, "%s%d: %d\n", ind, num, v)
			case []byte:
				if sub, err := parseFields(v); err == nil && len(sub) > 0 && len(v) > 1 {
					fmt.Fprintf(sb, "%s%d {\n", ind, num)
					dumpRaw(sb, v, depth+1)
					fmt.Fprintf(sb, "%s}\n", ind)
				} else {
					fmt.Fprintf(sb, "%s%d: %q\n", ind, num, v)
				}
			}
		}
	}
}
