package usecase

import (
	"context"
	"testing"
	"time"

	"bitly-url/internal/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClickRepo struct {
	mock.Mock
}

func (m *mockClickRepo) BatchInsert(ctx context.Context, clicks []*entity.Click) error {
	return m.Called(ctx, clicks).Error(0)
}

type mockCache struct {
	mock.Mock
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return m.Called(ctx, key, value, ttl).Error(0)
}

func (m *mockCache) Del(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

func (m *mockCache) Incr(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return m.Called(ctx, key, ttl).Error(0)
}

func (m *mockCache) Ping(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockCache) Close() error {
	return m.Called().Error(0)
}

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) Save(ctx context.Context, url *entity.URL) error {
	return m.Called(ctx, url).Error(0)
}

func (m *mockRepo) FindByID(ctx context.Context, id string) (*entity.URL, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*entity.URL), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepo) FindByShort(ctx context.Context, short string) (*entity.URL, error) {
	args := m.Called(ctx, short)
	if v := args.Get(0); v != nil {
		return v.(*entity.URL), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepo) FindAll(ctx context.Context, limit, offset int) ([]*entity.URL, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*entity.URL), args.Error(1)
}

func (m *mockRepo) IncrementClicks(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func TestShorten_ValidURL_Success(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	cache.On("Get", mock.Anything, mock.Anything).Return("", assert.AnError)
	repo.On("FindByShort", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	repo.On("Save", mock.Anything, mock.Anything).Return(nil)
	cache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	url, err := uc.Shorten(context.Background(), "https://example.com/valid-url")

	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.NotEmpty(t, url.Short)
	assert.Equal(t, "https://example.com/valid-url", url.Original)
	cache.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestShorten_InvalidURL_TooLong(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	longURL := "https://example.com/" + string(make([]byte, 2048))

	url, err := uc.Shorten(context.Background(), longURL)
	assert.Error(t, err)
	assert.Nil(t, url)
}

func TestShorten_InvalidURL_BadScheme(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	url, err := uc.Shorten(context.Background(), "ftp://example.com")
	assert.Error(t, err)
	assert.Nil(t, url)
}

func TestShorten_PrivateIP_Blocked(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	url, err := uc.Shorten(context.Background(), "http://localhost:8080/evil")
	assert.Error(t, err)
	assert.Nil(t, url)

	url, err = uc.Shorten(context.Background(), "http://127.0.0.1/evil")
	assert.Error(t, err)
	assert.Nil(t, url)

	url, err = uc.Shorten(context.Background(), "http://192.168.1.1/evil")
	assert.Error(t, err)
	assert.Nil(t, url)
}

func TestFindByShort_CacheHit_Success(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	cache.On("Get", mock.Anything, "short:abc123").Return("https://example.com", nil)
	url, err := uc.FindByShort(context.Background(), "abc123")

	assert.NoError(t, err)
	assert.Equal(t, "abc123", url.Short)
	assert.Equal(t, "https://example.com", url.Original)
	cache.AssertExpectations(t)
}

func TestFindByShort_CacheMiss_DBHit_Success(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	now := time.Now()
	existingURL := &entity.URL{
		ID:        uuid.New(),
		Short:     "abc123",
		Original:  "https://example.com",
		Clicks:    5,
		CreatedAt: now,
		UpdatedAt: now,
	}

	cache.On("Get", mock.Anything, "short:abc123").Return("", assert.AnError)
	repo.On("FindByShort", mock.Anything, "abc123").Return(existingURL, nil)
	cache.On("Set", mock.Anything, "short:abc123", "https://example.com", cacheTTL).Return(nil)

	url, err := uc.FindByShort(context.Background(), "abc123")

	assert.NoError(t, err)
	assert.Equal(t, "abc123", url.Short)
	assert.Equal(t, "https://example.com", url.Original)
	assert.Equal(t, int64(5), url.Clicks)
}

func TestFindByShort_NotFound(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	cache.On("Get", mock.Anything, "short:nonexist").Return("", assert.AnError)
	repo.On("FindByShort", mock.Anything, "nonexist").Return(nil, assert.AnError)

	url, err := uc.FindByShort(context.Background(), "nonexist")

	assert.Error(t, err)
	assert.Nil(t, url)
}

func TestFindByShort_Expired(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	expiresAt := time.Now().Add(-1 * time.Hour)
	expiredURL := &entity.URL{
		ID:        uuid.New(),
		Short:     "expired",
		Original:  "https://example.com",
		ExpiresAt: &expiresAt,
	}

	cache.On("Get", mock.Anything, "short:expired").Return("", assert.AnError)
	repo.On("FindByShort", mock.Anything, "expired").Return(expiredURL, nil)

	url, err := uc.FindByShort(context.Background(), "expired")

	assert.Error(t, err)
	assert.Nil(t, url)
}

func TestFindAll_DefaultLimit(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	repo.On("FindAll", mock.Anything, 50, 0).Return([]*entity.URL{}, nil)

	urls, err := uc.FindAll(context.Background(), 0, 0)

	assert.NoError(t, err)
	assert.Empty(t, urls)
}

func TestFindAll_ExceedsMaxLimit(t *testing.T) {
	repo := new(mockRepo)
	cache := new(mockCache)
	clickRepo := new(mockClickRepo)
	uc := NewURLUseCase(repo, clickRepo, cache, context.Background())

	repo.On("FindAll", mock.Anything, 50, 0).Return([]*entity.URL{}, nil)

	urls, err := uc.FindAll(context.Background(), 200, 0)

	assert.NoError(t, err)
	assert.Empty(t, urls)
}
