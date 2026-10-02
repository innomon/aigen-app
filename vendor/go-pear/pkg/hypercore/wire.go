package hypercore

import (
	"go-pear/pkg/compactenc"
)

// Message Type IDs for Hypercore Wire Protocol over Protomux
const (
	TypeSync      = 0
	TypeRequest   = 1
	TypeCancel    = 2
	TypeData      = 3
	TypeNoData    = 4
	TypeWant      = 5
	TypeUnwant    = 6
	TypeBitfield  = 7
	TypeRange     = 8
	TypeExtension = 9
)

// Handshake represents the initial peer channel handshake message.
type Handshake struct {
	Seeks      bool
	Capability [32]byte
}

func PreencodeHandshake(h Handshake) int {
	return 1 + 32
}

func EncodeHandshake(state *compactenc.State, h Handshake) error {
	var flags uint64
	if h.Seeks {
		flags = 1
	}
	if err := compactenc.EncodeUint(state, flags); err != nil {
		return err
	}
	return compactenc.EncodeFixed32(state, h.Capability)
}

func DecodeHandshake(state *compactenc.State) (Handshake, error) {
	var h Handshake
	flags, err := compactenc.DecodeUint(state)
	if err != nil {
		return h, err
	}
	h.Seeks = (flags & 1) != 0
	h.Capability, err = compactenc.DecodeFixed32(state)
	return h, err
}

// Sync is wire message Type 0: announces local and remote fork state.
type Sync struct {
	Fork         uint64
	Length       uint64
	RemoteLength uint64
	CanUpgrade   bool
	Uploading    bool
	Downloading  bool
	HasManifest  bool
	AllowPush    bool
}

func PreencodeSync(s Sync) int {
	return 1 + compactenc.PreencodeUint(s.Fork) + compactenc.PreencodeUint(s.Length) + compactenc.PreencodeUint(s.RemoteLength)
}

func EncodeSync(state *compactenc.State, s Sync) error {
	var flags uint64
	if s.CanUpgrade {
		flags |= 1
	}
	if s.Uploading {
		flags |= 2
	}
	if s.Downloading {
		flags |= 4
	}
	if s.HasManifest {
		flags |= 8
	}
	if s.AllowPush {
		flags |= 16
	}

	if err := compactenc.EncodeUint(state, flags); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, s.Fork); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, s.Length); err != nil {
		return err
	}
	return compactenc.EncodeUint(state, s.RemoteLength)
}

func DecodeSync(state *compactenc.State) (Sync, error) {
	var s Sync
	flags, err := compactenc.DecodeUint(state)
	if err != nil {
		return s, err
	}
	s.CanUpgrade = (flags & 1) != 0
	s.Uploading = (flags & 2) != 0
	s.Downloading = (flags & 4) != 0
	s.HasManifest = (flags & 8) != 0
	s.AllowPush = (flags & 16) != 0

	s.Fork, err = compactenc.DecodeUint(state)
	if err != nil {
		return s, err
	}
	s.Length, err = compactenc.DecodeUint(state)
	if err != nil {
		return s, err
	}
	s.RemoteLength, err = compactenc.DecodeUint(state)
	return s, err
}

// RequestBlock specifies a block or hash request.
type RequestBlock struct {
	Index uint64
	Nodes uint64
}

// RequestSeek specifies a byte seek request.
type RequestSeek struct {
	Bytes   uint64
	Padding uint64
}

// RequestUpgrade specifies a tree upgrade range request.
type RequestUpgrade struct {
	Start  uint64
	Length uint64
}

// Request is wire message Type 1: requests blocks, hashes, seeks, upgrades, or manifests.
type Request struct {
	ID       uint64
	Fork     uint64
	Block    *RequestBlock
	Hash     *RequestBlock
	Seek     *RequestSeek
	Upgrade  *RequestUpgrade
	Manifest bool
	Priority uint64
}

func PreencodeRequest(r Request) int {
	size := 1 + compactenc.PreencodeUint(r.ID) + compactenc.PreencodeUint(r.Fork)
	if r.Block != nil {
		size += compactenc.PreencodeUint(r.Block.Index) + compactenc.PreencodeUint(r.Block.Nodes)
	}
	if r.Hash != nil {
		size += compactenc.PreencodeUint(r.Hash.Index) + compactenc.PreencodeUint(r.Hash.Nodes)
	}
	if r.Seek != nil {
		size += compactenc.PreencodeUint(r.Seek.Bytes) + compactenc.PreencodeUint(r.Seek.Padding)
	}
	if r.Upgrade != nil {
		size += compactenc.PreencodeUint(r.Upgrade.Start) + compactenc.PreencodeUint(r.Upgrade.Length)
	}
	if r.Priority > 0 {
		size += compactenc.PreencodeUint(r.Priority)
	}
	return size
}

func EncodeRequest(state *compactenc.State, r Request) error {
	var flags uint64
	if r.Block != nil {
		flags |= 1
	}
	if r.Hash != nil {
		flags |= 2
	}
	if r.Seek != nil {
		flags |= 4
	}
	if r.Upgrade != nil {
		flags |= 8
	}
	if r.Manifest {
		flags |= 16
	}
	if r.Priority > 0 {
		flags |= 32
	}

	if err := compactenc.EncodeUint(state, flags); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, r.ID); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, r.Fork); err != nil {
		return err
	}

	if r.Block != nil {
		if err := compactenc.EncodeUint(state, r.Block.Index); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, r.Block.Nodes); err != nil {
			return err
		}
	}
	if r.Hash != nil {
		if err := compactenc.EncodeUint(state, r.Hash.Index); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, r.Hash.Nodes); err != nil {
			return err
		}
	}
	if r.Seek != nil {
		if err := compactenc.EncodeUint(state, r.Seek.Bytes); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, r.Seek.Padding); err != nil {
			return err
		}
	}
	if r.Upgrade != nil {
		if err := compactenc.EncodeUint(state, r.Upgrade.Start); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, r.Upgrade.Length); err != nil {
			return err
		}
	}
	if r.Priority > 0 {
		if err := compactenc.EncodeUint(state, r.Priority); err != nil {
			return err
		}
	}
	return nil
}

func DecodeRequest(state *compactenc.State) (Request, error) {
	var r Request
	flags, err := compactenc.DecodeUint(state)
	if err != nil {
		return r, err
	}
	r.ID, err = compactenc.DecodeUint(state)
	if err != nil {
		return r, err
	}
	r.Fork, err = compactenc.DecodeUint(state)
	if err != nil {
		return r, err
	}

	if (flags & 1) != 0 {
		idx, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		nodes, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		r.Block = &RequestBlock{Index: idx, Nodes: nodes}
	}
	if (flags & 2) != 0 {
		idx, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		nodes, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		r.Hash = &RequestBlock{Index: idx, Nodes: nodes}
	}
	if (flags & 4) != 0 {
		bytes, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		pad, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		r.Seek = &RequestSeek{Bytes: bytes, Padding: pad}
	}
	if (flags & 8) != 0 {
		st, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		l, err := compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
		r.Upgrade = &RequestUpgrade{Start: st, Length: l}
	}
	r.Manifest = (flags & 16) != 0
	if (flags & 32) != 0 {
		r.Priority, err = compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
	}
	return r, nil
}

// Cancel is wire message Type 2: cancels an inflight request.
type Cancel struct {
	Request uint64
}

func PreencodeCancel(c Cancel) int {
	return compactenc.PreencodeUint(c.Request)
}

func EncodeCancel(state *compactenc.State, c Cancel) error {
	return compactenc.EncodeUint(state, c.Request)
}

func DecodeCancel(state *compactenc.State) (Cancel, error) {
	req, err := compactenc.DecodeUint(state)
	return Cancel{Request: req}, err
}

func preencodeNodeArray(nodes []Node) int {
	sz := compactenc.PreencodeUint(uint64(len(nodes)))
	for _, n := range nodes {
		sz += compactenc.PreencodeUint(n.Index) + compactenc.PreencodeUint(n.Size) + 32
	}
	return sz
}

func encodeNodeArray(state *compactenc.State, nodes []Node) error {
	if err := compactenc.EncodeUint(state, uint64(len(nodes))); err != nil {
		return err
	}
	for _, n := range nodes {
		if err := compactenc.EncodeUint(state, n.Index); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, n.Size); err != nil {
			return err
		}
		if err := compactenc.EncodeFixed32(state, n.Hash); err != nil {
			return err
		}
	}
	return nil
}

func decodeNodeArray(state *compactenc.State) ([]Node, error) {
	count, err := compactenc.DecodeUint(state)
	if err != nil {
		return nil, err
	}
	res := make([]Node, count)
	for i := uint64(0); i < count; i++ {
		idx, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		sz, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		h, err := compactenc.DecodeFixed32(state)
		if err != nil {
			return nil, err
		}
		res[i] = Node{Index: idx, Size: sz, Hash: h}
	}
	return res, nil
}

// DataBlock contains block data and Merkle verification nodes.
type DataBlock struct {
	Index uint64
	Value []byte
	Nodes []Node
}

// DataHash contains hash verification nodes.
type DataHash struct {
	Index uint64
	Nodes []Node
}

// DataSeek contains seek verification nodes.
type DataSeek struct {
	Bytes uint64
	Nodes []Node
}

// DataUpgrade contains tree upgrade proofs and signature.
type DataUpgrade struct {
	Start           uint64
	Length          uint64
	Nodes           []Node
	AdditionalNodes []Node
	Signature       []byte
}

// Data is wire message Type 3: transmits blocks, hashes, seeks, upgrades, or manifests.
type Data struct {
	Request  uint64
	Fork     uint64
	Block    *DataBlock
	Hash     *DataHash
	Seek     *DataSeek
	Upgrade  *DataUpgrade
	Manifest *Manifest
}

func PreencodeData(d Data) int {
	size := 1 + compactenc.PreencodeUint(d.Request) + compactenc.PreencodeUint(d.Fork)
	if d.Block != nil {
		size += compactenc.PreencodeUint(d.Block.Index) + compactenc.PreencodeBuffer(d.Block.Value) + preencodeNodeArray(d.Block.Nodes)
	}
	if d.Hash != nil {
		size += compactenc.PreencodeUint(d.Hash.Index) + preencodeNodeArray(d.Hash.Nodes)
	}
	if d.Seek != nil {
		size += compactenc.PreencodeUint(d.Seek.Bytes) + preencodeNodeArray(d.Seek.Nodes)
	}
	if d.Upgrade != nil {
		size += compactenc.PreencodeUint(d.Upgrade.Start) + compactenc.PreencodeUint(d.Upgrade.Length) + preencodeNodeArray(d.Upgrade.Nodes) + preencodeNodeArray(d.Upgrade.AdditionalNodes) + compactenc.PreencodeOptionalBuffer(d.Upgrade.Signature)
	}
	if d.Manifest != nil {
		size += PreencodeManifest(d.Manifest)
	}
	return size
}

func EncodeData(state *compactenc.State, d Data) error {
	var flags uint64
	if d.Block != nil {
		flags |= 1
	}
	if d.Hash != nil {
		flags |= 2
	}
	if d.Seek != nil {
		flags |= 4
	}
	if d.Upgrade != nil {
		flags |= 8
	}
	if d.Manifest != nil {
		flags |= 16
	}

	if err := compactenc.EncodeUint(state, flags); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, d.Request); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, d.Fork); err != nil {
		return err
	}

	if d.Block != nil {
		if err := compactenc.EncodeUint(state, d.Block.Index); err != nil {
			return err
		}
		if err := compactenc.EncodeBuffer(state, d.Block.Value); err != nil {
			return err
		}
		if err := encodeNodeArray(state, d.Block.Nodes); err != nil {
			return err
		}
	}
	if d.Hash != nil {
		if err := compactenc.EncodeUint(state, d.Hash.Index); err != nil {
			return err
		}
		if err := encodeNodeArray(state, d.Hash.Nodes); err != nil {
			return err
		}
	}
	if d.Seek != nil {
		if err := compactenc.EncodeUint(state, d.Seek.Bytes); err != nil {
			return err
		}
		if err := encodeNodeArray(state, d.Seek.Nodes); err != nil {
			return err
		}
	}
	if d.Upgrade != nil {
		if err := compactenc.EncodeUint(state, d.Upgrade.Start); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, d.Upgrade.Length); err != nil {
			return err
		}
		if err := encodeNodeArray(state, d.Upgrade.Nodes); err != nil {
			return err
		}
		if err := encodeNodeArray(state, d.Upgrade.AdditionalNodes); err != nil {
			return err
		}
		if err := compactenc.EncodeOptionalBuffer(state, d.Upgrade.Signature); err != nil {
			return err
		}
	}
	if d.Manifest != nil {
		if err := EncodeManifest(state, d.Manifest); err != nil {
			return err
		}
	}
	return nil
}

func DecodeData(state *compactenc.State) (Data, error) {
	var d Data
	flags, err := compactenc.DecodeUint(state)
	if err != nil {
		return d, err
	}
	d.Request, err = compactenc.DecodeUint(state)
	if err != nil {
		return d, err
	}
	d.Fork, err = compactenc.DecodeUint(state)
	if err != nil {
		return d, err
	}

	if (flags & 1) != 0 {
		idx, err := compactenc.DecodeUint(state)
		if err != nil {
			return d, err
		}
		val, err := compactenc.DecodeBuffer(state)
		if err != nil {
			return d, err
		}
		nodes, err := decodeNodeArray(state)
		if err != nil {
			return d, err
		}
		d.Block = &DataBlock{Index: idx, Value: val, Nodes: nodes}
	}
	if (flags & 2) != 0 {
		idx, err := compactenc.DecodeUint(state)
		if err != nil {
			return d, err
		}
		nodes, err := decodeNodeArray(state)
		if err != nil {
			return d, err
		}
		d.Hash = &DataHash{Index: idx, Nodes: nodes}
	}
	if (flags & 4) != 0 {
		b, err := compactenc.DecodeUint(state)
		if err != nil {
			return d, err
		}
		nodes, err := decodeNodeArray(state)
		if err != nil {
			return d, err
		}
		d.Seek = &DataSeek{Bytes: b, Nodes: nodes}
	}
	if (flags & 8) != 0 {
		st, err := compactenc.DecodeUint(state)
		if err != nil {
			return d, err
		}
		l, err := compactenc.DecodeUint(state)
		if err != nil {
			return d, err
		}
		nodes, err := decodeNodeArray(state)
		if err != nil {
			return d, err
		}
		addNodes, err := decodeNodeArray(state)
		if err != nil {
			return d, err
		}
		sig, err := compactenc.DecodeOptionalBuffer(state)
		if err != nil {
			return d, err
		}
		d.Upgrade = &DataUpgrade{Start: st, Length: l, Nodes: nodes, AdditionalNodes: addNodes, Signature: sig}
	}
	if (flags & 16) != 0 {
		m, err := DecodeManifest(state)
		if err != nil {
			return d, err
		}
		d.Manifest = m
	}
	return d, nil
}

// NoData is wire message Type 4: negative response to a request.
type NoData struct {
	Request uint64
	Reason  uint64
}

func PreencodeNoData(n NoData) int {
	sz := compactenc.PreencodeUint(n.Request) + 1
	if n.Reason > 0 {
		sz += compactenc.PreencodeUint(n.Reason)
	}
	return sz
}

func EncodeNoData(state *compactenc.State, n NoData) error {
	if err := compactenc.EncodeUint(state, n.Request); err != nil {
		return err
	}
	var flags uint64
	if n.Reason > 0 {
		flags = 1
	}
	if err := compactenc.EncodeUint(state, flags); err != nil {
		return err
	}
	if n.Reason > 0 {
		return compactenc.EncodeUint(state, n.Reason)
	}
	return nil
}

func DecodeNoData(state *compactenc.State) (NoData, error) {
	var n NoData
	var err error
	n.Request, err = compactenc.DecodeUint(state)
	if err != nil {
		return n, err
	}
	if state.Start < state.End {
		flags, err := compactenc.DecodeUint(state)
		if err != nil {
			return n, err
		}
		if (flags & 1) != 0 {
			n.Reason, err = compactenc.DecodeUint(state)
			if err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

// Want is wire message Type 5: requests a block range from a peer.
type Want struct {
	Start  uint64
	Length uint64
	Any    bool
}

func PreencodeWant(w Want) int {
	sz := compactenc.PreencodeUint(w.Start) + compactenc.PreencodeUint(w.Length)
	if w.Any {
		sz += compactenc.PreencodeUint(1)
	}
	return sz
}

func EncodeWant(state *compactenc.State, w Want) error {
	if err := compactenc.EncodeUint(state, w.Start); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, w.Length); err != nil {
		return err
	}
	if w.Any {
		return compactenc.EncodeUint(state, 1)
	}
	return nil
}

func DecodeWant(state *compactenc.State) (Want, error) {
	var w Want
	var err error
	w.Start, err = compactenc.DecodeUint(state)
	if err != nil {
		return w, err
	}
	w.Length, err = compactenc.DecodeUint(state)
	if err != nil {
		return w, err
	}
	if state.Start < state.End {
		flags, err := compactenc.DecodeUint(state)
		if err != nil {
			return w, err
		}
		w.Any = (flags & 1) != 0
	}
	return w, nil
}

// Unwant is wire message Type 6: cancels a wanted range.
type Unwant = Want

func PreencodeUnwant(u Unwant) int                         { return PreencodeWant(u) }
func EncodeUnwant(state *compactenc.State, u Unwant) error { return EncodeWant(state, u) }
func DecodeUnwant(state *compactenc.State) (Unwant, error) { return DecodeWant(state) }

// BitfieldMessage is wire message Type 7: transmits available block bitfield words.
type BitfieldMessage struct {
	Start    uint64
	Bitfield []uint32
}

func PreencodeBitfield(b BitfieldMessage) int {
	return compactenc.PreencodeUint(b.Start) + compactenc.PreencodeUint32Array(b.Bitfield)
}

func EncodeBitfield(state *compactenc.State, b BitfieldMessage) error {
	if err := compactenc.EncodeUint(state, b.Start); err != nil {
		return err
	}
	return compactenc.EncodeUint32Array(state, b.Bitfield)
}

func DecodeBitfield(state *compactenc.State) (BitfieldMessage, error) {
	var b BitfieldMessage
	var err error
	b.Start, err = compactenc.DecodeUint(state)
	if err != nil {
		return b, err
	}
	b.Bitfield, err = compactenc.DecodeUint32Array(state)
	return b, err
}

// RangeMessage is wire message Type 8: announces contiguous available/dropped block range.
type RangeMessage struct {
	Drop   bool
	Start  uint64
	Length uint64
}

func PreencodeRange(r RangeMessage) int {
	sz := 1 + compactenc.PreencodeUint(r.Start)
	if r.Length != 1 {
		sz += compactenc.PreencodeUint(r.Length)
	}
	return sz
}

func EncodeRange(state *compactenc.State, r RangeMessage) error {
	var flags uint64
	if r.Drop {
		flags |= 1
	}
	if r.Length == 1 {
		flags |= 2
	}
	if err := compactenc.EncodeUint(state, flags); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, r.Start); err != nil {
		return err
	}
	if r.Length != 1 {
		return compactenc.EncodeUint(state, r.Length)
	}
	return nil
}

func DecodeRange(state *compactenc.State) (RangeMessage, error) {
	var r RangeMessage
	flags, err := compactenc.DecodeUint(state)
	if err != nil {
		return r, err
	}
	r.Drop = (flags & 1) != 0
	r.Start, err = compactenc.DecodeUint(state)
	if err != nil {
		return r, err
	}
	if (flags & 2) != 0 {
		r.Length = 1
	} else {
		r.Length, err = compactenc.DecodeUint(state)
		if err != nil {
			return r, err
		}
	}
	return r, nil
}

// ExtensionMessage is wire message Type 9: carries custom extension payloads.
type ExtensionMessage struct {
	Name    string
	Message []byte
}

func PreencodeExtension(e ExtensionMessage) int {
	return compactenc.PreencodeString(e.Name) + len(e.Message)
}

func EncodeExtension(state *compactenc.State, e ExtensionMessage) error {
	if err := compactenc.EncodeString(state, e.Name); err != nil {
		return err
	}
	if state.Start+len(e.Message) > len(state.Buffer) {
		return compactenc.ErrBufferTooSmall
	}
	copy(state.Buffer[state.Start:], e.Message)
	state.Start += len(e.Message)
	return nil
}

func DecodeExtension(state *compactenc.State) (ExtensionMessage, error) {
	var e ExtensionMessage
	name, err := compactenc.DecodeString(state)
	if err != nil {
		return e, err
	}
	e.Name = name
	if state.Start < state.End {
		msg := make([]byte, state.End-state.Start)
		copy(msg, state.Buffer[state.Start:state.End])
		state.Start = state.End
		e.Message = msg
	}
	return e, nil
}
