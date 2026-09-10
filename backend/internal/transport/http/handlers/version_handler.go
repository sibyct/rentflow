package handlers

import (
	"net/http"

	"propertymanagement/internal/transport/http/response"
)

// VersionInfo is build metadata, normally set at compile time via
// -ldflags -X (see cmd/api/main.go) and otherwise defaulted to "dev".
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

type VersionHandler struct {
	info VersionInfo
}

func NewVersionHandler(version, commit, buildDate string) *VersionHandler {
	return &VersionHandler{info: VersionInfo{Version: version, Commit: commit, BuildDate: buildDate}}
}

func (h *VersionHandler) Version(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, h.info)
}
