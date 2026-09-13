// ShortLinkGo 服务入口。
package main

import (
	"log"

	"ShortLinkGo/server/api"
	"ShortLinkGo/server/app"
	"ShortLinkGo/server/services"
)

func main() {
	cfg := app.Load()

	db, err := app.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// GeoIP 数据库：启动时加载本地文件（如有），启用后每小时自动检查更新
	services.StartGeoIPUpdater(db)

	r := api.New(cfg, db)

	addr := ":" + cfg.Port
	log.Printf("服务已启动，监听 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
