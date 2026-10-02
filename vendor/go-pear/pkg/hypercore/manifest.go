package hypercore

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"golang.org/x/crypto/blake2b"

	"go-pear/pkg/compactenc"
)

const (
	ManifestFlagPatch    = 0b00000001
	ManifestFlagPrologue = 0b00000010
	ManifestFlagLinked   = 0b00000100
	ManifestFlagUserData = 0b00001000
)

var (
	ErrUnknownHashType      = errors.New("hypercore: unknown hash type")
	ErrUnknownSignatureType = errors.New("hypercore: unknown signature type")
	ErrUnknownManifestType  = errors.New("hypercore: unknown manifest type")
	ErrQuorumNotMet         = errors.New("hypercore: signer quorum not met")
	ErrInvalidSignerIndex   = errors.New("hypercore: invalid signer index")
)

// Signer represents an authorized key allowed to sign tree roots.
type Signer struct {
	Signature string   // "ed25519"
	Namespace [32]byte // 32-byte namespace
	PublicKey [32]byte // 32-byte Ed25519 public key
}

// Prologue points to a parent core state.
type Prologue struct {
	Hash   [32]byte
	Length uint64
}

// Manifest defines the Hypercore v10 schema, signing rules, and authorization quorums.
type Manifest struct {
	Version    uint64
	Hash       string // "blake2b"
	AllowPatch bool
	Quorum     uint64
	Signers    []Signer
	Prologue   *Prologue
	Linked     [][32]byte
	UserData   []byte
}

// MultiSigInput represents an individual signer proof.
type MultiSigInput struct {
	Signer    uint64
	Signature [64]byte
	Patch     uint64
}

// PreencodeSigner computes byte size for a Signer.
func PreencodeSigner(s Signer) int {
	return 1 + 32 + 32 // signature type (1) + namespace (32) + publicKey (32)
}

// EncodeSigner encodes a Signer into state.
func EncodeSigner(state *compactenc.State, s Signer) error {
	if s.Signature != "ed25519" && s.Signature != "" {
		return ErrUnknownSignatureType
	}
	if err := compactenc.EncodeUint(state, 0); err != nil { // 0 = ed25519
		return err
	}
	if err := compactenc.EncodeFixed32(state, s.Namespace); err != nil {
		return err
	}
	return compactenc.EncodeFixed32(state, s.PublicKey)
}

// DecodeSigner decodes a Signer from state.
func DecodeSigner(state *compactenc.State) (Signer, error) {
	var s Signer
	sigType, err := compactenc.DecodeUint(state)
	if err != nil {
		return s, err
	}
	if sigType != 0 {
		return s, fmt.Errorf("%w: %d", ErrUnknownSignatureType, sigType)
	}
	s.Signature = "ed25519"
	s.Namespace, err = compactenc.DecodeFixed32(state)
	if err != nil {
		return s, err
	}
	s.PublicKey, err = compactenc.DecodeFixed32(state)
	return s, err
}

// PreencodeManifest calculates the buffer size needed for a Manifest.
func PreencodeManifest(m *Manifest) int {
	if m == nil {
		return 1
	}
	size := 1 // version byte

	if m.Version == 0 {
		size += 1 // hash type (0 for blake2b)
		size += 1 // type (0, 1, or 2)
		if m.Prologue != nil && len(m.Signers) == 0 {
			size += 32
			return size
		}
		if m.Quorum == 1 && len(m.Signers) == 1 && !m.AllowPatch {
			size += PreencodeSigner(m.Signers[0])
			return size
		}
		size += 1                                                // flags
		size += compactenc.PreencodeUint(m.Quorum)               // quorum
		size += compactenc.PreencodeUint(uint64(len(m.Signers))) // signers array len
		size += len(m.Signers) * (1 + 32 + 32)                   // signers
		return size
	}

	// Version 1+
	size += 1                                                // flags
	size += 1                                                // hash type
	size += compactenc.PreencodeUint(m.Quorum)               // quorum
	size += compactenc.PreencodeUint(uint64(len(m.Signers))) // signers array len
	size += len(m.Signers) * (1 + 32 + 32)                   // signers

	if m.Prologue != nil {
		size += 32 + compactenc.PreencodeUint(m.Prologue.Length)
	}
	if len(m.Linked) > 0 {
		size += compactenc.PreencodeFixed32Array(m.Linked)
	}
	if len(m.UserData) > 0 {
		size += compactenc.PreencodeBuffer(m.UserData)
	}
	return size
}

// EncodeManifest encodes a Manifest struct into state according to Hypercore v10.
func EncodeManifest(state *compactenc.State, m *Manifest) error {
	if m == nil {
		return compactenc.EncodeUint(state, 0)
	}

	if err := compactenc.EncodeUint(state, m.Version); err != nil {
		return err
	}

	if m.Version == 0 {
		// Hash: 0 = blake2b
		if err := compactenc.EncodeUint(state, 0); err != nil {
			return err
		}

		if m.Prologue != nil && len(m.Signers) == 0 {
			if err := compactenc.EncodeUint(state, 0); err != nil {
				return err
			}
			return compactenc.EncodeFixed32(state, m.Prologue.Hash)
		}

		if m.Quorum == 1 && len(m.Signers) == 1 && !m.AllowPatch {
			if err := compactenc.EncodeUint(state, 1); err != nil {
				return err
			}
			return EncodeSigner(state, m.Signers[0])
		}

		if err := compactenc.EncodeUint(state, 2); err != nil {
			return err
		}
		var flags uint64
		if m.AllowPatch {
			flags = 1
		}
		if err := compactenc.EncodeUint(state, flags); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, m.Quorum); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, uint64(len(m.Signers))); err != nil {
			return err
		}
		for _, s := range m.Signers {
			if err := EncodeSigner(state, s); err != nil {
				return err
			}
		}
		return nil
	}

	// Version 1+
	var flags uint64
	if m.AllowPatch {
		flags |= ManifestFlagPatch
	}
	if m.Prologue != nil {
		flags |= ManifestFlagPrologue
	}
	if len(m.Linked) > 0 {
		flags |= ManifestFlagLinked
	}
	if len(m.UserData) > 0 {
		flags |= ManifestFlagUserData
	}

	if err := compactenc.EncodeUint(state, flags); err != nil {
		return err
	}
	// Hash: 0 = blake2b
	if err := compactenc.EncodeUint(state, 0); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, m.Quorum); err != nil {
		return err
	}
	if err := compactenc.EncodeUint(state, uint64(len(m.Signers))); err != nil {
		return err
	}
	for _, s := range m.Signers {
		if err := EncodeSigner(state, s); err != nil {
			return err
		}
	}

	if m.Prologue != nil {
		if err := compactenc.EncodeFixed32(state, m.Prologue.Hash); err != nil {
			return err
		}
		if err := compactenc.EncodeUint(state, m.Prologue.Length); err != nil {
			return err
		}
	}
	if len(m.Linked) > 0 {
		if err := compactenc.EncodeFixed32Array(state, m.Linked); err != nil {
			return err
		}
	}
	if len(m.UserData) > 0 {
		if err := compactenc.EncodeBuffer(state, m.UserData); err != nil {
			return err
		}
	}
	return nil
}

// DecodeManifest decodes a Manifest struct from state.
func DecodeManifest(state *compactenc.State) (*Manifest, error) {
	version, err := compactenc.DecodeUint(state)
	if err != nil {
		return nil, err
	}

	if version == 0 {
		hashType, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		if hashType != 0 {
			return nil, ErrUnknownHashType
		}

		mType, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		if mType > 2 {
			return nil, fmt.Errorf("%w: %d", ErrUnknownManifestType, mType)
		}

		if mType == 0 {
			pHash, err := compactenc.DecodeFixed32(state)
			if err != nil {
				return nil, err
			}
			return &Manifest{
				Version:  0,
				Hash:     "blake2b",
				Quorum:   0,
				Signers:  []Signer{},
				Prologue: &Prologue{Hash: pHash, Length: 0},
			}, nil
		}

		if mType == 1 {
			s, err := DecodeSigner(state)
			if err != nil {
				return nil, err
			}
			return &Manifest{
				Version: 0,
				Hash:    "blake2b",
				Quorum:  1,
				Signers: []Signer{s},
			}, nil
		}

		flags, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		quorum, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		numSigners, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		signers := make([]Signer, numSigners)
		for i := uint64(0); i < numSigners; i++ {
			signers[i], err = DecodeSigner(state)
			if err != nil {
				return nil, err
			}
		}

		return &Manifest{
			Version:    0,
			Hash:       "blake2b",
			AllowPatch: (flags & 1) != 0,
			Quorum:     quorum,
			Signers:    signers,
		}, nil
	}

	// Version 1+
	if version > 2 {
		return nil, fmt.Errorf("hypercore: unsupported manifest version %d", version)
	}

	flags, err := compactenc.DecodeUint(state)
	if err != nil {
		return nil, err
	}
	hashType, err := compactenc.DecodeUint(state)
	if err != nil {
		return nil, err
	}
	if hashType != 0 {
		return nil, ErrUnknownHashType
	}
	quorum, err := compactenc.DecodeUint(state)
	if err != nil {
		return nil, err
	}
	numSigners, err := compactenc.DecodeUint(state)
	if err != nil {
		return nil, err
	}
	signers := make([]Signer, numSigners)
	for i := uint64(0); i < numSigners; i++ {
		signers[i], err = DecodeSigner(state)
		if err != nil {
			return nil, err
		}
	}

	var prol *Prologue
	if (flags & ManifestFlagPrologue) != 0 {
		pHash, err := compactenc.DecodeFixed32(state)
		if err != nil {
			return nil, err
		}
		pLen, err := compactenc.DecodeUint(state)
		if err != nil {
			return nil, err
		}
		prol = &Prologue{Hash: pHash, Length: pLen}
	}

	var linked [][32]byte
	if (flags & ManifestFlagLinked) != 0 {
		linked, err = compactenc.DecodeFixed32Array(state)
		if err != nil {
			return nil, err
		}
	}

	var userData []byte
	if (flags & ManifestFlagUserData) != 0 {
		userData, err = compactenc.DecodeBuffer(state)
		if err != nil {
			return nil, err
		}
	}

	return &Manifest{
		Version:    version,
		Hash:       "blake2b",
		AllowPatch: (flags & ManifestFlagPatch) != 0,
		Quorum:     quorum,
		Signers:    signers,
		Prologue:   prol,
		Linked:     linked,
		UserData:   userData,
	}, nil
}

// ManifestHash calculates the 32-byte BLAKE2b hash of the encoded manifest with ManifestNS domain separator:
// BLAKE2b(ManifestNS || compact_encode(manifest)).
func ManifestHash(m *Manifest) [32]byte {
	sz := PreencodeManifest(m)
	st := compactenc.NewAllocatedState(sz)
	_ = EncodeManifest(st, m)

	h, _ := blake2b.New256(nil)
	h.Write(ManifestNS[:])
	h.Write(st.Bytes())

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// VerifySignatures verifies that the provided multi-signature inputs satisfy the manifest quorum against treeSignable.
func (m *Manifest) VerifySignatures(treeSignable []byte, proofs []MultiSigInput) error {
	if m.Quorum == 0 {
		return nil
	}
	if uint64(len(proofs)) < m.Quorum {
		return ErrQuorumNotMet
	}

	validSigners := make(map[uint64]bool)
	for _, p := range proofs {
		if p.Signer >= uint64(len(m.Signers)) {
			return ErrInvalidSignerIndex
		}
		s := m.Signers[p.Signer]
		if ed25519.Verify(s.PublicKey[:], treeSignable, p.Signature[:]) {
			validSigners[p.Signer] = true
		}
	}

	if uint64(len(validSigners)) < m.Quorum {
		return ErrQuorumNotMet
	}
	return nil
}
