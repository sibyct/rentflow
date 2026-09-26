package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"propertymanagement/internal/domain"
	"propertymanagement/internal/transport/http/dto"
	"propertymanagement/internal/transport/http/middleware"
	"propertymanagement/internal/transport/http/response"
)

type LeaseHandler struct {
	svc domain.LeaseService
}

func NewLeaseHandler(svc domain.LeaseService) *LeaseHandler {
	return &LeaseHandler{svc: svc}
}

// Create backs the "unit -> terms" lease-creation flow — unitId comes
// from the URL, never a client-supplied body field, same rationale as
// UnitHandler.Create taking propertyId from the URL.
func (h *LeaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("create lease: %w", domain.ErrUnauthorized))
		return
	}

	unitID, err := parseUnitIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.CreateLeaseRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain(unitID)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("create lease: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.CreateLease(r.Context(), claims.UserID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewLeaseResponse(l))
}

func (h *LeaseHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("get lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.GetLease(r.Context(), id, claims.UserID, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseWithUnitPropertyResponse(l))
}

// List backs the global Leases page — every lease across every
// property the caller owns.
func (h *LeaseHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list leases: %w", domain.ErrUnauthorized))
		return
	}

	opts, err := parseLeaseListOptions(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	opts.PropertyAccess, _ = middleware.PropertyAccessFromContext(r.Context())

	leases, total, err := h.svc.ListLeasesForOwner(r.Context(), claims.UserID, opts)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, dto.NewLeaseWithUnitPropertyListResponse(leases), dto.LeaseListMeta{
		Total:  total,
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (h *LeaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("update lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.UpdateLeaseRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}

	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("update lease: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.UpdateLease(r.Context(), id, claims.UserID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseResponse(l))
}

func (h *LeaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	if err := h.svc.DeleteLease(r.Context(), id, claims.UserID, access); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListRentHistory backs the Lease Detail page's Rent History tab (R4).
func (h *LeaseHandler) ListRentHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list rent history: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	history, err := h.svc.ListRentHistory(r.Context(), id, claims.UserID, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseRentHistoryListResponse(history))
}

// ChangeRent is the only way a client can alter an active lease's
// effective rent (R3/R4/R6).
func (h *LeaseHandler) ChangeRent(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("change rent: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.ChangeRentRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("change rent: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.ChangeRent(r.Context(), id, claims.UserID, claims.ActorID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseResponse(l))
}

// GenerateRenewal turns an Accepted renewal into a new, real Lease
// record (R7/R8/R9).
func (h *LeaseHandler) GenerateRenewal(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("generate renewal lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.GenerateRenewalRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("generate renewal lease: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.GenerateRenewalLease(r.Context(), id, claims.UserID, claims.ActorID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewLeaseResponse(l))
}

// Terminate is the dedicated action for ending a lease (R10/R15) —
// distinct from a generic status update so the unit-vacancy wiring and
// audit entry always happen together.
func (h *LeaseHandler) Terminate(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("terminate lease: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.TerminateLeaseRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("terminate lease: %w: %w", domain.ErrInvalidInput, err))
		return
	}
	if req.MoveOutInspectionAttachmentID != "" {
		attachmentID, err := uuid.Parse(req.MoveOutInspectionAttachmentID)
		if err != nil {
			response.WriteError(w, r, fmt.Errorf("terminate lease: move_out_inspection_attachment_id %q: %w", req.MoveOutInspectionAttachmentID, domain.ErrInvalidInput))
			return
		}
		input.MoveOutInspectionAttachmentID = &attachmentID
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.TerminateLease(r.Context(), id, claims.UserID, claims.ActorID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseResponse(l))
}

// CorrectTermination backs the "Correct termination details" action —
// a separate, explicitly-labeled action for fixing a mistake in an
// already-terminated lease's termination record, distinct from
// Terminate and never reachable through a generic field edit.
func (h *LeaseHandler) CorrectTermination(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("correct termination: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.CorrectTerminationRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	input, err := req.ToDomain()
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("correct termination: %w: %w", domain.ErrInvalidInput, err))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	l, err := h.svc.CorrectTermination(r.Context(), id, claims.UserID, claims.ActorID, input, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseResponse(l))
}

// AuditLog backs the Lease Detail page's Change Log section (R18),
// mirroring accountingHandler.TransactionAudit's role for a ledger
// transaction.
func (h *LeaseHandler) AuditLog(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list lease audit: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	entries, err := h.svc.ListAudit(r.Context(), id, claims.UserID, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseAuditListResponse(entries))
}

// ListDocuments, AddDocument, and DeleteDocument back a lease's own
// Documents tab (R14) — mirroring UnitHandler's identical trio for
// unit-level documents.
func (h *LeaseHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("list lease documents: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	docs, err := h.svc.ListDocuments(r.Context(), id, claims.UserID, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.NewLeaseDocumentListResponse(docs))
}

func (h *LeaseHandler) AddDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("add lease document: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	var req dto.AddLeaseDocumentRequest
	if err := decodeAndValidate(r, &req); err != nil {
		response.WriteError(w, r, err)
		return
	}
	attachmentIDStr, category := req.ToDomain()
	attachmentID, err := uuid.Parse(attachmentIDStr)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("add lease document: attachment_id %q: %w", attachmentIDStr, domain.ErrInvalidInput))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	d, err := h.svc.AddDocument(r.Context(), id, claims.UserID, claims.ActorID, attachmentID, category, access)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.NewLeaseDocumentResponse(d))
}

func (h *LeaseHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, r, fmt.Errorf("delete lease document: %w", domain.ErrUnauthorized))
		return
	}

	id, err := parseLeaseIDParam(r)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	documentIDParam := chi.URLParam(r, "documentId")
	documentID, err := uuid.Parse(documentIDParam)
	if err != nil {
		response.WriteError(w, r, fmt.Errorf("delete lease document: id %q: %w", documentIDParam, domain.ErrInvalidInput))
		return
	}

	access, _ := middleware.PropertyAccessFromContext(r.Context())
	if err := h.svc.DeleteDocument(r.Context(), id, documentID, claims.UserID, access); err != nil {
		response.WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseLeaseListOptions mirrors parseUnitListOptions — unrecognized
// filter/sort values are a genuine 400, not silently ignored.
func parseLeaseListOptions(r *http.Request) (domain.LeaseListOptions, error) {
	q := r.URL.Query()
	limit, offset := parsePagination(r)

	opts := domain.LeaseListOptions{
		Limit:  limit,
		Offset: offset,
		Filter: domain.LeaseListFilter{Search: q.Get("search")},
	}

	var verrs domain.ValidationErrors

	if v := q.Get("property_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			verrs = append(verrs, &domain.ValidationError{Field: "property_id", Message: "must be a valid id"})
		} else {
			opts.Filter.PropertyID = &id
		}
	}
	if v := q.Get("unit_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			verrs = append(verrs, &domain.ValidationError{Field: "unit_id", Message: "must be a valid id"})
		} else {
			opts.Filter.UnitID = &id
		}
	}
	if v := q.Get("status"); v != "" {
		s := domain.LeaseDisplayStatus(v)
		if !s.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "status", Message: fmt.Sprintf("unknown status %q", v)})
		} else {
			opts.Filter.DisplayStatus = &s
		}
	}
	if v := q.Get("sort"); v != "" {
		s := domain.LeaseSortKey(v)
		if !s.Valid() {
			verrs = append(verrs, &domain.ValidationError{Field: "sort", Message: fmt.Sprintf("unknown sort key %q", v)})
		} else {
			opts.Sort = s
		}
	}
	if v := q.Get("order"); v != "" {
		switch v {
		case "asc":
			opts.SortDesc = false
		case "desc":
			opts.SortDesc = true
		default:
			verrs = append(verrs, &domain.ValidationError{Field: "order", Message: `must be "asc" or "desc"`})
		}
	}

	if len(verrs) > 0 {
		return domain.LeaseListOptions{}, fmt.Errorf("list leases: %w", verrs)
	}
	return opts, nil
}

func parseLeaseIDParam(r *http.Request) (uuid.UUID, error) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("lease id %q: %w", idParam, domain.ErrInvalidInput)
	}
	return id, nil
}
