package runner

import (
	"fmt"
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

func TestQRModules(t *testing.T) {
	tests := []struct {
		version int
		want    int
	}{
		{1, 21},
		{2, 25},
		{10, 57},
		{40, 177},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("V%d", tt.version), func(t *testing.T) {
			assert.Equal(t, tt.want, QRModules(tt.version))
		})
	}
}

func TestQRSizeMM(t *testing.T) {
	// V1: (21+8)*0.33 = 9.57mm
	assert.InDelta(t, 9.57, QRSizeMM(1, 0.33), 0.01)
	// V40: (177+8)*0.33 = 61.05mm
	assert.InDelta(t, 61.05, QRSizeMM(40, 0.33), 0.01)
}

func TestDataMatrixModules(t *testing.T) {
	tests := []struct {
		name    string
		dataLen int
		want    int
	}{
		{"tiny", 4, 10},
		{"small", 45, 22},
		{"medium", 500, 72},
		{"large", 2335, 144},
		{"too large", 2336, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DataMatrixModules(tt.dataLen))
		})
	}
}

func TestDataMatrixSizeMM(t *testing.T) {
	// 4 chars → 10 modules → (10+2)*0.33 = 3.96mm
	assert.InDelta(t, 3.96, DataMatrixSizeMM(4, 0.33), 0.01)
	// Max capacity → 144 modules → (144+2)*0.33 = 48.18mm
	assert.InDelta(t, 48.18, DataMatrixSizeMM(2335, 0.33), 0.01)
	// Too large → 0
	assert.Equal(t, 0.0, DataMatrixSizeMM(2336, 0.33))
}
