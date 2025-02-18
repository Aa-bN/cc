package main

import (
	"cc/base/config"
	"cc/core/client"
	"flag"
	"fmt"
	"sync"
)

func main() {
	// 从命令行获取配置文件路径
	cfgPath := flag.String("config", config.CfgPath, "config file path")
	flag.Parse()

	// 加载配置文件（LoadConfig中，若命令行未指定配置文件，则默认读取当前目录下的config.json）
	cfg, err := config.LoadConfig(*cfgPath)
	config.Cfg = cfg
	if err != nil {
		panic(err)
	}
	fmt.Println(config.Cfg)

	localPort := config.Cfg.PORT
	remoteAddr := config.Cfg.RemoteIP
	remotePort := config.Cfg.RemotePort

	// 创建客户端
	cli, err := client.NewClient(localPort, remoteAddr, remotePort)
	if err != nil {
		fmt.Printf("error creating client: %v\n", err)
		return
	}
	defer cli.Close()

	fmt.Printf("RTP Client started - Local Port: %d, Remote: %s:%d\n",
		localPort, remoteAddr, remotePort)
	rtpSSRC, _ := client.GetClientConfigSSRC(cli)
	fmt.Printf("Local SSRC: 0x%x\n", rtpSSRC)

	var wg sync.WaitGroup
	wg.Add(2)

	// 启动接受协程
	go cli.StartReceiving(&wg)

	// 启动发送协程
	go cli.StartSending(&wg)

	wg.Wait()

}
