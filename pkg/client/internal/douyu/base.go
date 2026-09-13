package douyu

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
	"github.com/guonaihong/gout"
	"github.com/iyear/pure-live-core/model"
	"github.com/iyear/pure-live-core/pkg/client/internal/abstract"
	"github.com/iyear/pure-live-core/pkg/conf"
	"github.com/iyear/pure-live-core/pkg/request"
	"github.com/iyear/pure-live-core/pkg/util"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type Douyu struct {
	*abstract.Client
	// buf 缓存不足一个完整帧的剩余数据, 供下一次 Handle 拼接
	buf []byte
}

func NewDouyu() (model.Client, error) {
	return &Douyu{}, nil
}

// Plat
func (d *Douyu) Plat() string {
	return conf.PlatDouyu
}

// GetPlayURL
// cdn: 主线路ws-h5、备用线路tct-h5pot
// rate: 1流畅；2高清；3超清；4蓝光4M；0蓝光8M或10M
//
// 签名流程(2025年新版): 从 swf_api/homeH5Enc 接口获取混淆JS，
// 用 goja 执行其中的 ub98484234(rid, did, tt) 得到 v/did/tt/sign 参数，
// 再请求 getH5Play 接口获取直播流地址。
func (d *Douyu) GetPlayURL(room string, qn int) (*model.PlayURL, error) {
	var qnm = map[int]int{
		conf.QnBest: 0,
		conf.QnHigh: 4,
		conf.QnMid:  3,
		conf.QnLow:  1,
	}

	params, err := d.getSignParams(room)
	if err != nil {
		return nil, err
	}
	params = fmt.Sprintf("%s&cdn=ws-h5&rate=%d", params, qnm[qn])

	var resp struct {
		Error int    `json:"error"`
		Msg   string `json:"msg"`
		Data  struct {
			RoomID       int64  `json:"room_id"`
			IsMixed      bool   `json:"is_mixed"`
			MixedLive    string `json:"mixed_live"`
			MixedURL     string `json:"mixed_url"`
			RtmpCdn      string `json:"rtmp_cdn"`
			RtmpURL      string `json:"rtmp_url"`
			RtmpLive     string `json:"rtmp_live"`
			ClientIP     string `json:"client_ip"`
			InNA         int    `json:"inNA"`
			RateSwitch   int    `json:"rateSwitch"`
			Rate         int    `json:"rate"`
			CdnsWithName []*struct {
				Name   string `json:"name"`
				Cdn    string `json:"cdn"`
				IsH265 bool   `json:"isH265"`
			} `json:"cdnsWithName"`
			Multirates []*struct {
				Name    string `json:"name"`
				Rate    int    `json:"rate"`
				HighBit int    `json:"highBit"`
				Bit     int    `json:"bit"`
			} `json:"multirates"`
		}
	}

	err = request.HTTP().POST(fmt.Sprintf("https://www.douyu.com/lapi/live/getH5Play/%s?", room) + params).
		SetHeader(gout.H{
			"UserAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36",
			"referer":   "https://www.douyu.com/",
			"origin":    "https://www.douyu.com",
		}).
		BindJSON(&resp).
		Do()
	if err != nil {
		return nil, err
	}
	if resp.Error != 0 {
		return nil, fmt.Errorf("failed to get play url: %s", resp.Msg)
	}
	return &model.PlayURL{
		Qn:     qn,
		Desc:   util.Qn2Desc(qn),
		Origin: resp.Data.RtmpURL + "/" + resp.Data.RtmpLive,
		CORS:   true,
		Type:   conf.StreamFlv,
	}, nil
}

// getSignParams 获取 getH5Play 接口所需的签名参数(v/did/tt/sign)
func (d *Douyu) getSignParams(room string) (string, error) {
	// 从 swf_api 接口获取混淆签名JS(新版混淆脚本, 含 ub98484234 函数)
	var enc struct {
		Error int               `json:"error"`
		Data  map[string]string `json:"data"`
	}
	if err := request.HTTP().GET(fmt.Sprintf("https://www.douyu.com/swf_api/homeH5Enc?rids=%s", room)).
		SetHeader(gout.H{
			"UserAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36",
			"referer":   "https://www.douyu.com/",
		}).
		BindJSON(&enc).
		Do(); err != nil {
		return "", err
	}
	if enc.Error != 0 {
		return "", fmt.Errorf("failed to get sign script: %d", enc.Error)
	}
	jsUb9 := enc.Data["room"+room]
	if jsUb9 == "" {
		return "", fmt.Errorf("failed to get sign script for room %s", room)
	}

	// 注入 CryptoJS.MD5 的实现(goja 中无 CryptoJS, 用 Go 的 md5 替代)
	vm := goja.New()
	if _, err := vm.RunString(`var CryptoJS = { MD5: function(s) { return { toString: function() { return __pure_live_md5(s); } }; } };`); err != nil {
		return "", err
	}
	if err := vm.Set("__pure_live_md5", func(s string) string {
		return fmt.Sprintf("%x", md5.Sum([]byte(s)))
	}); err != nil {
		return "", err
	}
	if _, err := vm.RunString(jsUb9); err != nil {
		return "", err
	}

	ub9, ok := goja.AssertFunction(vm.Get("ub98484234"))
	if !ok {
		return "", fmt.Errorf("failed to assert function ub9")
	}

	// did 为随机32位hex, tt 为当前时间戳
	did := util.RandHex(16)
	tt := strconv.FormatInt(time.Now().Unix(), 10)

	res, err := ub9(goja.Undefined(), vm.ToValue(room), vm.ToValue(did), vm.ToValue(tt))
	if err != nil {
		return "", err
	}
	return res.String(), nil
}

// GetRoomInfo 通过房间号获取房间信息
func (d *Douyu) GetRoomInfo(room string) (*model.RoomInfo, error) {
	var info struct {
		Error int `json:"error"`
		Data  struct {
			RoomId     string `json:"room_id"`
			OwnerName  string `json:"owner_name"`
			RoomStatus string `json:"room_status"`
			RoomName   string `json:"room_name"`
			RoomThumb  string `json:"room_thumb"`
			Avatar     string `json:"avatar"`
		} `json:"data"`
	}
	if err := request.HTTP().GET(fmt.Sprintf("https://open.douyucdn.cn/api/RoomApi/room/%s", room)).BindJSON(&info).Do(); err != nil {
		zap.S().Warnf("Douyu: GetRoomInfo: http.Get room:%v, err:%v", room, err)
		return nil, err
	}
	if info.Error != 0 {
		zap.S().Warnf("Douyu: GetRoomInfo: rsp err code not 0, room:%v", room)
		return nil, errors.New("request err")
	}
	link := fmt.Sprintf("https://www.douyu.com/%s", info.Data.RoomId)
	return &model.RoomInfo{
		Cover:  info.Data.RoomThumb,
		Avatar: info.Data.Avatar,
		Status: util.IF(info.Data.RoomStatus == "1", 1, 0).(int),
		Room:   info.Data.RoomId,
		Upper:  info.Data.OwnerName,
		Link:   link,
		Title:  info.Data.RoomName,
	}, nil
}

// Host
func (d *Douyu) Host(room string) string {
	_ = room
	return "wss://danmuproxy.douyu.com:8503/"
}

func (d *Douyu) Enter(room string) (int, [][]byte, error) {
	return websocket.BinaryMessage, [][]byte{
		encode(map[string]string{"type": "loginreq", "roomid": room}),
		encode(map[string]string{"type": "joingroup", "rid": room, "gid": "-9999"}),
	}, nil
}

func (d *Douyu) Handle(tp int, data []byte) ([]model.Msg, bool, error) {
	if tp != websocket.BinaryMessage {
		return nil, false, nil
	}
	d.buf = append(d.buf, data...)
	types, bodies, rest := splitPacks(d.buf)
	d.buf = rest

	var msgs []model.Msg
	for i, body := range bodies {
		// 只处理弹幕消息(类型690), 其他消息(心跳/进房/礼物等)跳过
		if types[i] != msgTypeChat {
			continue
		}
		m := decodeBody(body)
		if m["type"] != "chatmsg" {
			continue
		}
		msgs = append(msgs, &model.MsgDanmaku{
			Content: m["txt"],
			Type:    conf.DanmakuTypeRight, // 没找到弹幕显示位置的字段
			Color:   colorConv(m["col"]),
		})
	}
	return msgs, true, nil
}

func (d *Douyu) HeartBeat() (int, []byte, error) {
	return websocket.BinaryMessage, encode(map[string]string{
		"type": "mrkl",
	}), nil
}

func (d *Douyu) SendDanmaku(room string, content string, tp int, color int64) error {
	_ = room
	_ = content
	_ = tp
	_ = color
	return fmt.Errorf("todo")
}

func (d *Douyu) Stop() {

}
