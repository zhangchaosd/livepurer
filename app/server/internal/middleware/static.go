package middleware

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"path"
)

// Static returns a middleware handler that serves static files in the given directory.
func Static(dir string) gin.HandlerFunc {
	return static.Serve("/", static.LocalFile(dir, true))
}

// NoRoute SPA router
func NoRoute(dir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.File(path.Join(dir, "index.html"))
	}
}
