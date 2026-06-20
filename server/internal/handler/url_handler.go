package handler

import (
	"net/http"

	"bitly-url/internal/pkg/errors"
	"bitly-url/internal/usecase"

	"github.com/gin-gonic/gin"
)

type URLHandler struct {
	uc *usecase.URLUseCase
}

func NewURLHandler(uc *usecase.URLUseCase) *URLHandler {
	return &URLHandler{uc: uc}
}

type ShortenRequest struct {
	URL string `json:"url" binding:"required,url"`
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
// @Failure      500  {object}  map[string]string
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
	c.JSON(http.StatusCreated, url)
}

// Redirect   godoc
// @Summary      Redirect to original URL
// @Description  Redirect to the original URL using the short code
// @Tags         urls
// @Param        short path string true "Short code"
// @Success      301
// @Failure      404  {object}  map[string]string
// @Router       /{short} [get]
func (h *URLHandler) Redirect(c *gin.Context) {
	short := c.Param("short")
	url, err := h.uc.FindByShort(c.Request.Context(), short)
	if err != nil {
		c.Error(err)
		return
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
// @Description  Get all shortened URLs
// @Tags         urls
// @Produce      json
// @Success      200  {array}   entity.URL
// @Failure      500  {object}  map[string]string
// @Router       /api/urls [get]
func (h *URLHandler) List(c *gin.Context) {
	urls, err := h.uc.FindAll(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, urls)
}
