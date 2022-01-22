package backend

import (
	"github.com/gin-gonic/gin"
)

//goland:noinspection GoUnusedFunction
func mainFunc() {
	r := gin.Default()
	r.GET("/concatStrings", func(c *gin.Context) {
		a := c.Query("a")
		b := c.Query("b")
		res := concatStrings(a, b)
		c.String(200, res)
	})
	_ = r.Run()
}

//xim:handler("/concatStrings")
func concatStrings(a string, b string) string {
	return a + b
}
