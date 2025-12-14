package handler

import (
	"context"

	"vasset/downloader-service/internal/models"
	"vasset/downloader-service/internal/service"
	pb "vasset/downloader-service/proto"
)

// GRPCHandler gRPC 处理器
type GRPCHandler struct {
	pb.UnimplementedDownloaderServiceServer
	service *service.DownloaderService
}

// NewGRPCHandler 创建 gRPC 处理器
func NewGRPCHandler(svc *service.DownloaderService) *GRPCHandler {
	return &GRPCHandler{service: svc}
}

// GetTaskStatus 获取任务状态
func (h *GRPCHandler) GetTaskStatus(ctx context.Context, req *pb.GetTaskStatusRequest) (*pb.GetTaskStatusResponse, error) {
	record, err := h.service.GetTaskStatus(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	if record == nil {
		return &pb.GetTaskStatusResponse{
			TaskId:     req.TaskId,
			StatusText: "not_found",
		}, nil
	}

	resp := &pb.GetTaskStatusResponse{
		TaskId:     record.TaskID,
		Status:     int32(record.Status),
		StatusText: models.StatusText[record.Status],
		CreatedAt:  record.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if record.FilePath.Valid {
		resp.FilePath = record.FilePath.String
	}
	if record.ErrorMessage.Valid {
		resp.ErrorMessage = record.ErrorMessage.String
	}
	if record.CompletedAt.Valid {
		resp.CompletedAt = record.CompletedAt.Time.Format("2006-01-02T15:04:05Z")
	}

	return resp, nil
}

// GetDownloadHistory 获取下载历史
func (h *GRPCHandler) GetDownloadHistory(ctx context.Context, req *pb.GetDownloadHistoryRequest) (*pb.GetDownloadHistoryResponse, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var statusPtr *int
	if req.Status != 0 {
		status := int(req.Status)
		statusPtr = &status
	}

	records, total, err := h.service.GetDownloadHistory(ctx, req.UserId, page, pageSize, statusPtr)
	if err != nil {
		return nil, err
	}

	pbRecords := make([]*pb.DownloadRecord, 0, len(records))
	for _, record := range records {
		pbRecord := &pb.DownloadRecord{
			Id:        record.ID,
			TaskId:    record.TaskID,
			UserId:    record.UserID,
			Url:       record.URL,
			Platform:  record.Platform,
			Title:     record.Title,
			Mode:      record.Mode,
			Quality:   record.Quality,
			Status:    int32(record.Status),
			CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if record.FilePath.Valid {
			pbRecord.FilePath = record.FilePath.String
		}
		if record.FileName.Valid {
			pbRecord.FileName = record.FileName.String
		}
		if record.FileSize.Valid {
			pbRecord.FileSize = record.FileSize.Int64
		}
		if record.FileHash.Valid {
			pbRecord.FileHash = record.FileHash.String
		}
		if record.ErrorMessage.Valid {
			pbRecord.ErrorMessage = record.ErrorMessage.String
		}
		if record.CompletedAt.Valid {
			pbRecord.CompletedAt = record.CompletedAt.Time.Format("2006-01-02T15:04:05Z")
		}
		if record.ExpireAt.Valid {
			pbRecord.ExpireAt = record.ExpireAt.Time.Format("2006-01-02T15:04:05Z")
		}

		pbRecords = append(pbRecords, pbRecord)
	}

	return &pb.GetDownloadHistoryResponse{
		Records:  pbRecords,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

// CancelTask 取消任务
func (h *GRPCHandler) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.CancelTaskResponse, error) {
	success, err := h.service.CancelTask(ctx, req.TaskId, req.UserId)
	if err != nil {
		return nil, err
	}

	var message string
	if success {
		message = "Task cancelled successfully"
	} else {
		message = "Task cannot be cancelled"
	}

	return &pb.CancelTaskResponse{
		Success: success,
		Message: message,
	}, nil
}
