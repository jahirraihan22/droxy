package route

import (
	"droxy/config"
	"droxy/core/_http/service"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

func Init(server *echo.Echo) {
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
			log.Println("Forwarding Request => ", config.ContainerCache[containerName].NetworkSettings.Networks[config.ContainerCache[containerName].HostConfig.NetworkMode].IPAddress+":"+strconv.Itoa(int(config.ContainerCache[containerName].Ports[0].PrivatePort)))
			return ReverseProxy(config.ContainerCache[containerName].NetworkSettings.Networks[config.ContainerCache[containerName].HostConfig.NetworkMode].IPAddress, strconv.Itoa(int(config.ContainerCache[containerName].Ports[0].PrivatePort)))(ctx)
		} else {
			log.Println("[Error] container does not exist")
		}
		return ctx.JSON(http.StatusForbidden, fmt.Sprintf("Looks like the host %s is not exist ", ctx.Request().Host))
	})
}

//
//func ReverseProxy(target string, port string) echo.HandlerFunc {
//	urlStr := "http://" + target + ":" + port
//	u, err := url.Parse(urlStr)
//	if err != nil {
//		log.Println("Invalid target URL")
//		return nil
//	}
//
//	proxy := httputil.NewSingleHostReverseProxy(u)
//
//	path := u.Path
//	if strings.HasPrefix(path, "/") {
//		path = path[1:]
//	}
//
//	return func(ctx echo.Context) error {
//		// Update the request host and headers
//		r := ctx.Request()
//		r.Host = u.Host // Use the host part only
//		r.URL.Path = r.URL.Path + path
//		r.Header.Set("X-Forwarded-Host", ctx.Request().Host)
//		r.Header.Set("X-Real-IP", ctx.RealIP())
//
//		// Forward the request to the target server
//		proxy.ServeHTTP(ctx.Response(), r)
//		return nil
//	}

//}

func ReverseProxy(target string, port string) echo.HandlerFunc {
	urlStr := "http://" + target + ":" + port
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.Director = func(req *http.Request) {
		// Update the request URL and headers
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host

		path := u.Path
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}

		// Handle the incoming path
		// If you want to keep the incoming path as is:
		req.URL.Path = req.URL.Path + path

		log.Println("[INFO] serving => ", req.URL.Path)

		// If you want to modify the path (e.g., add a prefix):
		// req.URL.Path = "/new-prefix" + req.URL.Path

		req.Host = u.Host
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Real-IP", req.RemoteAddr)
	}

	//return func(ctx echo.Context) error {
	//	// Set up the response and handle the request
	//	proxy.ServeHTTP(ctx.Response(), ctx.Request())
	//	return nil
	//}

	return func(ctx echo.Context) error {
		// Check if the request is a WebSocket upgrade request
		if ctx.Request().Header.Get("Upgrade") == "websocket" {
			// Upgrade the connection to WebSocket
			w := websocket.Upgrader{}
			conn, err := w.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
			if err != nil {
				return err
			}
			if err != nil {
				log.Println("Failed to upgrade connection:", err)
				return err
			}
			defer conn.Close()

			// Create a new request to the target server for the WebSocket connection
			req := ctx.Request()
			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			req.Host = u.Host

			// Establish a WebSocket connection with the target server
			targetConn, _, err := websocket.DefaultDialer.Dial(req.URL.String(), req.Header)
			if err != nil {
				log.Println("Failed to connect to target WebSocket server:", err)
				return err
			}
			defer targetConn.Close()

			// Handle the WebSocket communication
			go func() {
				defer conn.Close()
				for {
					msgType, msg, err := targetConn.ReadMessage()
					if err != nil {
						log.Println("Read error:", err)
						break
					}
					if err := conn.WriteMessage(msgType, msg); err != nil {
						log.Println("Write error:", err)
						break
					}
				}
			}()

			for {
				msgType, msg, err := conn.ReadMessage()
				if err != nil {
					log.Println("Read error:", err)
					break
				}
				if err := targetConn.WriteMessage(msgType, msg); err != nil {
					log.Println("Write error:", err)
					break
				}
			}
		} else {
			// Handle regular HTTP requests
			proxy.ServeHTTP(ctx.Response(), ctx.Request())
		}
		return nil
	}
}
