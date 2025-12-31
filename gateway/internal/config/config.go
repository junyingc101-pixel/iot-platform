package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"gitee/getcharzp/iot-platform/gateway/internal/config"
	"gitee/getcharzp/iot-platform/gateway/internal/svc"
	"gitee/getcharzp/iot-platform/user/rpc/types/user"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建服务上下文
	ctx := svc.NewServiceContext(c)

	r := gin.Default()

	// 设置路由
	setupRoutes(r, ctx)

	fmt.Printf("API Gateway is running on port %d\n", c.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", c.Port), r))
}

func setupRoutes(r *gin.Engine, ctx *svc.ServiceContext) {
	// 反向代理中间件
	r.Any("/admin/*path", reverseProxy("http://127.0.0.1:14010"))
	r.Any("/user/*path", reverseProxy("http://127.0.0.1:14000"))
	r.Any("/open/*path", reverseProxy("http://127.0.0.1:16001"))
	r.Any("/device/*path", reverseProxy("http://127.0.0.1:15001"))

	// 需要认证的路由示例
	authorized := r.Group("/")
	authorized.Use(authMiddleware(ctx))
	{
		authorized.Any("/protected/admin/*path", reverseProxy("http://127.0.0.1:14010"))
		authorized.Any("/protected/user/*path", reverseProxy("http://127.0.0.1:14000"))
		authorized.Any("/protected/open/*path", reverseProxy("http://127.0.0.1:16001"))
		authorized.Any("/protected/device/*path", reverseProxy("http://127.0.0.1:15001"))
	}
}

func reverseProxy(target string) gin.HandlerFunc {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// 认证中间件
func authMiddleware(ctx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization头
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// 调用user.rpc服务进行认证
		authReq := &user.UserAuthRequest{Token: strings.TrimPrefix(token, "Bearer ")}

		// 设置超时
		gctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 执行认证
		authResp, err := ctx.UserRpc.Auth(gctx, authReq)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("userIdentity", authResp.Identity)
		c.Set("userId", authResp.Id)
		c.Next()
	}
}
