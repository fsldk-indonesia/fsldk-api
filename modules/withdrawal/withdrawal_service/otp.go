package withdrawal_service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// generateOtpCode menghasilkan kode OTP 6-digit acak secara kriptografis.
func generateOtpCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hashOtpCode menghasilkan hash sha256 dari kode OTP — kode tidak pernah
// disimpan plaintext (§12.7 secret management, diterapkan setara password).
func hashOtpCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
