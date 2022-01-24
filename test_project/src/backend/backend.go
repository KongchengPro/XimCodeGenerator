package backend

import (
	"github.com/gin-gonic/gin"
)

//goland:noinspection GoUnusedFunction
func mainFunc() {
	r := gin.Default()
	r.Use(Cors())
	r.GET("/concatStrings", func(c *gin.Context) {
		a := c.Query("a")
		b := c.Query("b")
		res := ConcatStrings(a, b)
		c.String(200, res)
	})
	_ = r.Run()
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			// 可将将* 替换为指定的域名
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}

		c.Next()
	}
}

//xim:HandlerFunc("/concatStrings")
func ConcatStrings(a string, b string) string {
	return a + b
}
