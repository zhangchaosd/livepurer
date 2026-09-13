package bilibili

import (
	"fmt"
	"github.com/iyear/biligo"
	"github.com/iyear/pure-live-core/model"
	"github.com/iyear/pure-live-core/pkg/client/internal/abstract"
	"github.com/iyear/pure-live-core/pkg/conf"
	"github.com/iyear/pure-live-core/pkg/request"
	"github.com/iyear/pure-live-core/pkg/util"
	"strconv"
)

type base struct {
	*abstract.Client
}

// NewBiliBili .
func NewBiliBili() (model.Client, error) {
	if !conf.Account.BiliBili.Enable {
		return &BiliComm{
			client: biligo.NewCommClient(&biligo.CommSetting{
				DebugMode: false,
			}),
		}, nil
	}
	b, err := biligo.NewBiliClient(&biligo.BiliSetting{
		Auth: &biligo.CookieAuth{
			DedeUserID:      conf.Account.BiliBili.DedeUserID,
			DedeUserIDCkMd5: conf.Account.BiliBili.DedeUserIDCkMd5,
			SESSDATA:        conf.Account.BiliBili.SESSDATA,
			BiliJCT:         conf.Account.BiliBili.BiliJCT,
		},
		DebugMode: false,
	})
	if err != nil {
		return nil, err
	}
	return &BiliBili{client: b}, nil
}

// Plat .
func (c *base) Plat() string {
	return conf.PlatBiliBili
}

// GetPlayURL .
func (c *base) GetPlayURL(room string, qn int) (*model.PlayURL, error) {
	client := biligo.NewCommClient(&biligo.CommSetting{})
	roomNum, err := strconv.ParseInt(room, 10, 64)
	if err != nil {
		return nil, err
	}

	// 内部维护一个qn映射表
	q := map[int]int{
		conf.QnBest: 20000,
		conf.QnHigh: 10000,
		conf.QnMid:  400,
		conf.QnLow:  250,
	}

	r, err := client.LiveGetPlayURL(roomNum, q[qn])
	if err != nil {
		return nil, err
	}
	return &model.PlayURL{
		Qn:     qn,
		Desc:   util.Qn2Desc(qn),
		Origin: r.DURL[0].URL,
		CORS:   false,
		Type:   conf.StreamFlv,
	}, nil
}

// GetRoomInfo .
func (c *base) GetRoomInfo(room string) (*model.RoomInfo, error) {
	roomNum, err := strconv.ParseInt(room, 10, 64)
	if err != nil || roomNum <= 0 {
		return nil, fmt.Errorf("invalid room ID")
	}
	var response roomInfoResponse
	headers := map[string]string{"User-Agent": "Mozilla/5.0", "Referer": "https://live.bilibili.com/"}
	if err := request.HTTP().GET(fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%d", roomNum)).SetHeader(headers).BindJSON(&response).Do(); err != nil {
		return nil, err
	}
	info, err := response.roomInfo()
	if err != nil {
		return nil, err
	}
	// Anchor enrichment must not discard an otherwise valid room and cover.
	var anchor struct {
		Code int `json:"code"`
		Data struct {
			Info struct {
				Name string `json:"uname"`
				Face string `json:"face"`
			} `json:"info"`
		} `json:"data"`
	}
	if err := request.HTTP().GET("https://api.live.bilibili.com/live_user/v1/UserInfo/get_anchor_in_room?roomid=" + info.Room).SetHeader(headers).BindJSON(&anchor).Do(); err == nil && anchor.Code == 0 {
		if anchor.Data.Info.Name != "" {
			info.Upper = anchor.Data.Info.Name
		}
		info.Avatar = anchor.Data.Info.Face
	}
	return info, nil
}

// Stop .
func (c *base) Stop() {

}
