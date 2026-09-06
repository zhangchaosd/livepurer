package server

import (
	"context"
	"fmt"
	"github.com/iyear/pure-live-core/app/server/internal/config"
	"github.com/iyear/pure-live-core/app/server/internal/frontend"
	"github.com/iyear/pure-live-core/app/server/internal/logger"
	"github.com/iyear/pure-live-core/app/server/internal/router"
	"github.com/iyear/pure-live-core/global"
	"github.com/iyear/pure-live-core/pkg/conf"
	"github.com/iyear/pure-live-core/pkg/db"
	"github.com/iyear/pure-live-core/pkg/request"
	"github.com/iyear/pure-live-core/pkg/util"
	"github.com/q191201771/naza/pkg/nazalog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func Run(serverConf string, accountConf string) {
	if err := config.InitServer(serverConf); err != nil {
		log.Fatalf("failed to read server config: %s", err)
	}
	if err := conf.InitAccount(accountConf); err != nil {
		log.Fatalf("failed to read account config: %s", err)
	}

	logger.Init(util.IF(config.Server.Debug, zapcore.DebugLevel, zapcore.InfoLevel).(zapcore.LevelEnabler))

	// lal包中的nazalog
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.IsToStdout = config.Server.Debug
	})

	// 频道配置为可选: 无配置文件时仅 m3u 接口不可用, 不影响其他功能
	channelsConf := filepath.Join(filepath.Dir(serverConf), "channels.yaml")
	if err := config.InitChannels(channelsConf); err != nil {
		zap.S().Warnw("failed to read channels config (m3u playlist disabled)", "path", channelsConf, "error", err)
	}

	zap.S().Infof("read config succ...")

	if err := os.MkdirAll(config.Server.Path, 0774); err != nil {
		zap.S().Fatalw("failed to mkdir", "error", err)
	}
	staticPath := filepath.Join(config.Server.Path, "static")
	if err := frontend.Extract(staticPath); err != nil {
		zap.S().Fatalw("failed to extract embedded frontend", "error", err)
	}

	sqlite, err := db.Init(filepath.Join(config.Server.Path, "data.db"))
	if err != nil {
		zap.S().Fatalw("failed to init database", "error", err)
	}
	global.DB = sqlite
	zap.S().Infof("init database succ...")

	if config.Server.Socks5.Enable {
		request.SetSocks5(config.Server.Socks5.Host, config.Server.Socks5.Port, config.Server.Socks5.User, config.Server.Socks5.Password)
	}

	zap.S().Infof("server runs on :%d,debug: %v", config.Server.Port, config.Server.Debug)

	handler := router.Init(staticPath)

	s := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Server.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErrors := make(chan error, 1)
	go func() { serveErrors <- s.ListenAndServe() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)
	select {
	case <-quit:
	case err := <-serveErrors:
		if err != nil && err != http.ErrServerClosed {
			zap.S().Fatalw("failed to listen and serve", "error", err)
		}
		return
	}
	zap.S().Info("shutdown server...")

	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	if err = s.Shutdown(ctx); err != nil {
		zap.S().Fatalw("server forced to shutdown", "error", err)
	}

	zap.S().Infow("server exited...")
}
