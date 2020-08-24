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

type Transcoder struct {}

func (*Transcoder) CropStream(ops CropArgs) (io.ReadCloser, error) {
	cmd := cropVideo(ops)

	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	runForget(cmd)
	return reader, nil
}

func (*Transcoder) Encode(input io.Reader, ops EncodeArgs)  {
	cmd := encodeVideo(ops)
}

// dirty solution consider runing with context
func runForget(cmd *exec.Cmd) {
	go func() {
		if err := cmd.Run(); err != nil {
			log.Printf("Err runing cmd, %s", cmd.Args)
		}
	}()
}


type CropArgs struct {
	Input  string
	X      int
	Y      int
	Height int
	Width  int
}

func cropVideo(ops CropArgs) *exec.Cmd {
	return exec.Command(ffmpeg,
		"-i", ops.Input,
		"-f", "rawvideo",
		"-vf", buildCropFilter(&ops),
		"pipe:")
}

type EncodeArgs struct {
	Height int
	Width  int
}

// Creates EncodeVideo command using ffmpeg
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
