package douyu

import (
	"encoding/binary"
	"github.com/gorilla/websocket"
	"github.com/iyear/pure-live-core/model"
	"testing"
)

// 构造一个标准的斗鱼弹幕帧
func makeChatFrame(txt, col string) []byte {
	body := "type@=chatmsg/txt@=" + txt + "/col@=" + col + "/nn@=user"
	// 帧 = [包长][包长][类型690][body][\x00]
	// 注意: 包长字段 = 帧总长 - 4 (协议约定, 实测确认)
	frameLen := msgHeaderLen + msgTypeLen + len(body) + 1
	frame := make([]byte, 0, frameLen)
	lenB := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenB, uint32(frameLen-4))
	frame = append(frame, lenB...)
	frame = append(frame, lenB...)
	typeB := make([]byte, 4)
	binary.LittleEndian.PutUint32(typeB, msgTypeChat)
	frame = append(frame, typeB...)
	frame = append(frame, []byte(body)...)
	frame = append(frame, 0x00)
	return frame
}

// 含特殊字符 / 和 @ 的弹幕, 验证转义
func makeChatFrameEscaped(txt string) []byte {
	escaped := ""
	for _, c := range txt {
		switch c {
		case '@':
			escaped += "@A"
		case '/':
			escaped += "@S"
		default:
			escaped += string(c)
		}
	}
	return makeChatFrame(escaped, "2")
}

func TestSplitPacks(t *testing.T) {
	// 单帧
	f1 := makeChatFrame("hello", "1")
	types, bodies, rest := splitPacks(f1)
	if len(types) != 1 || len(bodies) != 1 || len(rest) != 0 {
		t.Fatalf("single frame: types=%d bodies=%d rest=%d", len(types), len(bodies), len(rest))
	}
	if types[0] != msgTypeChat {
		t.Fatalf("msg type = %d, want 690", types[0])
	}
	m := decodeBody(bodies[0])
	if m["txt"] != "hello" || m["type"] != "chatmsg" || m["col"] != "1" {
		t.Fatalf("decode body: %v", m)
	}

	// 多帧粘包
	multi := append(append(append([]byte{}, f1...), makeChatFrame("second", "5")...), makeChatFrame("third", "3")...)
	types, bodies, rest = splitPacks(multi)
	if len(types) != 3 || len(rest) != 0 {
		t.Fatalf("multi frame: types=%d rest=%d", len(types), len(rest))
	}

	// 半包(拆成两段)
	half := len(f1) / 2
	types, bodies, rest = splitPacks(f1[:half])
	if len(types) != 0 || len(bodies) != 0 || len(rest) != half {
		t.Fatalf("half frame: types=%d rest=%d want %d", len(types), len(rest), half)
	}
	types, bodies, rest = splitPacks(append(rest, f1[half:]...))
	if len(types) != 1 || len(rest) != 0 {
		t.Fatalf("half+half: types=%d rest=%d", len(types), len(rest))
	}
}

func TestDecodeBodyEscape(t *testing.T) {
	f := makeChatFrameEscaped("a@b/cd")
	m := decodeBody(f[msgHeaderLen+msgTypeLen : len(f)-1])
	if m["txt"] != "a@b/cd" {
		t.Fatalf("escape decode: %q", m["txt"])
	}
}

func TestHandle(t *testing.T) {
	d := &Douyu{}
	f := makeChatFrame("弹幕测试", "6")
	msgs, ok, err := d.Handle(websocket.BinaryMessage, f)
	if err != nil || !ok || len(msgs) != 1 {
		t.Fatalf("handle: msgs=%d ok=%v err=%v", len(msgs), ok, err)
	}
	if msgs[0].(*model.MsgDanmaku).Content != "弹幕测试" {
		t.Fatalf("content = %q", msgs[0].(*model.MsgDanmaku).Content)
	}

	// 半包跨 Handle 调用缓存
	d2 := &Douyu{}
	half := len(f) / 2
	msgs, _, _ = d2.Handle(websocket.BinaryMessage, f[:half])
	if len(msgs) != 0 {
		t.Fatalf("half handle should produce nothing")
	}
	msgs, _, _ = d2.Handle(websocket.BinaryMessage, f[half:])
	if len(msgs) != 1 {
		t.Fatalf("second half should produce 1 msg")
	}
}
