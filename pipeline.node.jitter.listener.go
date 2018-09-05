package main

import (
	"context"

	plogger "github.com/heytribe/go-plogger"
	"github.com/heytribe/live-webrtcsignaling/srtp"
)

type PipelineNodeJitterListener struct {
	PipelineNode
	// I/O
	In chan *srtp.PacketRTP
	// Publisher
	Out chan *srtp.PacketRTP
	// Listener
	OutRTP  chan *srtp.PacketRTP
	OutRTCP chan *RtpUdpPacket
	// private
	buffer *ListenerBuffer
}

func NewPipelineNodeJitterListener(ctx context.Context, codecOption CodecOptions, pt uint16, ptRtx uint16,
	freq uint32, ssrc uint32, rtxSsrc uint32,
	jst JitterStreamTypeOptions, bitrate Bitrate, rtt *int64) *PipelineNodeJitterListener {
	n := new(PipelineNodeJitterListener)
	// FIXME: check err
	n.buffer, _ = NewJitterBufferListener(ctx, codecOption, pt, ptRtx, freq, ssrc, rtxSsrc, jst, bitrate, rtt, n)
	n.In = make(chan *srtp.PacketRTP, 128)
	n.Out = make(chan *srtp.PacketRTP, 128)
	n.OutRTP = make(chan *srtp.PacketRTP, 128)
	n.OutRTCP = make(chan *RtpUdpPacket, 128)
	return n
}

func (n *PipelineNodeJitterListener) SendRTX(seqs []uint16, ssrc uint32) {
	n.buffer.SendRTX(seqs, ssrc)
}

func (n *PipelineNodeJitterListener) Run(ctx context.Context) {
	n.Running = true
	n.emitStart()
	log := plogger.FromContextSafe(ctx)
	for {
		select {
		case <-ctx.Done():
			n.onStop(ctx)
			return
		case packet := <-n.In:
			n.buffer.PushPacket(packet)
		case packet := <-n.buffer.out:
			select {
			case n.Out <- packet:
			default:
				log.Warnf("Out is full, dropping packet from buffer.out")
			}
		case packet := <-n.buffer.outRTP:
			select {
			case n.OutRTP <- packet:
			default:
				log.Warnf("OutRTP is full, dropping packet from buffer.outRTP")
			}
		case packet := <-n.buffer.outRTCP:
			select {
			case n.OutRTCP <- packet:
			default:
				log.Warnf("OutRTCP is full, dropping packet from buffer.outRTCP")
			}
		case event := <-n.buffer.event:
			select {
			case n.Bus <- event:
			default:
				log.Warnf("Bus is full, dropping packet from buffer.event")
			}
		}
	}
}
