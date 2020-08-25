package transcoder

import (
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	encodeArgs = EncodeArgs{
		Height: 30,
		Width: 50,
	}

	cropArgs = CropArgs{
		Input:  "file",
		X:      200,
		Y:      100,
		Width:  50,
		Height: 30,
	}

	expectedOut = "I'm an 8k video"
)

func TestTranscoder_Encode(t *testing.T) {
	coder := Transcoder{
		encodeCmdFunc: func(args EncodeArgs) *exec.Cmd {
			require.Equal(t, encodeArgs, args)
			return exec.Command("cat")
		},
	}

	out, err := coder.Encode(strings.NewReader(expectedOut), encodeArgs)
	require.NoError(t, err)
	defer out.Close()

	result, err := ioutil.ReadAll(out)
	require.NoError(t, err)
	require.Equal(t, expectedOut, string(result))
}

func TestTranscoder_StreamTile(t *testing.T) {
	coder := Transcoder{
		cropCmdFunc: func(args *CropArgs) *exec.Cmd {
			require.Equal(t, args, &cropArgs)
			return exec.Command("echo", expectedOut)
		},
	}

	out, err := coder.StreamTile(&cropArgs)
	require.NoError(t, err)
	defer out.Close()

	result, err := ioutil.ReadAll(out)
	require.NoError(t, err)
	require.Equal(t, expectedOut + "\n", string(result))
}

func Test_cropVideo(t *testing.T) {
	cmd := cropVideo(&cropArgs)

	expected := []string{
		"ffmpeg",
		"-i", cropArgs.Input,
		"-f", "rawvideo",
		"-vf", "crop=w=50:h=30:x=200:y=100[a];[a]format=pix_fmts=yuv420p",
		"pipe:",
	}
	require.Equal(t, expected, cmd.Args)
}

func Test_encodeVideo(t *testing.T) {
	cmd := encodeVideo(encodeArgs)

	expected := []string{
		"ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "yuv420p",
		"-video_size", "50x30",
		"-i", "pipe:",
		"-vf", "hue=s=0",
		"-vcodec", "libx264",
		"-tune", "zerolatency",
		"-preset", "ultrafast",
		"-f", "mpegts",
		"pipe:1",
	}
	require.Equal(t, expected, cmd.Args)
}
