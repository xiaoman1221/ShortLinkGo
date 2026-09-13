package main

import (
	"log"

	"ShortLinkGo/config"
	"ShortLinkGo/database"
	"ShortLinkGo/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	r := router.New(cfg, db)

	addr := ":" + cfg.Port
	log.Printf("服务已启动，监听 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
