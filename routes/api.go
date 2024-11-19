package route

import (
	"droxy/config"
	"droxy/core/_http/service"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

func Init(server *echo.Echo) {
	// Apply CORS middleware
	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Replace "*" with specific origins for security
	}))
	server.GET("/health", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "I am alive.........")
	})
	server.POST("/container", func(ctx echo.Context) error {
		cs := new(service.Container)
		var err error
		err = ctx.Bind(cs)
		if err != nil {
			log.Println("Binding payload failed ", err)
			return ctx.JSON(http.StatusOK, err.Error())
		}
		if cs.Image == "" {
			return ctx.JSON(http.StatusOK, "Image name required!")

		}
		if cs.Name == "" {
			return ctx.JSON(http.StatusOK, "Container name required!")
		}
		if cs.Tag == "" {
			cs.Tag = "latest"
		}
		go func() {
			err = cs.CreateContainer(cs.Name, cs.Image, cs.Tag)
		}()
		if err != nil {
			return ctx.JSON(http.StatusOK, err.Error())
		}
		//"Container created successfully!"
		return ctx.JSON(http.StatusOK, config.ContainerCache)
	})
}

func InitProxyServer(server *echo.Echo) {
	server.GET("/health", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "Proxy server is running.........")
	})
	server.Any("/*", func(ctx echo.Context) error {
		log.Println(ctx.Request().Host)
		//if ctx.Request().Host == "load-bs.localhost" {
		//	log.Println(ctx.Request().Host)
		//}
		containerName := strings.Split(ctx.Request().Host, ".")[0]
		if _, ok := config.ContainerCache[containerName]; ok {
			if config.ContainerCache[containerName].NetworkSettings.Networks[config.ContainerCache[containerName].HostConfig.NetworkMode].IPAddress == "" {
				return ctx.JSON(http.StatusForbidden, fmt.Sprintf("Looks like  %s is not alive!", ctx.Request().Host))
			}
			log.Println("Forwarding Request => ", config.ContainerCache[containerName].NetworkSettings.Networks[config.ContainerCache[containerName].HostConfig.NetworkMode].IPAddress+":"+strconv.Itoa(int(config.ContainerCache[containerName].Ports[0].PrivatePort)))
			return ReverseProxy(config.ContainerCache[containerName].NetworkSettings.Networks[config.ContainerCache[containerName].HostConfig.NetworkMode].IPAddress, strconv.Itoa(int(config.ContainerCache[containerName].Ports[0].PrivatePort)))(ctx)
		} else {
			log.Println("[Error] container does not exist")
			return ctx.JSON(http.StatusForbidden, fmt.Sprintf("Looks like %s is not exist!", ctx.Request().Host))
		}
	})
}

func ReverseProxy(target string, port string) echo.HandlerFunc {
	urlStr := "http://" + target + ":" + port
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Println("Invalid target URL")
		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	path := u.Path
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	return func(ctx echo.Context) error {
		// Update the request host and headers
		r := ctx.Request()
		r.Host = u.Host // Use the host part only
		r.URL.Path = r.URL.Path + path
		r.Header.Set("X-Forwarded-Host", ctx.Request().Host)
		r.Header.Set("X-Real-IP", ctx.RealIP())

		// Forward the request to the target server
		proxy.ServeHTTP(ctx.Response(), r)
		return nil
	}
}
