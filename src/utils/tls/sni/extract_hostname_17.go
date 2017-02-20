// +build go1.1, !go1.8

package sni

func extractHostname(b []byte) string {
	if b[5] != 0x1 {
		return ""
	}

	// Random
	i := 43

	// Session ID [1 byte length, l bytes data]
	i += 1 + int(b[i])

	// CipherSuite [2 bytes length, l bytes data]
	i += 2 + (int(b[i]) << 8) + int(b[i+1])

	// CompressionMethod [1 byte length l bytes data]
	i += 1 + int(b[i])

	// Padding
	i += 2

	for i < len(b) {
		// 2 bytes extension type
		extensionType := (int(b[i]) << 8) + int(b[i+1])
		i += 2

		// 2 bytes extension length
		extensionLength := (int(b[i]) << 8) + int(b[i+1])
		i += 2

		if extensionType == 0 {

			// Names count
			namesCount := (int(b[i]) << 8) + int(b[i+1])
			i += 2

			for name := 0; name < namesCount; name++ {
				nameType := b[i]
				i += 1

				nameLen := (int(b[i]) << 8) + int(b[i+1])
				i += 2

				if nameType != 0 {
					continue
				}
				// Return the first name encoutered
				return string(b[i : i+nameLen])
			}
		}

		i += extensionLength
	}

	return ""

}
