package main

import (
	"eventexplorer/cache"
	"eventexplorer/config"
	_ "eventexplorer/routers"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		logs.Warn("Config data incomplete: %v", err)
	}
	cache.InitShared(cfg.CacheTTL)

	logs.Info("App running at http://localhost:8080 | cacheTTL=%s httpTimeout=%s perCategory=%d",
		cfg.CacheTTL, cfg.HTTPTimeout, cfg.EventsPerCategory)

	beego.Run()
}
