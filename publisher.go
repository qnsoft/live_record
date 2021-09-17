package record

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	. "github.com/qnsoft/live_sdk"
	"github.com/qnsoft/live_utils/codec"
)

func PublishFlvFile(streamPath string) error {
	flvPath := filepath.Join(config.Path, streamPath+".flv")
	os.MkdirAll(filepath.Dir(flvPath), 0755)
	if file, err := os.Open(flvPath); err == nil {
		stream := Stream{
			Type:       "FlvFile",
			StreamPath: streamPath,
		}
		if stream.Publish() {
			defer stream.Close()
			file.Seek(int64(len(codec.FLVHeader)), io.SeekStart)
			var lastTime uint32
			at := stream.NewAudioTrack(0)
			vt := stream.NewVideoTrack(0)
			for {
				if t, timestamp, payload, err := codec.ReadFLVTag(file); err == nil {
					switch t {
					case codec.FLV_TAG_TYPE_AUDIO:
						at.PushByteStream(timestamp, payload)
					case codec.FLV_TAG_TYPE_VIDEO:
						if timestamp != 0 {
							if lastTime == 0 {
								lastTime = timestamp
							}
						}
						vt.PushByteStream(timestamp, payload)
						time.Sleep(time.Duration(timestamp-lastTime) * time.Millisecond)
						lastTime = timestamp
					}
				} else {
					return err
				}
			}
		} else {
			return errors.New("Bad Name")
		}
	} else {
		return err
	}
}
