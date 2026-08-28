package douyu

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"strings"
)

// 斗鱼弹幕协议帧格式:
// [包长4字节小端(含12字节头)] [包长4字节(重复)] [消息类型4字节] [消息体] [0x00]
// 消息类型: 689=心跳响应 690=弹幕(chatmsg) 其他=进房/礼物等
const (
	msgHeaderLen = 8  // 两段包长
	msgTypeLen   = 4  // 消息类型
	msgMinLen    = 12 // 头部总长
	msgTypeChat  = 690
)

func encode(v interface{}) []byte {
	j, _ := json.Marshal(v)
	msg := strings.NewReplacer(`":"`, `@=`,
		`","`, `/`,
		`@`, `@A`,
		`/`, `@S`,
		`{"`, "",
		`"}`, "").Replace(string(j))
	total := make([]byte, 4)
	binary.LittleEndian.PutUint32(total, uint32(len(msg)+9))
	header := []byte{0xb1, 0x02, 0x00, 0x00}
	end := []byte{0x00}
	// 需要加两次数据包长度
	data := append(total, total...)
	data = append(data, header...)
	data = append(data, []byte(msg)...)
	data = append(data, end...)
	return data
}

// splitPacks 按协议帧头将数据拆分为多个完整数据包, 返回各包的消息类型与消息体,
// 以及不足以组成完整包的剩余字节(由调用方缓存, 等下一帧到达后继续拼接)
//
// 斗鱼协议帧布局: [包长4][包长4(重复)][消息类型4][消息体][0x00]
// 注意: 帧总长 = 包长字段 + 4 (包长字段不含自身起始的4字节, 实测确认)
func splitPacks(data []byte) (types []uint32, bodies [][]byte, rest []byte) {
	for {
		if len(data) < msgMinLen {
			rest = data
			return
		}
		l := int(binary.LittleEndian.Uint32(data[:4]))
		// 帧总长 = 包长字段 + 4
		total := l + 4
		// 包长异常(过小), 数据已损坏, 丢弃剩余数据
		if l < msgMinLen {
			rest = nil
			return
		}
		// 帧长超过剩余数据, 说明是半包, 保留等待下一帧
		if total > len(data) {
			rest = data
			return
		}
		pack := data[:total]
		data = data[total:]

		msgType := binary.LittleEndian.Uint32(pack[msgHeaderLen : msgHeaderLen+msgTypeLen])
		body := pack[msgHeaderLen+msgTypeLen:]
		// 去掉消息体末尾的 0x00 结束符
		body = bytes.TrimRight(body, "\x00")

		types = append(types, msgType)
		bodies = append(bodies, body)
	}
}

// decodeBody 解析斗鱼 KV 消息体(type@=chatmsg/rid@=xxx/...), 返回键值对
// 转义规则: @A 表示字面 @, @S 表示字面 /。
// 注意必须先按未反转义的 / 拆分字段, 再对每个字段做 @A/@S 反转义, 否则值里的 / 会被误拆
func decodeBody(msg []byte) map[string]string {
	m := make(map[string]string)
	for _, part := range strings.Split(string(msg), "/") {
		kv := strings.SplitN(part, "@=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.NewReplacer(`@A`, `@`, `@S`, `/`).Replace(kv[0])
		v := strings.NewReplacer(`@A`, `@`, `@S`, `/`).Replace(kv[1])
		m[k] = v
	}
	return m
}

var col = map[int]int64{
	1: 16723502,
	2: 52479,
	3: 6749952,
	5: 13369599,
	6: 16139391,
	4: 16737792,
}

func colorConv(color string) int64 {
	if color == "" {
		return 16777215
	}
	c, err := strconv.Atoi(color)
	if err != nil {
		return 16777215
	}
	return col[c]
}
