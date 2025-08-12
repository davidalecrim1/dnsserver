package dnsserver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaderMarshalBinary(t *testing.T) {
	header := NewHeader(1234, 0x0800, 1, 0, 0, 0)
	buf, err := header.MarshalBinary()
	require.NoError(t, err)

	want := NewHeader(1234, 0x0800, 1, 0, 0, 0)
	err = want.UnmarshalBinary(buf)
	require.NoError(t, err)

	require.Equal(t, want, header)
}

func TestMessageMarshalBinary(t *testing.T) {
	msg := Message{
		Header: NewHeader(1234, 0x0800, 1, 0, 0, 0),
		Questions: []Question{
			{Name: "google.com", Type: 1, Class: 1},
		},
	}
	buf, err := msg.MarshalBinary()
	require.NoError(t, err)

	got, err := NewMessageFromBytes(buf)
	require.NoError(t, err)

	require.Equal(t, msg, got)
	require.Equal(t, msg.Header, got.Header)
	require.Equal(t, msg.Questions, got.Questions)
}
