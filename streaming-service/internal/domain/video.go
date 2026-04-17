package domain

import "time"

// VideoStatus represents the processing state of a video.
type VideoStatus string

const (
	VideoStatusPending    VideoStatus = "pending"    // uploaded, queued for transcoding
	VideoStatusProcessing VideoStatus = "processing" // FFmpeg is running
	VideoStatusReady      VideoStatus = "ready"      // HLS playlist is available
	VideoStatusFailed     VideoStatus = "failed"     // transcoding error
)

// Video represents a video asset in the system.
// Pure Go struct — zero external dependencies.
type Video struct {
	ID           string      `json:"id"`
	UploaderID   string      `json:"uploader_id"`
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	SourcePath   string      `json:"source_path"`  // original file path in MinIO
	HLSPath      string      `json:"hls_path"`     // HLS master playlist path in MinIO
	ThumbnailURL string      `json:"thumbnail_url"`
	DurationSec  int         `json:"duration_sec"`
	Status       VideoStatus `json:"status"`
	ErrorMsg     string      `json:"error_msg,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// TranscodeJob is the message placed on the Redis queue.
// The worker reads this and kicks off FFmpeg.
type TranscodeJob struct {
	VideoID    string `json:"video_id"`
	SourcePath string `json:"source_path"` // MinIO object key
	OutputDir  string `json:"output_dir"`  // MinIO prefix for HLS segments
}
