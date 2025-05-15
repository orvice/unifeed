package main

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"github.com/gin-gonic/gin"
	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/http"
)

var router func(*gin.Engine)

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:   conf.Conf,
		Service:  "unifeed",
		Router:   http.Router,
		InitFunc: []func() error{},
	})
	return app
}
