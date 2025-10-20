package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// AUDecoder decodes Sun/NeXT audio (.au) files to PCM format.
// Supports μ-law encoded mono/stereo audio.
type AUDecoder struct {
	data       []byte // Decoded PCM data (16-bit signed)
	sampleRate int64  // Sample rate in Hz
	channels   int    // Number of channels (1=mono, 2=stereo)
	offset     int64  // Current read position
}

// AU file header structure (24 bytes minimum)
type auHeader struct {
	Magic      uint32 // 0x2e736e64 (".snd")
	DataOffset uint32 // Offset to audio data (typically 24)
	DataSize   uint32 // Size of audio data in bytes (0xFFFFFFFF if unknown)
	Encoding   uint32 // Audio encoding format
	SampleRate uint32 // Sample rate in Hz
	Channels   uint32 // Number of interleaved channels
}

const (
	auMagic         = 0x2e736e64 // ".snd" in big-endian
	auEncodingULaw  = 1          // 8-bit μ-law
	auEncodingPCM16 = 3          // 16-bit linear PCM
)

// μ-law decompression table (converts μ-law byte to 16-bit PCM)
var mulawTable = [256]int16{
	-32124, -31100, -30076, -29052, -28028, -27004, -25980, -24956,
	-23932, -22908, -21884, -20860, -19836, -18812, -17788, -16764,
	-15996, -15484, -14972, -14460, -13948, -13436, -12924, -12412,
	-11900, -11388, -10876, -10364, -9852, -9340, -8828, -8316,
	-7932, -7676, -7420, -7164, -6908, -6652, -6396, -6140,
	-5884, -5628, -5372, -5116, -4860, -4604, -4348, -4092,
	-3900, -3772, -3644, -3516, -3388, -3260, -3132, -3004,
	-2876, -2748, -2620, -2492, -2364, -2236, -2108, -1980,
	-1884, -1820, -1756, -1692, -1628, -1564, -1500, -1436,
	-1372, -1308, -1244, -1180, -1116, -1052, -988, -924,
	-876, -844, -812, -780, -748, -716, -684, -652,
	-620, -588, -556, -524, -492, -460, -428, -396,
	-372, -356, -340, -324, -308, -292, -276, -260,
	-244, -228, -212, -196, -180, -164, -148, -132,
	-120, -112, -104, -96, -88, -80, -72, -64,
	-56, -48, -40, -32, -24, -16, -8, 0,
	32124, 31100, 30076, 29052, 28028, 27004, 25980, 24956,
	23932, 22908, 21884, 20860, 19836, 18812, 17788, 16764,
	15996, 15484, 14972, 14460, 13948, 13436, 12924, 12412,
	11900, 11388, 10876, 10364, 9852, 9340, 8828, 8316,
	7932, 7676, 7420, 7164, 6908, 6652, 6396, 6140,
	5884, 5628, 5372, 5116, 4860, 4604, 4348, 4092,
	3900, 3772, 3644, 3516, 3388, 3260, 3132, 3004,
	2876, 2748, 2620, 2492, 2364, 2236, 2108, 1980,
	1884, 1820, 1756, 1692, 1628, 1564, 1500, 1436,
	1372, 1308, 1244, 1180, 1116, 1052, 988, 924,
	876, 844, 812, 780, 748, 716, 684, 652,
	620, 588, 556, 524, 492, 460, 428, 396,
	372, 356, 340, 324, 308, 292, 276, 260,
	244, 228, 212, 196, 180, 164, 148, 132,
	120, 112, 104, 96, 88, 80, 72, 64,
	56, 48, 40, 32, 24, 16, 8, 0,
}

// DecodeAU decodes a Sun/NeXT audio file (.au) from the given reader.
// Returns an AUDecoder that can be used with Ebitengine's audio system.
//
// Parameters:
//   - r: Reader containing AU file data
//
// Returns:
//   - *AUDecoder: Decoded audio stream
//   - error: Error if decoding fails
func DecodeAU(r io.Reader) (*AUDecoder, error) {
	// Read the entire file into memory
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read AU file: %w", err)
	}

	if len(data) < 24 {
		return nil, fmt.Errorf("AU file too short: %d bytes (minimum 24)", len(data))
	}

	// Parse header (big-endian)
	var header auHeader
	buf := bytes.NewReader(data)
	if err := binary.Read(buf, binary.BigEndian, &header); err != nil {
		return nil, fmt.Errorf("failed to read AU header: %w", err)
	}

	// Verify magic number
	if header.Magic != auMagic {
		return nil, fmt.Errorf("invalid AU magic number: 0x%08x (expected 0x%08x)", header.Magic, auMagic)
	}

	// Only support μ-law encoding for now
	if header.Encoding != auEncodingULaw {
		return nil, fmt.Errorf("unsupported AU encoding: %d (only μ-law [1] is supported)", header.Encoding)
	}

	// Validate channels
	if header.Channels < 1 || header.Channels > 2 {
		return nil, fmt.Errorf("unsupported channel count: %d (only 1-2 supported)", header.Channels)
	}

	// Extract audio data
	audioDataOffset := int(header.DataOffset)
	if audioDataOffset < 24 || audioDataOffset >= len(data) {
		return nil, fmt.Errorf("invalid data offset: %d (file size: %d)", audioDataOffset, len(data))
	}

	ulawData := data[audioDataOffset:]

	// Decode μ-law to 16-bit PCM
	pcmData := make([]byte, len(ulawData)*2) // 16-bit = 2 bytes per sample
	for i, ulaw := range ulawData {
		pcm := mulawTable[ulaw]
		// Write as little-endian 16-bit signed integer
		pcmData[i*2] = byte(pcm)
		pcmData[i*2+1] = byte(pcm >> 8)
	}

	return &AUDecoder{
		data:       pcmData,
		sampleRate: int64(header.SampleRate),
		channels:   int(header.Channels),
		offset:     0,
	}, nil
}

// Read reads decoded PCM data into p.
// Implements io.Reader interface.
func (d *AUDecoder) Read(p []byte) (n int, err error) {
	if d.offset >= int64(len(d.data)) {
		return 0, io.EOF
	}

	n = copy(p, d.data[d.offset:])
	d.offset += int64(n)
	return n, nil
}

// Seek sets the offset for the next Read.
// Implements io.Seeker interface.
func (d *AUDecoder) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64

	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = d.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(d.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("negative position: %d", newOffset)
	}

	d.offset = newOffset
	return newOffset, nil
}

// Length returns the total length of the decoded audio data in bytes.
// Required by Ebitengine's audio.Player.
func (d *AUDecoder) Length() int64 {
	return int64(len(d.data))
}

// SampleRate returns the sample rate of the audio in Hz.
func (d *AUDecoder) SampleRate() int64 {
	return d.sampleRate
}

// Channels returns the number of audio channels (1=mono, 2=stereo).
func (d *AUDecoder) Channels() int {
	return d.channels
}
