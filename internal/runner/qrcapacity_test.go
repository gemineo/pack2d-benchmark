package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQRVersionForSize(t *testing.T) {
	tests := []struct {
		name        string
		dataLen     int
		ecLevel     string
		wantVersion int
		wantFits    bool
	}{
		{"tiny L", 10, "L", 1, true},
		{"tiny H", 10, "H", 1, true},
		{"medium M", 500, "M", 14, true},
		{"max L", 4296, "L", 40, true},
		{"over max L", 4297, "L", 40, false},
		{"max H", 1852, "H", 40, true},
		{"over max H", 1853, "H", 40, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, fits := QRVersionForSize(tt.dataLen, tt.ecLevel)
			assert.Equal(t, tt.wantVersion, version)
			assert.Equal(t, tt.wantFits, fits)
		})
	}
}

func TestMaxQRECLevel(t *testing.T) {
	tests := []struct {
		name      string
		dataLen   int
		wantEC    string
		wantVer   int
	}{
		{"very small fits H", 10, "H", 1},
		{"medium fits Q", 2000, "Q", 33},
		{"large fits L only", 4000, "L", 38},
		{"too large", 5000, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec, ver := MaxQRECLevel(tt.dataLen)
			assert.Equal(t, tt.wantEC, ec)
			if tt.wantEC != "" {
				assert.Greater(t, ver, 0)
			} else {
				assert.Equal(t, tt.wantVer, ver)
			}
		})
	}
}

func TestDataMatrixFits(t *testing.T) {
	assert.True(t, DataMatrixFits(100))
	assert.True(t, DataMatrixFits(2335))
	assert.False(t, DataMatrixFits(2336))
}

func TestQRMaxCapacity(t *testing.T) {
	assert.Equal(t, 4296, QRMaxCapacity("L"))
	assert.Equal(t, 3391, QRMaxCapacity("M"))
	assert.Equal(t, 2420, QRMaxCapacity("Q"))
	assert.Equal(t, 1852, QRMaxCapacity("H"))
}
