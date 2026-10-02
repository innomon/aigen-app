package secretstream

import (
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// SymmetricState manages the running hash H and chaining key CK.
type SymmetricState struct {
	h           [32]byte
	ck          [32]byte
	cipherState *CipherState
}

func NewSymmetricState() *SymmetricState {
	h := blake2bHash([]byte(ProtocolName))
	cs, _ := NewCipherState(nil)
	return &SymmetricState{
		h:           h,
		ck:          h,
		cipherState: cs,
	}
}

func (ss *SymmetricState) MixKey(dhOutput []byte) {
	var tempK [32]byte
	ss.ck, tempK = HKDF2(ss.ck, dhOutput)
	cs, _ := NewCipherState(tempK[:])
	ss.cipherState = cs
}

func (ss *SymmetricState) MixHash(data []byte) {
	var buf []byte
	buf = append(buf, ss.h[:]...)
	buf = append(buf, data...)
	ss.h = blake2bHash(buf)
}

func (ss *SymmetricState) EncryptAndHash(plaintext []byte) []byte {
	ciphertext := ss.cipherState.EncryptWithAd(ss.h[:], plaintext)
	ss.MixHash(ciphertext)
	return ciphertext
}

func (ss *SymmetricState) DecryptAndHash(ciphertext []byte) ([]byte, error) {
	plaintext, err := ss.cipherState.DecryptWithAd(ss.h[:], ciphertext)
	if err != nil {
		return nil, err
	}
	ss.MixHash(ciphertext)
	return plaintext, nil
}

func (ss *SymmetricState) Split() (*CipherState, *CipherState) {
	k1, k2 := HKDF2(ss.ck, nil)
	c1, _ := NewCipherState(k1[:])
	c2, _ := NewCipherState(k2[:])
	return c1, c2
}

// HandshakeState coordinates the 3-step Noise XX handshake.
type HandshakeState struct {
	ss           *SymmetricState
	localStatic  *KeyPair
	localEph     *KeyPair
	remoteStatic [KeySize]byte
	remoteEph    [KeySize]byte
	isInitiator  bool
	step         int
	psk          *[KeySize]byte
}

// NewHandshakeState initializes a new Noise XX handshake state machine.
func NewHandshakeState(localStatic *KeyPair, isInitiator bool) (*HandshakeState, error) {
	return NewHandshakeStateWithPSK(localStatic, isInitiator, nil)
}

// NewHandshakeStateWithPSK initializes a new Noise XX handshake state machine with optional PSK.
func NewHandshakeStateWithPSK(localStatic *KeyPair, isInitiator bool, psk *[KeySize]byte) (*HandshakeState, error) {
	eph, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	if localStatic == nil {
		localStatic, err = GenerateKeyPair()
		if err != nil {
			return nil, err
		}
	}
	return &HandshakeState{
		ss:          NewSymmetricState(),
		localStatic: localStatic,
		localEph:    eph,
		isInitiator: isInitiator,
		step:        0,
		psk:         psk,
	}, nil
}

// WriteMessage produces the next outbound handshake payload.
func (hs *HandshakeState) WriteMessage(payload []byte) ([]byte, error) {
	var msg []byte

	if hs.isInitiator {
		switch hs.step {
		case 0:
			// Msg 1 (-> e): Write e, MixHash(e), EncryptAndHash(payload)
			msg = append(msg, hs.localEph.Public[:]...)
			hs.ss.MixHash(hs.localEph.Public[:])
			msg = append(msg, hs.ss.EncryptAndHash(payload)...)
			hs.step = 1
			return msg, nil

		case 2:
			// Msg 3 (-> s, se): EncryptAndHash(s), MixKey(DH(s, re)), EncryptAndHash(payload)
			sEnc := hs.ss.EncryptAndHash(hs.localStatic.Public[:])
			msg = append(msg, sEnc...)

			var se [KeySize]byte
			curve25519.ScalarMult(&se, &hs.localStatic.Private, &hs.remoteEph)
			hs.ss.MixKey(se[:])

			if hs.psk != nil {
				hs.ss.MixKey(hs.psk[:])
			}

			msg = append(msg, hs.ss.EncryptAndHash(payload)...)
			hs.step = 3
			return msg, nil

		default:
			return nil, fmt.Errorf("invalid handshake write step %d for initiator", hs.step)
		}
	} else {
		// Responder
		switch hs.step {
		case 1:
			// Msg 2 (<- e, ee, s, es): Write e, MixHash(e), MixKey(DH(e, re)), EncryptAndHash(s), MixKey(DH(s, re)), EncryptAndHash(payload)
			msg = append(msg, hs.localEph.Public[:]...)
			hs.ss.MixHash(hs.localEph.Public[:])

			var ee [KeySize]byte
			curve25519.ScalarMult(&ee, &hs.localEph.Private, &hs.remoteEph)
			hs.ss.MixKey(ee[:])

			sEnc := hs.ss.EncryptAndHash(hs.localStatic.Public[:])
			msg = append(msg, sEnc...)

			var es [KeySize]byte
			curve25519.ScalarMult(&es, &hs.localStatic.Private, &hs.remoteEph)
			hs.ss.MixKey(es[:])

			msg = append(msg, hs.ss.EncryptAndHash(payload)...)
			hs.step = 2
			return msg, nil

		default:
			return nil, fmt.Errorf("invalid handshake write step %d for responder", hs.step)
		}
	}
}

// ReadMessage processes an inbound handshake payload.
func (hs *HandshakeState) ReadMessage(msg []byte) ([]byte, error) {
	if hs.isInitiator {
		switch hs.step {
		case 1:
			// Reading Msg 2: (<- e, ee, s, es, payload)
			if len(msg) < KeySize+KeySize+TagSize {
				return nil, ErrHandshakeFailed
			}
			copy(hs.remoteEph[:], msg[:KeySize])
			hs.ss.MixHash(hs.remoteEph[:])

			var ee [KeySize]byte
			curve25519.ScalarMult(&ee, &hs.localEph.Private, &hs.remoteEph)
			hs.ss.MixKey(ee[:])

			sCipher := msg[KeySize : KeySize+KeySize+TagSize]
			sPlain, err := hs.ss.DecryptAndHash(sCipher)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt remote static key: %w", err)
			}
			copy(hs.remoteStatic[:], sPlain)

			var es [KeySize]byte
			curve25519.ScalarMult(&es, &hs.localEph.Private, &hs.remoteStatic)
			hs.ss.MixKey(es[:])

			payloadCipher := msg[KeySize+KeySize+TagSize:]
			payload, err := hs.ss.DecryptAndHash(payloadCipher)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt handshake payload: %w", err)
			}
			hs.step = 2
			return payload, nil

		default:
			return nil, fmt.Errorf("invalid handshake read step %d for initiator", hs.step)
		}
	} else {
		// Responder
		switch hs.step {
		case 0:
			// Reading Msg 1: (-> e, payload)
			if len(msg) < KeySize {
				return nil, ErrHandshakeFailed
			}
			copy(hs.remoteEph[:], msg[:KeySize])
			hs.ss.MixHash(hs.remoteEph[:])

			payloadCipher := msg[KeySize:]
			payload, err := hs.ss.DecryptAndHash(payloadCipher)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt handshake payload: %w", err)
			}
			hs.step = 1
			return payload, nil

		case 2:
			// Reading Msg 3: (-> s, se, payload)
			if len(msg) < KeySize+TagSize {
				return nil, ErrHandshakeFailed
			}
			sCipher := msg[:KeySize+TagSize]
			sPlain, err := hs.ss.DecryptAndHash(sCipher)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt remote static key: %w", err)
			}
			copy(hs.remoteStatic[:], sPlain)

			var se [KeySize]byte
			curve25519.ScalarMult(&se, &hs.localEph.Private, &hs.remoteStatic)
			hs.ss.MixKey(se[:])

			if hs.psk != nil {
				hs.ss.MixKey(hs.psk[:])
			}

			payloadCipher := msg[KeySize+TagSize:]
			payload, err := hs.ss.DecryptAndHash(payloadCipher)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt handshake payload: %w", err)
			}
			hs.step = 3
			return payload, nil

		default:
			return nil, fmt.Errorf("invalid handshake read step %d for responder", hs.step)
		}
	}
}

// Finalize splits cipher state into transmission (tx) and reception (rx) channels.
func (hs *HandshakeState) Finalize() (tx, rx *CipherState) {
	c1, c2 := hs.ss.Split()
	if hs.isInitiator {
		return c1, c2
	}
	return c2, c1
}
