package handler

import (
	"context"
	"net/http"
	"strconv"

	"bitly-url/internal/entity"
	"bitly-url/internal/metrics"
	"bitly-url/internal/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type URLUseCase interface {
	Shorten(ctx context.Context, original string) (*entity.URL, error)
	FindByShort(ctx context.Context, short string) (*entity.URL, error)
	FindAll(ctx context.Context, limit, offset int) ([]*entity.URL, error)
	TrackClick(shortID uuid.UUID, ip, userAgent, referer string)
	Delete(ctx context.Context, id string) error
}

type URLHandler struct {
	uc URLUseCase
}

func NewURLHandler(uc URLUseCase) *URLHandler {
	return &URLHandler{uc: uc}
}

type ShortenRequest struct {
	URL       string `json:"url" binding:"required,url"`
	ExpiresIn string `json:"expires_in,omitempty"` // optional: "24h", "7d", "30d"
}

// Shorten    godoc
// @Summary      Shorten a URL
// @Description  Create a short URL from a long URL
// @Tags         urls
// @Accept       json
// @Produce      json
// @Param        request body ShortenRequest true "URL to shorten"
// @Success      201  {object}  entity.URL
// @Failure      400  {object}  map[string]string
// @Failure      429  {object}  map[string]string
// @Router       /api/shorten [post]
func (h *URLHandler) Shorten(c *gin.Context) {
	var req ShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.ErrBadRequest)
		return
	}

	url, err := h.uc.Shorten(c.Request.Context(), req.URL)
	if err != nil {
		c.Error(err)
		return
	}
	metrics.URLShortenTotal.Inc()
	c.JSON(http.StatusCreated, url)
}

// Redirect   godoc
// @Summary      Redirect to original URL
// @Description  Redirect to the original URL using the short code
// @Tags         urls
// @Param        short path string true "Short code"
// @Success      302
// @Failure      404  {object}  map[string]string
// @Failure      410  {object}  map[string]string
// @Router       /{short} [get]
func (h *URLHandler) Redirect(c *gin.Context) {
	short := c.Param("short")
	url, err := h.uc.FindByShort(c.Request.Context(), short)
	if err != nil {
		c.Error(err)
		return
	}

	if url.ID != uuid.Nil {
		h.uc.TrackClick(
			url.ID,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			c.GetHeader("Referer"),
		)
	}

	c.Redirect(http.StatusFound, url.Original)
}

// GetByShort godoc
// @Summary      Get URL by short code
// @Description  Get the original URL for a short code
// @Tags         urls
// @Produce      json
// @Param        short path string true "Short code"
// @Success      200  {object}  entity.URL
// @Failure      404  {object}  map[string]string
// @Router       /api/urls/{short} [get]
func (h *URLHandler) GetByShort(c *gin.Context) {
	short := c.Param("short")
	url, err := h.uc.FindByShort(c.Request.Context(), short)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, url)
}

// List       godoc
// @Summary      List all URLs
// @Description  Get all shortened URLs with pagination
// @Tags         urls
// @Produce      json
// @Param        limit query int false "Limit (default 50)"
// @Param        offset query int false "Offset (default 0)"
// @Success      200  {array}   entity.URL
// @Failure      500  {object}  map[string]string
// @Router       /api/urls [get]
func (h *URLHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil {
		limit = 50
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		offset = 0
	}

	urls, err := h.uc.FindAll(c.Request.Context(), limit, offset)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, urls)
}
