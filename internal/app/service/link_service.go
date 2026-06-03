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

	"google.golang.org/protobuf/types/known/timestamppb"
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

// генерация случайно кода на основе случайного выбора 8-ми символов.
// Не продуктовое решение, т.к. есть вероятность коллизей
func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func validateURL(rawURL string) error {
	if rawURL == "" {
		return domain.ErrInvalidURL
	}
	// Защита от переполнения полей БД / DoS
	if len(rawURL) > 2048 {
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

func (s *AppLinkService) CreateLink(ctx context.Context, req *pb.CreateLinkRequest) (*pb.CreateLinkResponse, error) {
	if req.Url == "" {
		return nil, domain.ErrInvalidURL
	}

	if err := validateURL(req.Url); err != nil {
		return nil, err
	}

	u, _ := url.Parse(req.Url)
	normalizedURL := u.String()

	link := &domain.Link{
		ShortCode:   generateShortCode(),
		OriginalURL: normalizedURL,
		Visits:      0,
	}

	if _, err := s.repo.Create(ctx, link); err != nil {
		return nil, err
	}

	s.cache.set(link.ShortCode, link)

	return &pb.CreateLinkResponse{
		ShortCode: link.ShortCode,
	}, nil
}

func (s *AppLinkService) GetLink(ctx context.Context, req *pb.GetLinkRequest) (*pb.GetLinkResponse, error) {

	if cached, ok := s.cache.get(req.ShortCode); ok {
		slog.Info("Cache HIT", "short_code", req.ShortCode)
		_ = s.repo.IncrementVisits(ctx, req.ShortCode)
		cached.Visits++
		s.cache.set(req.ShortCode, cached)
		return &pb.GetLinkResponse{
			Url:    cached.OriginalURL,
			Visits: int32(cached.Visits),
		}, nil
	}

	slog.Info("Cache MISS", "short_code", req.ShortCode)
	link, err := s.repo.GetByShortCode(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}
	link.Visits++
	s.cache.set(req.ShortCode, link)

	// Увеличиваем счётчик
	errIncrement := s.repo.IncrementVisits(ctx, req.ShortCode)
	if errIncrement != nil {
		return nil, errIncrement
	}

	return &pb.GetLinkResponse{
		Url:    link.OriginalURL,
		Visits: int32(link.Visits),
	}, nil
}

func (s *AppLinkService) ListLinks(ctx context.Context, req *pb.ListLinksRequest) (*pb.ListLinksResponse, error) {
	limit := 20
	if req.Limit > 0 {
		limit = int(req.Limit)
	}

	links, total, err := s.repo.List(ctx, limit, int(req.Offset))
	if err != nil {
		return nil, err
	}

	items := make([]*pb.Link, len(links))
	for i, l := range links {
		items[i] = &pb.Link{
			Id:          int32(l.ID),
			ShortCode:   l.ShortCode,
			OriginalUrl: l.OriginalURL,
			CreatedAt:   timestamppb.New(l.CreatedAt),
			Visits:      int32(l.Visits),
		}
	}

	return &pb.ListLinksResponse{
		Items: items,
		Total: int32(total),
	}, nil
}

func (s *AppLinkService) DeleteLink(ctx context.Context, req *pb.DeleteLinkRequest) (*pb.DeleteLinkResponse, error) {
	if err := s.repo.Delete(ctx, req.ShortCode); err != nil {
		return nil, err
	}
	s.cache.delete(req.ShortCode)
	return &pb.DeleteLinkResponse{}, nil
}

func (s *AppLinkService) GetLinkStats(ctx context.Context, req *pb.GetLinkStatsRequest) (*pb.GetLinkStatsResponse, error) {
	link, err := s.repo.GetByShortCode(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	return &pb.GetLinkStatsResponse{
		ShortCode: link.ShortCode,
		Url:       link.OriginalURL,
		Visits:    int32(link.Visits),
		CreatedAt: timestamppb.New(link.CreatedAt),
	}, nil
}

func (s *AppLinkService) Close() {
	s.cache.Stop()
}
