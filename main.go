package main

import (
	"github.com/gin-gonic/gin"
	"html/template"
)

func SetCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, e := c.Request.Cookie("style")
		if e == nil { // Refresh Cookie
			c.SetCookie(cookie.Name, cookie.Value, 604800, cookie.Path, cookie.Domain, cookie.Secure, cookie.HttpOnly)
		} else { // Set New Cookie
			c.SetCookie("style", "normal", 604800, "/", "localhost", false, true)
		}
		c.Next()
	}
}
func main() {
	//connect to database
	Connect()
	defer db.Close()   //close database
	_ = seg.LoadDict() //load dict
	r := gin.Default()
	r.Use(SetCookie())
	r.SetFuncMap(template.FuncMap{
		"unescaped": unescaped,
	})
	// Load Resources
	r.Static("/bin", "./bin")
	r.LoadHTMLGlob("templates/**/*")
	// Router
	r.GET("/", index)
	r.GET("/datapack/:id", datapack)
	r.GET("/random/datapack", datapackRand)
	r.GET("/search", search)
	r.GET("/author", authorList)
	r.GET("/author/:id", author)
	r.GET("/random/author", authorRand)
	r.GET("/tag", tagList)
	r.GET("/tag/:id", tag)
	r.GET("/random/tag", tagRand)
	r.GET("/language", language)
	r.GET("/guide", guide)
	r.GET("/about", about)
	r.POST("/thumb", thumbByID)
	_ = r.Run(":4080")
}
