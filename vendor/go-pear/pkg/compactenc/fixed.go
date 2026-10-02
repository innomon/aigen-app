package compactenc

// EncodeFixed32 writes exactly 32 bytes without a length prefix.
func EncodeFixed32(s *State, val [32]byte) error {
	if s.Start+32 > len(s.Buffer) {
		return ErrBufferTooSmall
	}
	copy(s.Buffer[s.Start:], val[:])
	s.Start += 32
	return nil
}

// DecodeFixed32 reads exactly 32 bytes from state.
func DecodeFixed32(s *State) ([32]byte, error) {
	var result [32]byte
	if s.Start+32 > s.End {
		return result, ErrUnexpectedEOF
	}
	copy(result[:], s.Buffer[s.Start:s.Start+32])
	s.Start += 32
	return result, nil
}

// EncodeFixed64 writes exactly 64 bytes without a length prefix.
func EncodeFixed64(s *State, val [64]byte) error {
	if s.Start+64 > len(s.Buffer) {
		return ErrBufferTooSmall
	}
	copy(s.Buffer[s.Start:], val[:])
	s.Start += 64
	return nil
}

// DecodeFixed64 reads exactly 64 bytes from state.
func DecodeFixed64(s *State) ([64]byte, error) {
	var result [64]byte
	if s.Start+64 > s.End {
		return result, ErrUnexpectedEOF
	}
	copy(result[:], s.Buffer[s.Start:s.Start+64])
	s.Start += 64
	return result, nil
}

// PreencodeFixed32Array calculates byte size required for an array of 32-byte hashes.
func PreencodeFixed32Array(arr [][32]byte) int {
	return PreencodeUint(uint64(len(arr))) + len(arr)*32
}

// EncodeFixed32Array encodes a slice of [32]byte hashes with compact uint length.
func EncodeFixed32Array(s *State, arr [][32]byte) error {
	if err := EncodeUint(s, uint64(len(arr))); err != nil {
		return err
	}
	if s.Start+len(arr)*32 > len(s.Buffer) {
		return ErrBufferTooSmall
	}
	for _, item := range arr {
		copy(s.Buffer[s.Start:], item[:])
		s.Start += 32
	}
	return nil
}

// DecodeFixed32Array decodes a slice of [32]byte hashes with compact uint length.
func DecodeFixed32Array(s *State) ([][32]byte, error) {
	count, err := DecodeUint(s)
	if err != nil {
		return nil, err
	}
	intCount := int(count)
	if intCount < 0 || s.Start+intCount*32 > s.End {
		return nil, ErrUnexpectedEOF
	}
	res := make([][32]byte, intCount)
	for i := 0; i < intCount; i++ {
		copy(res[i][:], s.Buffer[s.Start:s.Start+32])
		s.Start += 32
	}
	return res, nil
}
