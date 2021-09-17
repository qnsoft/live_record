package record

import (
	"io"
	"os"
	"path/filepath"

	. "github.com/qnsoft/live_sdk"
	. "github.com/qnsoft/live_utils"
	"github.com/qnsoft/live_utils/codec"
)

func getDuration(file FileWr) uint32 {
	_, err := file.Seek(-4, io.SeekEnd)
	if err == nil {
		var tagSize uint32
		if tagSize, err = ReadByteToUint32(file, true); err == nil {
			_, err = file.Seek(-int64(tagSize)-4, io.SeekEnd)
			if err == nil {
				_, timestamp, _, err := codec.ReadFLVTag(file)
				if err == nil {
					return timestamp
				}
			}
		}
	}
	return 0
}
func SaveFlv(streamPath string, append bool) error {
	flag := os.O_CREATE
	filePath := filepath.Join(config.Path, streamPath+".flv")
	if append && !Exist(filePath) {
		append = false
	}
	if append {
		flag = flag | os.O_RDWR | os.O_APPEND
	} else {
		flag = flag | os.O_TRUNC | os.O_WRONLY
	}
	var file FileWr
	var err error
	if ExtraConfig.CreateFileFn == nil {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
		file, err = os.OpenFile(filePath, flag, 0755)
		if err != nil {
			return err
		}
	} else {
		file, err = ExtraConfig.CreateFileFn(filePath)
	}
	// return avformat.WriteFLVTag(file, packet)
	p := Subscriber{
		ID:   filePath,
		Type: "FlvRecord",
	}
	var offsetTime uint32
	if append {
		offsetTime = getDuration(file)
		file.Seek(0, io.SeekEnd)
	} else {
		_, err = file.Write(codec.FLVHeader)
	}
	if err == nil {
		recordings.Store(filePath, &p)
		if err := p.Subscribe(streamPath); err == nil {
			vt, at := p.WaitVideoTrack(), p.WaitAudioTrack()
			p.OnAudio = func(ts uint32, audio *AudioPack) {
				if !append && at.CodecID == 10 { //AAC格式需要发送AAC头
					codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_AUDIO, 0, at.ExtraData)
				}
				codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_AUDIO, ts+offsetTime, audio.Payload)
				p.OnAudio = func(ts uint32, audio *AudioPack) {
					codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_AUDIO, ts+offsetTime, audio.Payload)
				}
			}
			p.OnVideo = func(ts uint32, video *VideoPack) {
				if !append {
					codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_VIDEO, 0, vt.ExtraData.Payload)
				}
				codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_VIDEO, ts+offsetTime, video.Payload)
				p.OnVideo = func(ts uint32, video *VideoPack) {
					codec.WriteFLVTag(file, codec.FLV_TAG_TYPE_VIDEO, ts+offsetTime, video.Payload)
				}
			}
			go func() {
				p.Play(at, vt)
				file.Close()
			}()
		}

	} else {
		file.Close()
	}
	return err
}
