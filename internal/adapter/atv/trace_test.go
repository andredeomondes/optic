package atv

import (
	"strings"
	"testing"

	"optic/internal/domain"

	"google.golang.org/protobuf/encoding/protowire"
)

type traceEvents struct {
	domain.SessionEvents
	lines []string
}

func (e *traceEvents) OnTrace(_, line string) { e.lines = append(e.lines, line) }

// batchEditFrame builds RemoteImeBatchEdit (21) carrying text, like SendText does.
func batchEditFrame(text string) []byte {
	var obj []byte
	obj = protowire.AppendTag(obj, 3, protowire.BytesType)
	obj = protowire.AppendString(obj, text)
	var edit []byte
	edit = protowire.AppendTag(edit, 2, protowire.BytesType)
	edit = protowire.AppendBytes(edit, obj)
	var p []byte
	p = protowire.AppendTag(p, 3, protowire.BytesType)
	p = protowire.AppendBytes(p, edit)
	var msg []byte
	msg = protowire.AppendTag(msg, 21, protowire.BytesType)
	return protowire.AppendBytes(msg, p)
}

func TestTracerOmitsTypedText(t *testing.T) {
	for _, text := range []string{"minhasenha123", "hi"} {
		ev := &traceEvents{}
		tracer(ev)(">", batchEditFrame(text))
		if len(ev.lines) != 1 || ev.lines[0] != "21 { [texto omitido] }" {
			t.Fatalf("%q: lines = %q", text, ev.lines)
		}
		if strings.Contains(ev.lines[0], "minhasenha") || strings.Contains(ev.lines[0], "105") {
			t.Errorf("%q leaked into trace: %q", text, ev.lines[0])
		}
	}
}

func TestTracerKeepsOtherFrames(t *testing.T) {
	ev := &traceEvents{}
	var msg []byte
	msg = protowire.AppendTag(msg, 10, protowire.BytesType)
	msg = protowire.AppendBytes(msg, []byte{0x08, 0x03, 0x10, 0x03})
	tracer(ev)(">", msg)
	if len(ev.lines) != 1 || strings.Contains(ev.lines[0], "texto omitido") {
		t.Fatalf("lines = %q", ev.lines)
	}
}
