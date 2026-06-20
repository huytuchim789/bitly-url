package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bitly-url/internal/entity"
	"bitly-url/internal/middleware"
	"bitly-url/internal/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUseCase struct {
	mock.Mock
}

func (m *mockUseCase) Shorten(ctx context.Context, original string) (*entity.URL, error) {
	args := m.Called(ctx, original)
	if v := args.Get(0); v != nil {
		return v.(*entity.URL), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUseCase) FindByShort(ctx context.Context, short string) (*entity.URL, error) {
	args := m.Called(ctx, short)
	if v := args.Get(0); v != nil {
		return v.(*entity.URL), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUseCase) TrackClick(shortID uuid.UUID, ip, userAgent, referer string) {
	m.Called(shortID, ip, userAgent, referer)
}

func (m *mockUseCase) FindAll(ctx context.Context, limit, offset int) ([]*entity.URL, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*entity.URL), args.Error(1)
}

func (m *mockUseCase) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())
	return r
}

func TestHandler_Shorten_Success(t *testing.T) {
	uc := new(mockUseCase)
	h := NewURLHandler(uc)

	now := time.Now()
	expected := &entity.URL{
		ID:        uuid.New(),
		Short:     "abc123",
		Original:  "https://example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	uc.On("Shorten", mock.Anything, "https://example.com").Return(expected, nil)

	r := setupRouter()
	r.POST("/api/shorten", h.Shorten)

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp entity.URL
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", resp.Short)
	assert.Equal(t, "https://example.com", resp.Original)

	uc.AssertExpectations(t)
}

func TestHandler_Shorten_BadRequest(t *testing.T) {
	uc := new(mockUseCase)
	h := NewURLHandler(uc)

	r := setupRouter()
	r.POST("/api/shorten", h.Shorten)

	body := `{"bad":"request"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Redirect_Success(t *testing.T) {
	uc := new(mockUseCase)
	h := NewURLHandler(uc)

	url := &entity.URL{
		ID:       uuid.New(),
		Short:    "abc123",
		Original: "https://example.com",
	}

	uc.On("FindByShort", mock.Anything, "abc123").Return(url, nil)
	uc.On("TrackClick", url.ID, "192.0.2.1", "test-agent", "").Return()

	r := setupRouter()
	r.GET("/:short", h.Redirect)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))

	uc.AssertExpectations(t)
}

func TestHandler_Redirect_NotFound(t *testing.T) {
	uc := new(mockUseCase)
	h := NewURLHandler(uc)

	uc.On("FindByShort", mock.Anything, "nonexist").Return(nil, errors.ErrNotFound)

	r := setupRouter()
	r.GET("/:short", h.Redirect)

	req := httptest.NewRequest(http.MethodGet, "/nonexist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_List_Success(t *testing.T) {
	uc := new(mockUseCase)
	h := NewURLHandler(uc)

	now := time.Now()
	urls := []*entity.URL{
		{
			ID:        uuid.New(),
			Short:     "abc123",
			Original:  "https://example.com",
			CreatedAt: now,
		},
	}

	uc.On("FindAll", mock.Anything, 50, 0).Return(urls, nil)

	r := setupRouter()
	r.GET("/api/urls", h.List)

	req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []*entity.URL
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, "abc123", resp[0].Short)

	uc.AssertExpectations(t)
}
