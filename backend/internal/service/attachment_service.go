package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"propertymanagement/internal/domain"
)

const (
	uploadURLTTL   = 10 * time.Minute
	downloadURLTTL = 15 * time.Minute
)

// AttachmentService issues presigned uploads and downloads. Every id a
// client passes is checked against the caller's ownership first, and a
// foreign id answers ErrNotFound like every other resource.
type AttachmentService struct {
	repo    domain.AttachmentRepository
	storage domain.FileStorage
	log     *slog.Logger
	now     clock
}

func NewAttachmentService(repo domain.AttachmentRepository, storage domain.FileStorage, log *slog.Logger) *AttachmentService {
	return &AttachmentService{repo: repo, storage: storage, log: log, now: systemClock}
}

var _ domain.AttachmentService = (*AttachmentService)(nil)

// cleanFilename keeps only the base name and printable characters, and
// caps the length — it is shown back to the user and used in a
// Content-Disposition header, never as (part of) the storage key.
func cleanFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if len(name) > 200 {
		name = name[:200]
	}
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	return name
}

func (s *AttachmentService) Presign(ctx context.Context, ownerID uuid.UUID, filename, contentType string, sizeBytes int64) (*domain.PresignedUpload, error) {
	ext, ok := domain.AllowedAttachmentContentTypes[contentType]
	if !ok {
		return nil, fmt.Errorf("presign attachment: %w", validationError("content_type", "only JPEG, PNG, WebP images and PDFs can be uploaded"))
	}
	if sizeBytes <= 0 || sizeBytes > domain.MaxAttachmentBytes {
		return nil, fmt.Errorf("presign attachment: %w", validationError("size_bytes", fmt.Sprintf("must be between 1 byte and %d MB", domain.MaxAttachmentBytes>>20)))
	}

	id := uuid.New()
	a := &domain.Attachment{
		ID:          id,
		OwnerID:     ownerID,
		ObjectKey:   fmt.Sprintf("%s/%s%s", ownerID, id, ext),
		Filename:    cleanFilename(filename),
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		Status:      domain.AttachmentStatusPending,
		CreatedAt:   s.now(),
	}
	url, fields, err := s.storage.PresignUpload(ctx, a.ObjectKey, contentType, domain.MaxAttachmentBytes, uploadURLTTL)
	if err != nil {
		return nil, fmt.Errorf("presign attachment: %w", err)
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("presign attachment: %w", err)
	}
	return &domain.PresignedUpload{Attachment: a, URL: url, Fields: fields}, nil
}

func (s *AttachmentService) owned(ctx context.Context, ownerID, id uuid.UUID) (*domain.Attachment, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return a, nil
}

// Confirm verifies the client's upload actually landed (and is within
// limits — the POST policy enforces this at the store, this is the
// belt-and-braces check) before the attachment can be referenced.
func (s *AttachmentService) Confirm(ctx context.Context, ownerID, id uuid.UUID) (*domain.Attachment, error) {
	a, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, fmt.Errorf("confirm attachment %s: %w", id, err)
	}
	if a.Status == domain.AttachmentStatusReady {
		return a, nil
	}
	size, contentType, err := s.storage.Stat(ctx, a.ObjectKey)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("confirm attachment %s: %w", id, validationError("file", "the upload did not complete"))
		}
		return nil, fmt.Errorf("confirm attachment %s: %w", id, err)
	}
	if size <= 0 || size > domain.MaxAttachmentBytes || contentType != a.ContentType {
		if delErr := s.storage.Delete(ctx, a.ObjectKey); delErr != nil {
			s.log.WarnContext(ctx, "failed to delete rejected upload", slog.String("attachment_id", id.String()), slog.Any("error", delErr))
		}
		return nil, fmt.Errorf("confirm attachment %s: %w", id, validationError("file", "the uploaded file was rejected"))
	}
	if err := s.repo.MarkReady(ctx, id, size); err != nil {
		return nil, fmt.Errorf("confirm attachment %s: %w", id, err)
	}
	a.Status, a.SizeBytes = domain.AttachmentStatusReady, size
	return a, nil
}

func (s *AttachmentService) DownloadURL(ctx context.Context, ownerID, id uuid.UUID) (string, string, error) {
	a, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return "", "", fmt.Errorf("download attachment %s: %w", id, err)
	}
	if a.Status != domain.AttachmentStatusReady {
		return "", "", fmt.Errorf("download attachment %s: %w", id, domain.ErrNotFound)
	}
	url, err := s.storage.PresignDownload(ctx, a.ObjectKey, a.Filename, downloadURLTTL)
	if err != nil {
		return "", "", fmt.Errorf("download attachment %s: %w", id, err)
	}
	return url, a.Filename, nil
}

func (s *AttachmentService) Store(ctx context.Context, ownerID uuid.UUID, filename, contentType string, data []byte) (*domain.Attachment, error) {
	ext, ok := domain.AllowedAttachmentContentTypes[contentType]
	if !ok {
		return nil, fmt.Errorf("store attachment: %w", validationError("content_type", "unsupported content type"))
	}
	id := uuid.New()
	a := &domain.Attachment{
		ID:          id,
		OwnerID:     ownerID,
		ObjectKey:   fmt.Sprintf("%s/%s%s", ownerID, id, ext),
		Filename:    cleanFilename(filename),
		ContentType: contentType,
		SizeBytes:   int64(len(data)),
		Status:      domain.AttachmentStatusReady,
		CreatedAt:   s.now(),
	}
	if err := s.storage.Put(ctx, a.ObjectKey, contentType, data); err != nil {
		return nil, fmt.Errorf("store attachment: %w", err)
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("store attachment: %w", err)
	}
	return a, nil
}

func (s *AttachmentService) Read(ctx context.Context, ownerID, id uuid.UUID) (*domain.Attachment, []byte, error) {
	a, err := s.owned(ctx, ownerID, id)
	if err != nil {
		return nil, nil, fmt.Errorf("read attachment %s: %w", id, err)
	}
	data, err := s.storage.Get(ctx, a.ObjectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("read attachment %s: %w", id, err)
	}
	return a, data, nil
}
