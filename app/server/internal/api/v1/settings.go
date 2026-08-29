package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/app/server/internal/config"
	"github.com/iyear/pure-live-core/pkg/conf"
	"github.com/iyear/pure-live-core/pkg/ecode"
	"github.com/iyear/pure-live-core/pkg/format"
)

func GetServerSettings(c *gin.Context) {
	format.HTTP(c, ecode.Success, nil, config.GetServer())
}

func SaveServerSettings(c *gin.Context) {
	var next config.ServerConfig
	if err := c.ShouldBindJSON(&next); err != nil {
		format.HTTP(c, ecode.InvalidParams, err, nil)
		return
	}
	if err := config.SaveServer(next); err != nil {
		format.HTTP(c, ecode.InvalidParams, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, gin.H{"restartRequired": true})
}

func GetAccountSettings(c *gin.Context) {
	format.HTTP(c, ecode.Success, nil, conf.GetAccountMasked())
}

func SaveAccountSettings(c *gin.Context) {
	var next conf.AccountConfig
	if err := c.ShouldBindJSON(&next); err != nil {
		format.HTTP(c, ecode.InvalidParams, err, nil)
		return
	}
	if err := conf.SaveAccount(next); err != nil {
		format.HTTP(c, ecode.InvalidParams, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, gin.H{"restartRequired": true})
}

func GetChannels(c *gin.Context) {
	format.HTTP(c, ecode.Success, nil, config.GetChannels())
}

func SaveChannels(c *gin.Context) {
	var next []config.Channel
	if err := c.ShouldBindJSON(&next); err != nil {
		format.HTTP(c, ecode.InvalidParams, err, nil)
		return
	}
	if err := config.SaveChannels(next); err != nil {
		format.HTTP(c, ecode.InvalidParams, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, gin.H{"restartRequired": false})
}
