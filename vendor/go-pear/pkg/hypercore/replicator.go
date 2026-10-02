package hypercore

import (
	"context"
	"crypto/ed25519"
	"errors"
	"sync"
	"time"

	"go-pear/pkg/compactenc"
	"go-pear/pkg/flattree"
	"go-pear/pkg/protomux"
)

var (
	ErrCapabilityMismatch = errors.New("hypercore: peer capability mismatch")
	ErrRequestTimeout     = errors.New("hypercore: request timeout")
)

// ReplicatorPeer coordinates replication over a Protomux channel with a remote peer.
type ReplicatorPeer struct {
	mu             sync.RWMutex
	core           *Hypercore
	mux            *protomux.Multiplexer
	channel        *protomux.Channel
	isInitiator    bool
	handshakeHash  [32]byte
	remoteBitfield *RemoteBitfield

	remoteFork   uint64
	remoteLength uint64
	canUpgrade   bool
	opened       bool

	nextReqID uint64
	inflight  map[uint64]chan *Data
}

// NewReplicatorPeer creates and opens a Hypercore replication channel over Protomux.
func NewReplicatorPeer(core *Hypercore, mux *protomux.Multiplexer, isInitiator bool, handshakeHash [32]byte) (*ReplicatorPeer, error) {
	peer := &ReplicatorPeer{
		core:           core,
		mux:            mux,
		isInitiator:    isInitiator,
		handshakeHash:  handshakeHash,
		remoteBitfield: NewRemoteBitfield(),
		inflight:       make(map[uint64]chan *Data),
		nextReqID:      1,
	}

	channel, err := mux.OpenChannel("hypercore/alpha", peer.onMessage)
	if err != nil {
		return nil, err
	}
	peer.channel = channel

	// Send handshake
	cap := ReplicateCapability(isInitiator, core.Key, handshakeHash)
	hs := Handshake{Seeks: true, Capability: cap}
	sz := PreencodeHandshake(hs)
	st := compactenc.NewAllocatedState(sz)
	_ = EncodeHandshake(st, hs)
	_ = channel.SendMessage(0xff, st.Bytes()) // 0xff handshake type or open

	// Send initial Sync
	_ = peer.SendSync(Sync{
		Fork:         0,
		Length:       core.Length(),
		RemoteLength: 0,
		CanUpgrade:   true,
		Uploading:    true,
		Downloading:  true,
		HasManifest:  core.Manifest != nil,
		AllowPush:    true,
	})

	return peer, nil
}

func (p *ReplicatorPeer) onMessage(typeID uint64, payload []byte) error {
	state := compactenc.NewState(payload)

	switch typeID {
	case 0xff: // Handshake
		hs, err := DecodeHandshake(state)
		if err != nil {
			return err
		}
		expectedCap := ReplicateCapability(!p.isInitiator, p.core.Key, p.handshakeHash)
		if hs.Capability != expectedCap {
			return ErrCapabilityMismatch
		}
		p.mu.Lock()
		p.opened = true
		p.mu.Unlock()
		return nil

	case TypeSync:
		syncMsg, err := DecodeSync(state)
		if err != nil {
			return err
		}
		p.mu.Lock()
		p.remoteFork = syncMsg.Fork
		p.remoteLength = syncMsg.Length
		p.canUpgrade = syncMsg.CanUpgrade
		p.mu.Unlock()
		return nil

	case TypeRequest:
		req, err := DecodeRequest(state)
		if err != nil {
			return err
		}
		if req.Block != nil {
			data, err := p.core.Get(req.Block.Index)
			if err != nil {
				// Send NoData
				return p.SendNoData(NoData{Request: req.ID, Reason: 1})
			}
			proofNodes, _ := p.core.Tree.Proof(req.Block.Index)
			var convertedNodes []Node
			for _, pn := range proofNodes {
				convertedNodes = append(convertedNodes, *pn)
			}

			dMsg := Data{
				Request: req.ID,
				Fork:    0,
				Block: &DataBlock{
					Index: req.Block.Index,
					Value: data,
					Nodes: convertedNodes,
				},
			}

			if p.core.Length() > 0 {
				var rootNodes []Node
				for _, r := range p.core.Tree.Roots() {
					rootNodes = append(rootNodes, *r)
				}
				var sig []byte
				if p.core.KeyPair != nil {
					mHash := ManifestHash(p.core.Manifest)
					tHash := TreeHash(p.core.Tree.Roots())
					signable := TreeSignable(mHash, tHash, p.core.Length(), 0)
					sig = ed25519.Sign(p.core.KeyPair.SecretKey[:], signable)
				}
				dMsg.Upgrade = &DataUpgrade{
					Start:     0,
					Length:    p.core.Length(),
					Nodes:     rootNodes,
					Signature: sig,
				}
			}

			return p.SendData(dMsg)
		}
		return nil

	case TypeData:
		dataMsg, err := DecodeData(state)
		if err != nil {
			return err
		}
		if dataMsg.Upgrade != nil {
			var upRoots []*Node
			for i := range dataMsg.Upgrade.Nodes {
				upRoots = append(upRoots, &dataMsg.Upgrade.Nodes[i])
			}
			tHash := TreeHash(upRoots)
			mHash := ManifestHash(p.core.Manifest)
			signable := TreeSignable(mHash, tHash, dataMsg.Upgrade.Length, 0)

			if len(dataMsg.Upgrade.Signature) > 0 {
				var sig [64]byte
				copy(sig[:], dataMsg.Upgrade.Signature)
				proofs := []MultiSigInput{{Signer: 0, Signature: sig}}
				if err := p.core.Manifest.VerifySignatures(signable, proofs); err != nil {
					if !ed25519.Verify(p.core.Key[:], signable, dataMsg.Upgrade.Signature) {
						return ErrInvalidProof
					}
				}
			}
			p.core.Tree.SetRoots(upRoots, dataMsg.Upgrade.Length)
		}

		if dataMsg.Block != nil {
			// Verify Merkle proof and save
			leafNode := &Node{
				Index: flattree.Index(0, dataMsg.Block.Index),
				Size:  uint64(len(dataMsg.Block.Value)),
				Hash:  LeafHash(dataMsg.Block.Value),
			}
			var converted []*Node
			for i := range dataMsg.Block.Nodes {
				converted = append(converted, &dataMsg.Block.Nodes[i])
			}
			if !p.core.Tree.Verify(leafNode, converted) {
				return ErrInvalidProof
			}
			p.core.Put(dataMsg.Block.Index, dataMsg.Block.Value)
		}

		p.mu.Lock()
		waiter, exists := p.inflight[dataMsg.Request]
		if exists {
			delete(p.inflight, dataMsg.Request)
		}
		p.mu.Unlock()

		if exists && waiter != nil {
			select {
			case waiter <- &dataMsg:
			default:
			}
		}
		return nil

	case TypeBitfield:
		bfMsg, err := DecodeBitfield(state)
		if err != nil {
			return err
		}
		p.remoteBitfield.Insert(bfMsg.Start, bfMsg.Bitfield)
		return nil

	case TypeRange:
		rangeMsg, err := DecodeRange(state)
		if err != nil {
			return err
		}
		p.remoteBitfield.SetRange(rangeMsg.Start, rangeMsg.Length, !rangeMsg.Drop)
		return nil

	default:
		return nil
	}
}

// SendSync sends a wire Sync message.
func (p *ReplicatorPeer) SendSync(s Sync) error {
	sz := PreencodeSync(s)
	st := compactenc.NewAllocatedState(sz)
	if err := EncodeSync(st, s); err != nil {
		return err
	}
	return p.channel.SendMessage(TypeSync, st.Bytes())
}

// SendRequest sends a wire Request message.
func (p *ReplicatorPeer) SendRequest(r Request) error {
	sz := PreencodeRequest(r)
	st := compactenc.NewAllocatedState(sz)
	if err := EncodeRequest(st, r); err != nil {
		return err
	}
	return p.channel.SendMessage(TypeRequest, st.Bytes())
}

// SendData sends a wire Data message.
func (p *ReplicatorPeer) SendData(d Data) error {
	sz := PreencodeData(d)
	st := compactenc.NewAllocatedState(sz)
	if err := EncodeData(st, d); err != nil {
		return err
	}
	return p.channel.SendMessage(TypeData, st.Bytes())
}

// SendNoData sends a wire NoData message.
func (p *ReplicatorPeer) SendNoData(n NoData) error {
	sz := PreencodeNoData(n)
	st := compactenc.NewAllocatedState(sz)
	if err := EncodeNoData(st, n); err != nil {
		return err
	}
	return p.channel.SendMessage(TypeNoData, st.Bytes())
}

// SendRange sends a wire Range message.
func (p *ReplicatorPeer) SendRange(start, length uint64, drop bool) error {
	r := RangeMessage{Start: start, Length: length, Drop: drop}
	sz := PreencodeRange(r)
	st := compactenc.NewAllocatedState(sz)
	if err := EncodeRange(st, r); err != nil {
		return err
	}
	return p.channel.SendMessage(TypeRange, st.Bytes())
}

// SendBitfield sends a wire Bitfield message.
func (p *ReplicatorPeer) SendBitfield(start uint64, words []uint32) error {
	b := BitfieldMessage{Start: start, Bitfield: words}
	sz := PreencodeBitfield(b)
	st := compactenc.NewAllocatedState(sz)
	if err := EncodeBitfield(st, b); err != nil {
		return err
	}
	return p.channel.SendMessage(TypeBitfield, st.Bytes())
}

// RequestBlock requests a block from the peer and waits for completion.
func (p *ReplicatorPeer) RequestBlock(ctx context.Context, index uint64) ([]byte, error) {
	p.mu.Lock()
	reqID := p.nextReqID
	p.nextReqID++
	ch := make(chan *Data, 1)
	p.inflight[reqID] = ch
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.inflight, reqID)
		p.mu.Unlock()
	}()

	req := Request{
		ID:    reqID,
		Fork:  p.remoteFork,
		Block: &RequestBlock{Index: index, Nodes: 0},
	}
	if err := p.SendRequest(req); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		return nil, ErrRequestTimeout
	case data := <-ch:
		if data == nil || data.Block == nil {
			return nil, ErrBlockNotFound
		}
		return data.Block.Value, nil
	}
}

// Close terminates the peer replication channel.
func (p *ReplicatorPeer) Close() error {
	return p.channel.Close()
}
