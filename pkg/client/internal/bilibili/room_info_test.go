package bilibili

import (
	"encoding/json"
	"testing"
)

func TestRoomInfoCoverAndKeyframe(t *testing.T) {
	var response roomInfoResponse
	if err := json.Unmarshal([]byte(`{"code":0,"data":{"room_id":7734200,"uid":50329118,"live_status":1,"title":"直播标题","user_cover":"https://example.com/cover.jpg","keyframe":"https://example.com/frame.jpg"}}`), &response); err != nil {
		t.Fatal(err)
	}
	info, err := response.roomInfo()
	if err != nil || info.Room != "7734200" || info.Cover != "https://example.com/cover.jpg" || info.Status != 1 || info.Title != "直播标题" {
		t.Fatalf("unexpected info: %+v %v", info, err)
	}
	response.Data.Cover = ""
	response.Data.Status = 2
	info, err = response.roomInfo()
	if err != nil || info.Cover != "https://example.com/frame.jpg" || info.Status != 0 {
		t.Fatalf("unexpected fallback: %+v %v", info, err)
	}
	response.Code = -352
	if _, err := response.roomInfo(); err == nil {
		t.Fatal("API error accepted")
	}
	response.Code = 0
	response.Data.RoomID = 0
	if _, err := response.roomInfo(); err == nil {
		t.Fatal("empty room accepted")
	}
}
