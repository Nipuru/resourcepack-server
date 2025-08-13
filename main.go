package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"resourcepack-server/config"
	"resourcepack-server/packs"
	"resourcepack-server/server"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("创建日志目录失败: %v", err)
	}

	logger := initLogger()
	defer logger.Sync()

	logger.Info("正在启动资源包服务器...")

	if err := createDirectoryStructure(); err != nil {
		logger.Fatal("创建目录结构失败", zap.Error(err))
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}
	logger.Info("配置加载成功")

	packsConfig := &packs.Config{
		Directory:           cfg.Packs.Directory,
		FileMonitor:         cfg.Packs.FileMonitor,
		FileMonitorInterval: time.Duration(cfg.Packs.FileMonitorInterval * float64(time.Second)),
		ScanCooldown:        time.Duration(cfg.Packs.ScanCooldown * float64(time.Second)),
	}

	packsManager, err := packs.NewPacksManager(packsConfig, logger)
	if err != nil {
		logger.Fatal("初始化资源包管理器失败", zap.Error(err))
	}
	logger.Info("资源包管理器初始化完成")

	httpServer := server.NewServer(cfg, packsManager, logger)
	logger.Info("HTTP服务器初始化完成")

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("启动HTTP服务器", zap.String("address", addr))
		
		if err := httpServer.Run(); err != nil {
			logger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	waitForShutdown(logger, packsManager)
}

func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"logs/server.log", "stdout"}
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}

	return logger
}

func createDirectoryStructure() error {
	dirs := []string{
		"config",
		"logs",
		"resourcepacks",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}

	return nil
}

func waitForShutdown(logger *zap.Logger, packsManager *packs.PacksManager) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("收到关闭信号", zap.String("signal", sig.String()))

	logger.Info("正在关闭服务器...")
	
	packsManager.StopFileMonitoring()
	logger.Info("文件监控已停止")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		logger.Warn("关闭超时，强制退出")
	case <-time.After(2 * time.Second):
		logger.Info("服务器已关闭")
	}
}
