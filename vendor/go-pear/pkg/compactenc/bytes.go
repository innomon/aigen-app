package compactenc

// PreencodeBuffer calculates size of length prefix + raw byte slice.
func PreencodeBuffer(b []byte) int {
	return PreencodeUint(uint64(len(b))) + len(b)
}

// EncodeBuffer encodes a length-prefixed raw byte slice.
func EncodeBuffer(s *State, val []byte) error {
	if err := EncodeUint(s, uint64(len(val))); err != nil {
		return err
	}
	if s.Start+len(val) > len(s.Buffer) {
		return ErrBufferTooSmall
	}
	copy(s.Buffer[s.Start:], val)
	s.Start += len(val)
	return nil
}

// DecodeBuffer decodes a length-prefixed raw byte slice.
func DecodeBuffer(s *State) ([]byte, error) {
	length, err := DecodeUint(s)
	if err != nil {
		return nil, err
	}

	intLen := int(length)
	if intLen < 0 || s.Start+intLen > s.End {
		return nil, ErrUnexpectedEOF
	}

	result := make([]byte, intLen)
	copy(result, s.Buffer[s.Start:s.Start+intLen])
	s.Start += intLen
	return result, nil
}

// PreencodeString calculates size of length prefix + UTF-8 string bytes.
func PreencodeString(str string) int {
	return PreencodeUint(uint64(len(str))) + len(str)
}

// EncodeString encodes a UTF-8 string with compact uint length prefix.
func EncodeString(s *State, val string) error {
	return EncodeBuffer(s, []byte(val))
}

// DecodeString decodes a UTF-8 string with compact uint length prefix.
func DecodeString(s *State) (string, error) {
	b, err := DecodeBuffer(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// PreencodeOptionalBuffer calculates size of optional buffer (1 byte if nil, or PreencodeBuffer if present).
func PreencodeOptionalBuffer(b []byte) int {
	if b == nil {
		return 1
	}
	return PreencodeBuffer(b)
}

// EncodeOptionalBuffer encodes an optional byte slice (compact uint 0 if nil/empty, otherwise length prefix + bytes).
func EncodeOptionalBuffer(s *State, b []byte) error {
	if b == nil {
		return EncodeUint(s, 0)
	}
	return EncodeBuffer(s, b)
}

// DecodeOptionalBuffer decodes an optional byte slice. Returns nil if length is 0.
func DecodeOptionalBuffer(s *State) ([]byte, error) {
	if s.Start >= s.End {
		return nil, ErrUnexpectedEOF
	}
	length, err := DecodeUint(s)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	intLen := int(length)
	if intLen < 0 || s.Start+intLen > s.End {
		return nil, ErrUnexpectedEOF
	}
	res := make([]byte, intLen)
	copy(res, s.Buffer[s.Start:s.Start+intLen])
	s.Start += intLen
	return res, nil
}
