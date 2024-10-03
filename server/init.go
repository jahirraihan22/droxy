package server

import (
	"droxy/core/_http/service"
	route "droxy/routes"
	"github.com/labstack/echo/v4"
)

func Init() {
	managementAPI := echo.New()
	proxyServer := echo.New()

	route.Init(managementAPI)
	route.InitProxyServer(proxyServer)

	go func() {
		service.LookUpEvent()
	}()

	go func() {
		proxyServer.Logger.Fatal(proxyServer.Start(":8000"))
	}()

	go func() {
		managementAPI.Logger.Fatal(managementAPI.Start(":8080"))
	}()

	select {}
}
