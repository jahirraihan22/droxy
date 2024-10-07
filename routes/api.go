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
//
//}

// ReverseProxy function for handling HTTP and WebSocket requests
//func ReverseProxy(target string, port string) echo.HandlerFunc {
//	urlStr := "http://" + target + ":" + port
//	u, err := url.Parse(urlStr)
//	if err != nil {
//		log.Fatalf("Failed to parse target URL: %v", err)
//	}
//
//	proxy := httputil.NewSingleHostReverseProxy(u)
//	path := u.Path
//	if strings.HasPrefix(path, "/") {
//		path = path[1:]
//	}
//	// Customize the proxy's director to modify requests before forwarding
//	proxy.Director = func(req *http.Request) {
//		// Update the request URL and headers
//		req.URL.Scheme = u.Scheme
//		req.URL.Host = u.Host
//
//		// Keep or modify the incoming path
//		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/")
//		req.Host = u.Host
//		req.Header.Set("X-Forwarded-Host", req.Host)
//		req.Header.Set("X-Real-IP", req.RemoteAddr)
//	}
//	//// Modify the response to override the Referrer-Policy
//	//proxy.ModifyResponse = func(res *http.Response) error {
//	//	// Override Referrer-Policy header
//	//	res.Header.Set("Referrer-Policy", "no-referrer-when-downgrade")
//	//	// You can also disable the referrer policy entirely:
//	//	// res.Header.Set("Referrer-Policy", "no-referrer")
//	//	return nil
//	//}
//
//	// WebSocket upgrader
//	upgdr := websocket.Upgrader{
//		CheckOrigin: func(r *http.Request) bool {
//			// Optionally validate the request origin (can be used for security)
//			return true
//		},
//	}
//
//	return func(ctx echo.Context) error {
//		// Check if the request is a WebSocket upgrade request
//		if websocket.IsWebSocketUpgrade(ctx.Request()) {
//			// Upgrade the connection to WebSocket
//			conn, err := upgdr.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
//			if err != nil {
//				log.Println("Failed to upgrade to WebSocket:", err)
//				return err
//			}
//			defer conn.Close()
//
//			// Construct the WebSocket URL
//			targetWSURL := "ws://" + target + ":" + port + path
//
//			// Remove headers that should not be forwarded to the target WebSocket server
//			requestHeader := ctx.Request().Header.Clone()
//			requestHeader.Del("Connection")
//			requestHeader.Del("Upgrade")
//
//			// Create a WebSocket dialer to connect to the target server
//			targetConn, _, err := websocket.DefaultDialer.Dial(targetWSURL, requestHeader)
//			if err != nil {
//				log.Println("Failed to connect to target WebSocket server:", err)
//				return err
//			}
//			defer targetConn.Close()
//
//			// Handle bidirectional WebSocket communication
//			go func() {
//				defer conn.Close()
//				for {
//					msgType, msg, err := targetConn.ReadMessage()
//					if err != nil {
//						log.Println("Error reading from target:", err)
//						break
//					}
//					if err := conn.WriteMessage(msgType, msg); err != nil {
//						log.Println("Error writing to client:", err)
//						break
//					}
//				}
//			}()
//
//			for {
//				msgType, msg, err := conn.ReadMessage()
//				if err != nil {
//					log.Println("Error reading from client:", err)
//					break
//				}
//				if err := targetConn.WriteMessage(msgType, msg); err != nil {
//					log.Println("Error writing to target:", err)
//					break
//				}
//			}
//		} else {
//			// Handle regular HTTP requests
//			proxy.ServeHTTP(ctx.Response(), ctx.Request())
//		}
//		return nil
//	}

func ReverseProxy(target string, port string) echo.HandlerFunc {
	urlStr := "http://" + target + ":" + port
	u, err := url.Parse(urlStr)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	// Customize the proxy's director to modify requests before forwarding
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
		req.Host = u.Host
		req.URL.Path = req.URL.Path // Keep the original path

		if websocket.IsWebSocketUpgrade(req) {
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
		}

	}

	// WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins, adjust as needed
		},
	}

	return func(ctx echo.Context) error {
		// Check if the request is a WebSocket upgrade request
		if websocket.IsWebSocketUpgrade(ctx.Request()) {
			// Upgrade the connection to WebSocket
			conn, err := upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
			if err != nil {
				log.Println("Failed to upgrade to WebSocket:", err)
				return err
			}
			defer conn.Close()

			// Prepare a new request to the target server
			targetWSURL := "ws://" + target + ":" + port + ctx.Request().URL.Path
			targetReq, err := http.NewRequest("GET", targetWSURL, nil)
			if err != nil {
				log.Println("Failed to create target WebSocket request:", err)
				return err
			}

			//// Forward headers, but skip WebSocket-specific headers
			//for key, values := range ctx.Request().Header {
			//	for _, value := range values {
			//		if key == "Connection" || key == "Upgrade" ||
			//			key == "Sec-WebSocket-Key" || key == "Sec-WebSocket-Version" ||
			//			key == "Sec-WebSocket-Extensions" || key == "Sec-WebSocket-Protocol" {
			//			continue // Skip duplicate headers
			//		}
			//		targetReq.Header.Add(key, value)
			//	}
			//}

			// Create a WebSocket dialer to connect to the target server
			targetConn, _, err := websocket.DefaultDialer.Dial(targetReq.URL.String(), targetReq.Header)
			if err != nil {
				log.Println("Failed to connect to target WebSocket server:", err)
				return err
			}
			defer targetConn.Close()

			// Handle bidirectional WebSocket communication
			go func() {
				for {
					msgType, msg, err := targetConn.ReadMessage()
					if err != nil {
						log.Println("Error reading from target:", err)
						break
					}
					if err := conn.WriteMessage(msgType, msg); err != nil {
						log.Println("Error writing to client:", err)
						break
					}
				}
			}()

			for {
				msgType, msg, err := conn.ReadMessage()
				if err != nil {
					log.Println("Error reading from client:", err)
					break
				}
				if err := targetConn.WriteMessage(msgType, msg); err != nil {
					log.Println("Error writing to target:", err)
					break
				}
			}
			return nil // Return nil to prevent writing further
		} else {
			// Handle regular HTTP requests
			proxy.ServeHTTP(ctx.Response(), ctx.Request())
		}
		return nil
	}
}
