package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// 默认配置文件路径
const (
	CfgPath = "config.json"
)

// 配置信息结构体，与配置文件对应
type Config struct {
	// NODE_ID string `json:"NODE_ID"`
	ROLE       string `json:"ROLE"`
	IP         string `json:"IP"`
	PORT       int    `json:"PORT"`
	RemoteIP   string `json:"RemoteIP"`
	RemotePort int    `json:"RemotePort"`
	// UPPER_NODE string `json:"UPPER_NODE"`
}

// 全局结构体，供其他模块获取配置信息
var Cfg *Config

// 配置信息加载
// 参数为指定配置文件路径，若未指定则使用默认路径
func LoadConfig(cfgSpecific ...string) (*Config, error) {
	cfgPath := CfgPath

	if len(cfgSpecific) > 0 && cfgSpecific[0] != "" {
		cfgPath = cfgSpecific[0]
	}

	fmt.Println("Loading config from: ", cfgPath)
	file, err := os.Open(cfgPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
