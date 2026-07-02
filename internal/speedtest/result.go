// Package speedtest orchestrates a full velox measurement against a server:
// connectivity, latency, download, and upload.
package speedtest

import "time"

// Phase names a stage of a run.
type Phase string

// Phase constants for each stage of a measurement run.
const (
	PhaseConnectivity Phase = "connectivity"
	PhaseLatency      Phase = "latency"
	PhaseDownload     Phase = "download"
	PhaseUpload       Phase = "upload"
)

// PhaseOutcome records whether a phase succeeded and its measured value.
type PhaseOutcome struct {
	OK    bool    `json:"ok"`
	Error string  `json:"error,omitempty"`
	Value float64 `json:"value,omitempty"`
}

// ServerInfo is the trimmed server description emitted in results.
type ServerInfo struct {
	Machine    string `json:"machine"`
	City       string `json:"city,omitempty"`
	Country    string `json:"country,omitempty"`
	IsFallback bool   `json:"isFallback,omitempty"`
}

// MeasurementResult is the outcome of a `velox --check-internet` run. Its JSON
// encoding conforms to contracts/result.schema.json.
type MeasurementResult struct {
	Online       bool                   `json:"online"`
	LatencyMs    float64                `json:"latencyMs,omitempty"`
	JitterMs     float64                `json:"jitterMs,omitempty"`
	DownloadMbps float64                `json:"downloadMbps,omitempty"`
	UploadMbps   float64                `json:"uploadMbps,omitempty"`
	Server       *ServerInfo            `json:"server,omitempty"`
	DistanceKm   *float64               `json:"distanceKm"`
	StartedAt    time.Time              `json:"startedAt"`
	DurationMs   int64                  `json:"durationMs"`
	PhaseStatus  map[Phase]PhaseOutcome `json:"phaseStatus"`
}
