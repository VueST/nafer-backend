package transcoder

import (
	"context"
)

// Transcoder defines the interface for video transcoding.
// This is the "Port" — any implementation (FFmpeg, cloud service) can satisfy it.
type Transcoder interface {
	// Transcode converts a source video file to HLS format.
	// sourcePath: local or accessible file path of the source video.
	// outputDir:  directory to write HLS segments and playlist.
	// Returns the path to the generated master.m3u8 playlist.
	Transcode(ctx context.Context, sourcePath, outputDir string) (playlistPath string, err error)
}
