package core

import "github.com/golang/glog"
import "fmt"

// HaveMsg holds a have message data payload
type HaveMsg struct {
	// TODO: start chunk / end chunk
	Start ChunkID
	End   ChunkID
}

// TODO: SendBatchHaves - like SendHave but batches multiple haves into a single datagram

func (p *Peer) SendHave(start ChunkID, end ChunkID, remote PeerID, sid SwarmID) error {
	glog.Infof("%v SendHave Chunks %d-%d, to %v, on %v", p.ID(), start, end, remote, sid)
	swarm, ok1 := p.swarms[sid]
	if !ok1 {
		return fmt.Errorf("SendHave could not find %v", sid)
	}
	ours, ok2 := swarm.chans[remote]
	if !ok2 {
		return fmt.Errorf("SendHave could not find channel for %v on %v", remote, sid)
	}
	c, ok3 := p.chans[ours]
	if !ok3 {
		return fmt.Errorf("SendHave could not find channel %v", ours)
	}
	h := HaveMsg{Start: start, End: end}
	m := Msg{Op: Have, Data: h}
	d := Datagram{ChanID: c.theirs, Msgs: []Msg{m}}
	return p.sendDatagram(d, ours)
}

func (p *Peer) handleHave(cid ChanID, m Msg, remote PeerID) error {
	glog.Infof("%v handleHave from %v", p.ID(), remote)
	c, ok1 := p.chans[cid]
	if !ok1 {
		return fmt.Errorf("handleHave could not find chan %v", cid)
	}
	sid := c.sw
	s, ok2 := p.swarms[sid]
	if !ok2 {
		return fmt.Errorf("handleHave could not find %v", sid)
	}
	h, ok3 := m.Data.(HaveMsg)
	if !ok3 {
		return MsgError{c: cid, m: m, info: "could not convert to Have"}
	}

	return p.requestWantedChunksInRange(h.Start, h.End, remote, sid, s)
}

func (p *Peer) requestWantedChunksInRange(start ChunkID, end ChunkID, remote PeerID, sid SwarmID, s *Swarm) error {
	var startRegion ChunkID
	var endRegion ChunkID
	wantedRegion := false
	for i := start; i <= end; i++ {
		s.AddRemoteHave(i, remote)
		if s.WantChunk(i) {
			endRegion = i
			if !wantedRegion {
				wantedRegion = true
				startRegion = i
			}
		} else {
			if wantedRegion {
				wantedRegion = false
				err := p.SendRequest(startRegion, endRegion, remote, sid)
				if err != nil {
					return err
				}
			}
		}
	}
	// If we left the for loop in a wantedRegion, send the request for it
	if wantedRegion {
		err := p.SendRequest(startRegion, endRegion, remote, sid)
		if err != nil {
			return err
		}
	}
	return nil
}
