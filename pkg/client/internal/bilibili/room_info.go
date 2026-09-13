package bilibili

import (
	"fmt"
	"github.com/iyear/pure-live-core/model"
	"strconv"
)

type roomInfoResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RoomID   int64  `json:"room_id"`
		UID      int64  `json:"uid"`
		Status   int    `json:"live_status"`
		Title    string `json:"title"`
		Cover    string `json:"user_cover"`
		Keyframe string `json:"keyframe"`
	} `json:"data"`
}

func (response roomInfoResponse) roomInfo() (*model.RoomInfo, error) {
	if response.Code != 0 || response.Data.RoomID <= 0 {
		return nil, fmt.Errorf("bilibili room info: code %d, %s", response.Code, response.Message)
	}
	data := response.Data
	cover := data.Cover
	if cover == "" {
		cover = data.Keyframe
	}
	status := 0
	if data.Status == 1 {
		status = 1
	}
	room := strconv.FormatInt(data.RoomID, 10)
	return &model.RoomInfo{Room: room, Title: data.Title, Upper: fmt.Sprintf("主播 %d", data.UID), Status: status, Link: "https://live.bilibili.com/" + room, Cover: cover}, nil
}
