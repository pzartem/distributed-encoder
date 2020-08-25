package transcoder

import (
	"fmt"
	"io"
	"log"
	"os/exec"
)

const (
	ffmpeg = "ffmpeg"
)

// Transcoder performs video operations
type Transcoder struct {
	encodeCmdFunc func(EncodeArgs) *exec.Cmd
	cropCmdFunc   func(*CropArgs) *exec.Cmd
}

func New() *Transcoder {
	return &Transcoder{
		encodeCmdFunc: encodeVideo,
		cropCmdFunc:   cropVideo,
	}
}

// TileStream cuts the video and streams the output
func (t *Transcoder) StreamTile(ops *CropArgs) (io.ReadCloser, error) {
	cmd := t.cropCmdFunc(ops)
	return runForget(cmd)
}

// Encode encodes the video stream
func (t *Transcoder) Encode(input io.Reader, ops EncodeArgs) (io.ReadCloser, error) {
	cmd := t.encodeCmdFunc(ops)
	cmd.Stdin = input

	return runForget(cmd)
}

// dirty solution consider running with context
func runForget(cmd *exec.Cmd) (io.ReadCloser, error) {
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("CMD ERROR: %s", cmd.Args)
		}
	}()
	return out, nil
}

// CropArgs for crop stream
type CropArgs struct {
	// input for the src
	Input string
	// X position
	X int
	// Y position
	Y int
	// Height pixel resolution
	Height int
	// Width pixel resolution
	Width int
}

func cropVideo(ops *CropArgs) *exec.Cmd {
	return exec.Command(ffmpeg,
		"-i", ops.Input,
		"-f", "rawvideo",
		"-vf", buildCropFilter(ops),
		"pipe:")
}

// EncodeArgs for encoding encoding
type EncodeArgs struct {
	// Height resolution
	Height int
	// Width pixel resolution
	Width int
}

// encodeVideo command using ffmpeg
func encodeVideo(ops EncodeArgs) *exec.Cmd {
	return exec.Command(ffmpeg,
		"-f", "rawvideo",
		"-pixel_format", "yuv420p",
		"-video_size", fmt.Sprintf("%vx%v", ops.Width, ops.Height),
		"-i", "pipe:",
		"-vf", "hue=s=0",
		"-vcodec", "libx264",
		"-tune", "zerolatency",
		"-preset", "ultrafast",
		"-f", "mpegts",
		"pipe:1")
}

func buildCropFilter(ops *CropArgs) string {
	return fmt.Sprintf("crop=w=%v:h=%v:x=%v:y=%v[a];[a]format=pix_fmts=yuv420p", ops.Width, ops.Height, ops.X, ops.Y)
}
