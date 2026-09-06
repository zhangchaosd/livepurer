package huya

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/util/tools"
	"github.com/gorilla/websocket"
	"github.com/iyear/pure-live-core/model"
	"github.com/iyear/pure-live-core/pkg/client/internal/abstract"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/danmaku"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/heartbeat"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/online"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/push_msg"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/ws_cmd"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/ws_user_info"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/ws_verify_cookie_req"
	"github.com/iyear/pure-live-core/pkg/conf"
	"github.com/iyear/pure-live-core/pkg/util"
	"github.com/tidwall/gjson"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type Huya struct {
	*abstract.Client
	Cookies string
	UID     int64
}
type H map[string]interface{}

// NewHuya .
func NewHuya() (model.Client, error) {
	if !conf.Account.Huya.Enable {
		return &Huya{Cookies: "", UID: 0}, nil
	}

	yyuid, err := strconv.ParseInt(util.GetCookie(conf.Account.Huya.Cookies, "yyuid"), 10, 64)
	if err != nil {
		return nil, err
	}
	return &Huya{Cookies: conf.Account.Huya.Cookies, UID: yyuid}, nil
}

// Plat .
func (h *Huya) Plat() string {
	return conf.PlatHuya
}

// GetPlayURL .
func (h *Huya) GetPlayURL(room string, qn int) (*model.PlayURL, error) {
	roomInfo, err := getRoomInfo(room)
	if err != nil {
		return nil, err
	}
	u, err := nativeFLVURL(roomInfo)
	if err != nil {
		return nil, err
	}
	configurePlayURL(u)
	if err = refreshAntiCode(u, time.Now()); err != nil {
		return nil, err
	}

	return &model.PlayURL{
		Qn:     qn,
		Desc:   util.Qn2Desc(qn),
		Origin: u.String(),
		CORS:   false,
		Type:   conf.StreamFlv,
	}, err
}

func nativeFLVURL(roomInfo gjson.Result) (*url.URL, error) {
	// Use native FLV endpoints. Rewriting a mobile HLS URL retains mobile
	// anti-code parameters and can return short/repeated fragments.
	lines := roomInfo.Get("roomInfo.tLiveInfo.tLiveStreamInfo.vStreamInfo.value").Array()
	var u *url.URL
	var err error
	// HS and AL provide continuous FLV responses; TX can redirect to a CDN with
	// an invalid TLS hostname. Never bypass certificate validation.
	for _, cdn := range []string{"HS", "AL", "TX"} {
		for _, line := range lines {
			if line.Get("sCdnType").String() != cdn {
				continue
			}
			endpoint := line.Get("sFlvUrl").String()
			name := line.Get("sStreamName").String()
			code := line.Get("sFlvAntiCode").String()
			if endpoint == "" || name == "" || code == "" {
				continue
			}
			u, err = url.Parse(endpoint + "/" + name + ".flv?" + code)
			if err != nil {
				return nil, err
			}
			u.Scheme = "https"
			break
		}
		if u != nil {
			break
		}
	}
	if u == nil {
		return nil, fmt.Errorf("no available Huya FLV stream")
	}
	return u, nil
}

func configurePlayURL(u *url.URL) {
	query := u.Query()
	// 设置最高清晰度
	query.Set("ratio", "0")
	// 虎牙会在未指定编码时优先返回 HEVC(H.265) 流。浏览器中的
	// flv.js 与许多 IPTV 客户端不支持 HEVC FLV，显式请求 H.264，
	// 以保证网页播放和 M3U 播放地址的兼容性。
	query.Set("codec", "264")
	u.RawQuery = query.Encode()
}

// refreshAntiCode signs the native FLV request using the web client's two-stage
// digest. Signing the raw seqid (the legacy scheme) yields truncated streams.
func refreshAntiCode(u *url.URL, now time.Time) error {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	return refreshAntiCodeWithUID(u, now, 12340000+binary.BigEndian.Uint32(random[:])%10000)
}

func refreshAntiCodeWithUID(u *url.URL, now time.Time, uid uint32) error {
	query := u.Query()
	decoded, err := base64.StdEncoding.DecodeString(query.Get("fm"))
	if err != nil {
		return fmt.Errorf("decode Huya anti-code: %w", err)
	}
	prefix := strings.SplitN(string(decoded), "_", 2)[0]
	streamName := strings.TrimSuffix(path.Base(u.Path), path.Ext(u.Path))
	wsTime := query.Get("wsTime")
	if prefix == "" || streamName == "" || wsTime == "" {
		return fmt.Errorf("invalid Huya anti-code")
	}
	convertedUID := strconv.FormatUint(uint64(uid<<8|uid>>24), 10)
	seqID := strconv.FormatInt(now.UnixMilli()+int64(uid), 10)
	// Native web FLV parameters must be signed together, not inherited from HLS.
	query.Set("ctype", "huya_live")
	query.Set("t", "100")
	inner := fmt.Sprintf("%x", md5.Sum([]byte(seqID+"|huya_live|100")))
	signature := strings.Join([]string{prefix, convertedUID, streamName, inner, wsTime}, "_")
	query.Set("wsSecret", fmt.Sprintf("%x", md5.Sum([]byte(signature))))
	query.Set("u", convertedUID)
	query.Set("seqid", seqID)
	query.Set("sdk_sid", strconv.FormatInt(now.UnixMilli(), 10))
	query.Set("ver", "1")
	query.Set("sv", "2401090219")
	query.Del("fm")
	query.Del("txyp")
	u.RawQuery = query.Encode()
	return nil
}

// GetRoomInfo .
func (h *Huya) GetRoomInfo(room string) (*model.RoomInfo, error) {
	j, err := getRoomInfo(room)
	if err != nil {
		return nil, err
	}
	return &model.RoomInfo{
		Status: util.IF(j.Get("roomInfo.eLiveStatus").Int() == 2, 1, 0).(int),
		Room:   room,
		Upper:  j.Get("roomInfo.tProfileInfo.sNick").String(),
		Link:   fmt.Sprintf("https://www.huya.com/%s", room),
		Title:  j.Get("roomInfo.tLiveInfo.sIntroduction").String(),
	}, nil
}

// Host .
func (h *Huya) Host(room string) string {
	_ = room
	return "wss://cdnws.api.huya.com/"
}

// Enter .
func (h *Huya) Enter(room string) (int, [][]byte, error) {
	roomInfo, err := getRoomInfo(room)
	if err != nil {
		return -1, nil, err
	}
	lYyid := roomInfo.Get("roomInfo.tLiveInfo.lYyid").Int()
	lChannelId := roomInfo.Get("roomInfo.tLiveInfo.tLiveStreamInfo.vStreamInfo.value.0.lChannelId").Int()
	lSubChannelId := roomInfo.Get("roomInfo.tLiveInfo.tLiveStreamInfo.vStreamInfo.value.0.lSubChannelId").Int()
	// fmt.Println(lYyid, lChannelId, lSubChannelId)

	info := ws_user_info.WSUserInfo{
		LUid:       lYyid,
		BAnonymous: true,
		SGuid:      "",
		SToken:     "",
		LTid:       lChannelId,
		LSid:       lSubChannelId,
		LGroupId:   lYyid,
		LGroupType: 3,
	}

	buf := codec.NewBuffer()
	if err = info.WriteTo(buf); err != nil {
		return -1, nil, err
	}

	wsCmd := ws_cmd.WebSocketCommand{
		ICmdType: ewsCmdRegisterReq,
		VData:    tools.ByteToInt8(buf.ToBytes()),
	}

	buf = codec.NewBuffer()

	if err = wsCmd.WriteTo(buf); err != nil {
		return -1, nil, err
	}
	return websocket.BinaryMessage, [][]byte{buf.ToBytes()}, nil
}

// HeartBeat .
func (h *Huya) HeartBeat() (int, []byte, error) {
	userID := heartbeat.UserId{
		SHuyaUA: "webh5&1.0.0&websocket",
	}

	hbMsg := heartbeat.UserHeartBeatReq{
		TId:         userID,
		BWatchVideo: true,
		ELineType:   1,
	}

	buf := codec.NewBuffer()

	if err := hbMsg.WriteTo(buf); err != nil {
		return -1, nil, err
	}
	return websocket.BinaryMessage, buf.ToBytes(), nil
}

// Handle .
func (h *Huya) Handle(tp int, msg []byte) ([]model.Msg, bool, error) {
	if tp != websocket.BinaryMessage {
		return nil, false, nil
	}
	cmd := ws_cmd.WebSocketCommand{}
	if err := cmd.ReadFrom(codec.NewReader(msg)); err != nil {
		return nil, false, err
	}
	switch cmd.ICmdType {
	case ewsCmdS2CMsgPushReq:
		return h.handleMsgPushReq(codec.FromInt8(cmd.VData))
	}
	return nil, false, nil
}

func (h *Huya) handleMsgPushReq(b []byte) ([]model.Msg, bool, error) {
	r := make([]model.Msg, 1)
	msg := push_msg.WSPushMessage{}
	if err := msg.ReadFrom(codec.NewReader(b)); err != nil {
		return nil, false, err
	}
	// fmt.Println(msg.EPushType, msg.IUri)
	switch msg.IUri {
	case 1400: // 弹幕
		d := danmaku.MessageNotice{}
		if err := d.ReadFrom(codec.NewReader(codec.FromInt8(msg.SMsg))); err != nil {
			return nil, false, err
		}
		// fmt.Println(d.SContent, d.TUserInfo.SNickName, d.IShowMode, d.TBulletFormat.IFontColor)
		r[0] = &model.MsgDanmaku{
			Content: d.SContent,
			Type:    0, // TODO 没找到虎牙弹幕mode的字段
			Color:   int64(util.IF(d.TBulletFormat.IFontColor == -1, int32(16777215), d.TBulletFormat.IFontColor).(int32)),
		}
		return r, true, nil

	case 8006: // 直播间热度
		on := online.AttendeeCountNotice{}
		if err := on.ReadFrom(codec.NewReader(codec.FromInt8(msg.SMsg))); err != nil {
			return nil, false, err
		}
		r[0] = &model.MsgHot{Hot: int64(on.IAttendeeCount)}
		return r, true, nil
	}
	return nil, false, nil
}

// SendDanmaku .
func (h *Huya) SendDanmaku(room string, content string, tp int, color int64) error {
	_ = room
	_ = content
	_ = tp
	_ = color
	return errors.New("Huya danmaku sending is not supported")
}

func login(uid int64, cookies string) ([]byte, error) {
	verify := ws_verify_cookie_req.WSVerifyCookieReq{
		LUid:    uid,
		SUA:     "webh5&1.0.0&websocket",
		SCookie: cookies,
	}

	vbuf := codec.NewBuffer()
	if err := verify.WriteTo(vbuf); err != nil {
		return nil, err
	}
	wsCmd := ws_cmd.WebSocketCommand{
		ICmdType: ewsCmdC2SVerifyCookieReq,
		VData:    tools.ByteToInt8(vbuf.ToBytes()),
	}

	wbuf := codec.NewBuffer()
	if err := wsCmd.WriteTo(wbuf); err != nil {
		return nil, err
	}

	return wbuf.ToBytes(), nil
}

// Stop .
func (h *Huya) Stop() {

}
