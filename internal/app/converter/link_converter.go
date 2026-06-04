package converter

import (
	pb "linkService/generated/go/v1"
	"linkService/internal/app/domain"
	linkdto "linkService/internal/app/dto/link"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToCreateLinkInput(pbReq *pb.CreateLinkRequest) domain.CreateLinkInput {
	return domain.CreateLinkInput{
		Url: pbReq.Url,
	}
}

func ToCreateLinkResponse(shortCode string) *pb.CreateLinkResponse {
	return &pb.CreateLinkResponse{ShortCode: shortCode}
}

func ToDomainLink(pbReq *pb.GetLinkRequest) string {
	return pbReq.ShortCode
}

func ToLinkResponse(link *domain.Link) *linkdto.LinkResponse {
	return &linkdto.LinkResponse{
		ID:          link.ID,
		ShortCode:   link.ShortCode,
		OriginalURL: link.OriginalURL,
		CreatedAt:   link.CreatedAt,
		Visits:      link.Visits,
	}
}

func ToPbLink(link *domain.Link) *pb.Link {
	return &pb.Link{
		Id:          int32(link.ID),
		ShortCode:   link.ShortCode,
		OriginalUrl: link.OriginalURL,
		CreatedAt:   timestamppb.New(link.CreatedAt),
		Visits:      int32(link.Visits),
	}
}

func ToPbGetLinkResponse(link *domain.Link) *pb.GetLinkResponse {
	return &pb.GetLinkResponse{
		Url:    link.OriginalURL,
		Visits: int32(link.Visits),
	}
}

func ToPbLinkStatsResponse(link *domain.Link) *pb.GetLinkStatsResponse {
	return &pb.GetLinkStatsResponse{
		ShortCode: link.ShortCode,
		Url:       link.OriginalURL,
		Visits:    int32(link.Visits),
		CreatedAt: timestamppb.New(link.CreatedAt),
	}
}
