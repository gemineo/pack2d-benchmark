package runner

// QR Code alphanumeric capacity table per ISO 18004.
// Index: [version-1][ecLevel], where ecLevel: 0=L, 1=M, 2=Q, 3=H.
// Versions 1-40.
var qrAlphanumericCapacity = [40][4]int{
	{25, 20, 16, 10},         // V1
	{47, 38, 29, 20},         // V2
	{77, 61, 47, 35},         // V3
	{114, 90, 67, 50},        // V4
	{154, 122, 87, 64},       // V5
	{195, 154, 108, 84},      // V6
	{224, 178, 125, 93},      // V7
	{279, 221, 157, 122},     // V8
	{335, 262, 189, 143},     // V9
	{395, 311, 221, 174},     // V10
	{468, 366, 259, 200},     // V11
	{535, 419, 296, 227},     // V12
	{619, 483, 352, 259},     // V13
	{667, 528, 376, 283},     // V14
	{758, 600, 426, 321},     // V15
	{854, 656, 470, 365},     // V16
	{938, 734, 531, 408},     // V17
	{1046, 816, 574, 452},    // V18
	{1153, 909, 644, 493},    // V19
	{1249, 970, 702, 557},    // V20
	{1352, 1035, 742, 587},   // V21
	{1460, 1134, 823, 640},   // V22
	{1588, 1248, 890, 672},   // V23
	{1704, 1326, 963, 744},   // V24
	{1853, 1451, 1041, 779},  // V25
	{1990, 1542, 1094, 864},  // V26
	{2132, 1637, 1172, 910},  // V27
	{2223, 1732, 1263, 958},  // V28
	{2369, 1839, 1322, 1016}, // V29
	{2520, 1994, 1429, 1080}, // V30
	{2677, 2113, 1499, 1150}, // V31
	{2840, 2238, 1618, 1226}, // V32
	{3009, 2369, 1700, 1307}, // V33
	{3183, 2506, 1787, 1394}, // V34
	{3351, 2632, 1867, 1431}, // V35
	{3537, 2780, 1966, 1530}, // V36
	{3729, 2894, 2071, 1591}, // V37
	{3927, 3054, 2181, 1658}, // V38
	{4087, 3220, 2298, 1774}, // V39
	{4296, 3391, 2420, 1852}, // V40
}

// DataMatrix ECC 200 maximum alphanumeric capacity.
const dataMatrixMaxCapacity = 2335

// ECLevelIndex maps an EC level string to the table column index.
func ECLevelIndex(ec string) int {
	switch ec {
	case "L":
		return 0
	case "M":
		return 1
	case "Q":
		return 2
	case "H":
		return 3
	default:
		return 1 // default to M
	}
}

// QRVersionForSize returns the smallest QR version that can hold dataLen
// alphanumeric characters at the given EC level, and whether it fits.
func QRVersionForSize(dataLen int, ecLevel string) (version int, fits bool) {
	ecIdx := ECLevelIndex(ecLevel)
	for v := range qrAlphanumericCapacity {
		if qrAlphanumericCapacity[v][ecIdx] >= dataLen {
			return v + 1, true
		}
	}
	return 40, false
}

// MaxQRECLevel returns the highest EC level achievable for the given data length,
// along with the QR version needed. Returns empty string if it doesn't fit at any level.
func MaxQRECLevel(dataLen int) (ecLevel string, version int) {
	levels := []string{"H", "Q", "M", "L"} // highest first
	for _, ec := range levels {
		v, fits := QRVersionForSize(dataLen, ec)
		if fits {
			return ec, v
		}
	}
	return "", 0
}

// QRMaxCapacity returns the maximum alphanumeric capacity for a given EC level (version 40).
func QRMaxCapacity(ecLevel string) int {
	return qrAlphanumericCapacity[39][ECLevelIndex(ecLevel)]
}

// DataMatrixFits checks if the data fits in a DataMatrix ECC 200 symbol.
func DataMatrixFits(dataLen int) bool {
	return dataLen <= dataMatrixMaxCapacity
}
