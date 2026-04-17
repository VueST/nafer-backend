package transcoder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FFmpegTranscoder implements the Transcoder interface using local FFmpeg.
// FFmpeg must be installed and available on PATH (guaranteed by the Dockerfile).
type FFmpegTranscoder struct {
	// segmentDuration controls HLS segment length in seconds.
	segmentDuration int
}

// NewFFmpegTranscoder returns a new FFmpegTranscoder with sensible defaults.
func NewFFmpegTranscoder() *FFmpegTranscoder {
	return &FFmpegTranscoder{segmentDuration: 4}
}

// Transcode runs FFmpeg to convert a source video into an HLS stream.
// It produces:
//   - Multiple renditions (360p, 720p, 1080p) with adaptive bitrate
//   - A master.m3u8 playlist referencing all renditions
//
// The caller is responsible for uploading outputDir contents to MinIO.
func (t *FFmpegTranscoder) Transcode(ctx context.Context, sourcePath, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Define renditions: [height, videoBitrate, audioBitrate, bandwidth (for master)]
	renditions := []struct {
		height    int
		vBitrate  string
		aBitrate  string
		bandwidth string
	}{
		{360, "800k", "96k", "896000"},
		{720, "2500k", "128k", "2628000"},
		{1080, "5000k", "192k", "5192000"},
	}

	masterContent := "#EXTM3U\n#EXT-X-VERSION:3\n"
	variantPaths := []string{}

	for _, r := range renditions {
		segDir := filepath.Join(outputDir, fmt.Sprintf("%dp", r.height))
		if err := os.MkdirAll(segDir, 0755); err != nil {
			return "", fmt.Errorf("creating segment dir for %dp: %w", r.height, err)
		}
		playlistFile := filepath.Join(segDir, "index.m3u8")
		segPattern := filepath.Join(segDir, "seg%03d.ts")

		//nolint:gosec // sourcePath comes from internal job queue, not user input
		args := []string{
			"-i", sourcePath,
			"-vf", fmt.Sprintf("scale=-2:%d", r.height),
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "23",
			"-b:v", r.vBitrate,
			"-c:a", "aac",
			"-b:a", r.aBitrate,
			"-hls_time", fmt.Sprintf("%d", t.segmentDuration),
			"-hls_list_size", "0",
			"-hls_segment_filename", segPattern,
			"-f", "hls",
			playlistFile,
		}

		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("ffmpeg failed for %dp: %w", r.height, err)
		}

		// Append variant stream to master playlist
		masterContent += fmt.Sprintf(
			"#EXT-X-STREAM-INF:BANDWIDTH=%s,RESOLUTION=?x%d\n%dp/index.m3u8\n",
			r.bandwidth, r.height, r.height,
		)
		variantPaths = append(variantPaths, playlistFile)
	}

	// Write master playlist
	masterPath := filepath.Join(outputDir, "master.m3u8")
	if err := os.WriteFile(masterPath, []byte(masterContent), 0644); err != nil {
		return "", fmt.Errorf("writing master playlist: %w", err)
	}

	_ = variantPaths // already referenced in master playlist
	return masterPath, nil
}
