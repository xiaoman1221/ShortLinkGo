// ShortLinkGo 服务入口。
package main

import (
	"log"

	"ShortLinkGo/server/api"
	"ShortLinkGo/server/app"
)

func main() {
	cfg := app.Load()

	db, err := app.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	r := api.New(cfg, db)

	addr := ":" + cfg.Port
	log.Printf("服务已启动，监听 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
