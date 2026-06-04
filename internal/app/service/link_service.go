package service

import (
	"context"
	pb "linkService/generated/go/v1"
	"linkService/internal/app/domain"
	"log/slog"
	"math/rand"
	"net/url"
	"strings"
	"time"
)

type AppLinkService struct {
	pb.UnimplementedLinkServiceServer
	repo  domain.LinkRepository
	cache *linkCache
}

func NewAppLinkService(repo domain.LinkRepository) *AppLinkService {
	return &AppLinkService{
		repo:  repo,
		cache: newLinkCache(100, 1*time.Minute),
	}
}

// генерация случайно кода на основе случайного выбора символов.
// Не продуктовое решение, т.к. есть вероятность коллизей
func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const shortCodeSize = 8
	b := make([]byte, shortCodeSize)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func validateURL(rawURL string) error {
	if rawURL == "" {
		return domain.ErrInvalidURL
	}
	const dbMaxTextSize = 2048
	if len(rawURL) > dbMaxTextSize {
		return domain.ErrInvalidURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return domain.ErrInvalidURL
	}

	// Разрешаем только безопасные схемы
	if u.Scheme != "http" && u.Scheme != "https" {
		return domain.ErrInvalidURL
	}

	// Хост обязателен
	if u.Host == "" {
		return domain.ErrInvalidURL
	}

	// Защита от SSRF: блокируем локальные/внутренние адреса
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" {
		return domain.ErrInvalidURL
	}

	return nil
}

func (s *AppLinkService) CreateLink(ctx context.Context, req domain.CreateLinkInput) (string, error) {
	if req.Url == "" {
		return "", domain.ErrInvalidURL
	}

	if err := validateURL(req.Url); err != nil {
		return "", err
	}

	u, _ := url.Parse(req.Url)
	normalizedURL := u.String()

	link := &domain.Link{
		ShortCode:   generateShortCode(),
		OriginalURL: normalizedURL,
		Visits:      0,
	}

	if _, err := s.repo.Create(ctx, link); err != nil {
		return "", err
	}

	s.cache.set(link.ShortCode, link)

	return link.ShortCode, nil
}

func (s *AppLinkService) GetLink(ctx context.Context, shortCode string) (*domain.Link, error) {

	if cached, ok := s.cache.get(shortCode); ok {
		slog.Info("Cache HIT", "short_code", shortCode)
		_ = s.repo.IncrementVisits(ctx, shortCode)
		cached.IncementVisits()
		s.cache.set(shortCode, cached)
		return cached, nil
	}

	slog.Info("Cache MISS", "short_code", shortCode)
	link, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	link.IncementVisits()
	s.cache.set(shortCode, link)

	// Увеличиваем счётчик
	errIncrement := s.repo.IncrementVisits(ctx, shortCode)
	if errIncrement != nil {
		return nil, errIncrement
	}

	return link, nil
}

func (s *AppLinkService) ListLinks(ctx context.Context, req domain.ListLinksInput) ([]*domain.Link, int, error) {
	limit := 20
	if req.Limit > 0 {
		limit = int(req.Limit)
	}

	links, total, err := s.repo.List(ctx, limit, int(req.Offset))
	if err != nil {
		return nil, 0, err
	}

	return links, total, nil
}

func (s *AppLinkService) DeleteLink(ctx context.Context, shortCode string) error {
	if err := s.repo.Delete(ctx, shortCode); err != nil {
		return err
	}
	s.cache.delete(shortCode)
	return nil
}

func (s *AppLinkService) GetLinkStats(ctx context.Context, shortCode string) (*domain.Link, error) {
	link, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	return link, nil
}

func (s *AppLinkService) Close() {
	s.cache.Stop()
}
