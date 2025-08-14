package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

//go:embed config.template.toml
var configTemplate []byte

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Packs  PacksConfig  `mapstructure:"packs"`
	Log    LogConfig    `mapstructure:"logging"`
}

type ServerConfig struct {
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
	Debug bool   `mapstructure:"debug"`
}

type PacksConfig struct {
	Directory           string  `mapstructure:"directory"`
	FileMonitor         bool    `mapstructure:"file_monitor"`
	FileMonitorInterval float64 `mapstructure:"file_monitor_interval"`
	ScanCooldown        float64 `mapstructure:"scan_cooldown"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("config")

	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.debug", false)
	viper.SetDefault("packs.directory", "/resourcepacks")
	viper.SetDefault("packs.file_monitor", true)
	viper.SetDefault("packs.file_monitor_interval", 1.0)
	viper.SetDefault("packs.scan_cooldown", 2.0)
	viper.SetDefault("logging.level", "INFO")
	viper.SetDefault("logging.file", "logs/server.log")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := createConfigFromEmbedded(); err != nil {
				return nil, fmt.Errorf("创建配置文件失败: %w", err)
			}
			if err := viper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("读取配置文件失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

func createConfigFromEmbedded() error {
	configPath := "config.toml"

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	if err := os.WriteFile(configPath, configTemplate, 0644); err != nil {
		return err
	}
	return nil
}
