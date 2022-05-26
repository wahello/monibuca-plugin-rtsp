package rtsp

import (
	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/aac"
	"go.uber.org/zap"
	. "m7s.live/engine/v4"
	"m7s.live/engine/v4/codec"
	"m7s.live/engine/v4/track"
)

type RTSPSubscriber struct {
	Subscriber
	RTSPIO
}

func (s *RTSPSubscriber) OnEvent(event any) {
	switch v := event.(type) {
	case *track.Video:
		switch v.CodecID {
		case codec.CodecID_H264:
			extra := v.DecoderConfiguration.Raw
			if vtrack, err := gortsplib.NewTrackH264(v.DecoderConfiguration.PayloadType, extra[0], extra[1], nil); err == nil {
				s.videoTrackId = len(s.tracks)
				s.tracks = append(s.tracks, vtrack)
			}
		case codec.CodecID_H265:
			if vtrack, err := NewH265Track(v.DecoderConfiguration.PayloadType, v.DecoderConfiguration.Raw); err == nil {
				s.videoTrackId = len(s.tracks)
				s.tracks = append(s.tracks, vtrack)
			}
		}
		s.AddTrack(v)
	case *track.Audio:
		switch v.CodecID {
		case codec.CodecID_AAC:
			var mpegConf aac.MPEG4AudioConfig
			mpegConf.Decode(v.DecoderConfiguration.Raw)
			if atrack, err := gortsplib.NewTrackAAC(v.DecoderConfiguration.PayloadType, int(mpegConf.Type), mpegConf.SampleRate, mpegConf.ChannelCount, mpegConf.AOTSpecificConfig, 13, 3, 3); err == nil {
				s.audioTrackId = len(s.tracks)
				s.tracks = append(s.tracks, atrack)
			} else {
				v.Stream.Error("error creating AAC track", zap.Error(err))
			}
		case codec.CodecID_PCMA:
			s.audioTrackId = len(s.tracks)
			s.tracks = append(s.tracks, NewG711(v.DecoderConfiguration.PayloadType, true))
		case codec.CodecID_PCMU:
			s.audioTrackId = len(s.tracks)
			s.tracks = append(s.tracks, NewG711(v.DecoderConfiguration.PayloadType, false))
		}
		s.AddTrack(v)
	case ISubscriber:
		s.stream = gortsplib.NewServerStream(s.tracks)
	case *AudioFrame:
		for _, pack := range v.RTP {
			s.stream.WritePacketRTP(s.audioTrackId, &pack.Packet, v.PTS == v.DTS)
		}
	case *VideoFrame:
		for _, pack := range v.RTP {
			s.stream.WritePacketRTP(s.videoTrackId, &pack.Packet, v.PTS == v.DTS)
		}
	default:
		s.Subscriber.OnEvent(event)
	}
}
