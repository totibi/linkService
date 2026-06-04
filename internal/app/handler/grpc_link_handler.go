package handler

import (
	"context"
	"linkService/internal/app/domain"

	pb "linkService/generated/go/v1"
	"linkService/internal/app/converter"
	"linkService/internal/app/service"
)

type LinkHandler struct {
	pb.UnimplementedLinkServiceServer
	service *service.AppLinkService
}

func NewLinkHandler(svc *service.AppLinkService) *LinkHandler {
	return &LinkHandler{service: svc}
}

func (h *LinkHandler) CreateLink(ctx context.Context, req *pb.CreateLinkRequest) (*pb.CreateLinkResponse, error) {
	input := converter.ToCreateLinkInput(req)

	shortCode, err := h.service.CreateLink(ctx, input)
	if err != nil {
		return nil, err
	}

	return converter.ToCreateLinkResponse(shortCode), nil
}

func (h *LinkHandler) GetLink(ctx context.Context, req *pb.GetLinkRequest) (*pb.GetLinkResponse, error) {
	link, err := h.service.GetLink(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	return converter.ToPbGetLinkResponse(link), nil
}

func (h *LinkHandler) ListLinks(ctx context.Context, req *pb.ListLinksRequest) (*pb.ListLinksResponse, error) {
	input := domain.ListLinksInput{
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}

	links, total, err := h.service.ListLinks(ctx, input)
	if err != nil {
		return nil, err
	}

	items := make([]*pb.Link, len(links))
	for i, link := range links {
		items[i] = converter.ToPbLink(link)
	}

	return &pb.ListLinksResponse{
		Items: items,
		Total: int32(total),
	}, nil
}

func (h *LinkHandler) DeleteLink(ctx context.Context, req *pb.DeleteLinkRequest) (*pb.DeleteLinkResponse, error) {
	if err := h.service.DeleteLink(ctx, req.ShortCode); err != nil {
		return nil, err
	}
	return &pb.DeleteLinkResponse{}, nil
}

func (h *LinkHandler) GetLinkStats(ctx context.Context, req *pb.GetLinkStatsRequest) (*pb.GetLinkStatsResponse, error) {
	link, err := h.service.GetLinkStats(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}
	return converter.ToPbLinkStatsResponse(link), nil
}
