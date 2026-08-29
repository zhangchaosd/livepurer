package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/pkg/request"
	"io"
	"net/http"
)

func Proxy(c *gin.Context) {
	target, err := request.ValidatePublicURL(c.GetHeader("PL-URL"), "http", "https")
	if err != nil {
		c.String(http.StatusBadRequest, "invalid proxy target: %v", err)
		return
	}
	req, err := http.NewRequestWithContext(c, c.Request.Method, target.String(), c.Request.Body)

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)
	req.Header = c.Request.Header.Clone()

	req.Header.Del("PL-URL")
	client := *request.HTTP().Client()
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer resp.Body.Close()

	for k := range resp.Header {
		for j := range resp.Header[k] {
			c.Header(k, resp.Header[k][j])
		}
	}

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}
