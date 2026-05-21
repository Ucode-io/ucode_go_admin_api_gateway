package v1

import (
	"context"
	"net/http"
	"strings"
	"time"

	cs "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
)

const (
	mfeShortLinkRedisPrefix = "mfe:s:"
	mfeShortLinkRedisTTL    = 30 * 24 * time.Hour
)

func mfeShortURL(base, slug string) string {
	return strings.TrimRight(base, "/") + "/p/" + slug
}

func (h *HandlerV1) RedirectShortURL(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.Status(http.StatusNotFound)
		return
	}

	ctx := c.Request.Context()

	if h.centralRedis != nil {
		if url, err := h.centralRedis.Get(ctx, mfeShortLinkRedisPrefix+slug).Result(); err == nil {
			c.Redirect(http.StatusMovedPermanently, url)
			return
		}
	}

	link, err := h.companyServices.MfeShortLink().GetBySlug(ctx, &cs.MfeShortLinkSlugReq{Slug: slug})
	if err != nil || link.GetUrl() == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "short link not found"})
		return
	}

	if h.centralRedis != nil {
		go func() {
			_ = h.centralRedis.Set(context.Background(), mfeShortLinkRedisPrefix+slug, link.GetUrl(), mfeShortLinkRedisTTL).Err()
		}()
	}

	c.Redirect(http.StatusMovedPermanently, link.GetUrl())
}
