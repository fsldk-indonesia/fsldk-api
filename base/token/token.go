// Package token menangani pembuatan dan validasi JWT (access & refresh token)
// sesuai TechSpec bagian 17.2.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims adalah payload access token.
type Claims struct {
	UserID               int64  `json:"userID"`
	Email                string `json:"email"`
	RoleID               int64  `json:"roleID"`
	RoleName             string `json:"roleName"`
	EmailVerified        bool   `json:"emailVerified"`
	OrganizationID       *int64 `json:"organizationID,omitempty"`
	OrganizationTypeCode string `json:"organizationTypeCode,omitempty"`
	WildcardTierAccess   string `json:"wildcardTierAccess,omitempty"`
	jwt.RegisteredClaims
}

// RefreshClaims adalah payload refresh token (minimal).
type RefreshClaims struct {
	UserID int64 `json:"userID"`
	jwt.RegisteredClaims
}

// Manager mengelola penerbitan & verifikasi token dengan secret & masa berlaku
// yang dikonfigurasi.
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpire  time.Duration
	refreshExpire time.Duration
}

// NewManager membuat Manager token.
func NewManager(accessSecret, refreshSecret string, accessExpireMin, refreshExpireMin int) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpire:  time.Duration(accessExpireMin) * time.Minute,
		refreshExpire: time.Duration(refreshExpireMin) * time.Minute,
	}
}

// AccessParams menampung data identitas untuk penerbitan access token.
type AccessParams struct {
	UserID               int64
	RoleID               int64
	Email                string
	RoleName             string
	EmailVerified        bool
	OrganizationID       *int64
	OrganizationTypeCode string
	WildcardTierAccess   string
}

// GenerateAccess membuat access token berisi identitas, status verifikasi,
// dan scope organisasi.
func (m *Manager) GenerateAccess(p AccessParams) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:               p.UserID,
		Email:                p.Email,
		RoleID:               p.RoleID,
		RoleName:             p.RoleName,
		EmailVerified:        p.EmailVerified,
		OrganizationID:       p.OrganizationID,
		OrganizationTypeCode: p.OrganizationTypeCode,
		WildcardTierAccess:   p.WildcardTierAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.Email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessExpire)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.accessSecret)
}

// GenerateRefresh membuat refresh token.
func (m *Manager) GenerateRefresh(userID int64) (string, error) {
	now := time.Now()
	claims := RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExpire)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.refreshSecret)
}

// ParseAccess memvalidasi & mengekstrak klaim access token.
func (m *Manager) ParseAccess(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing tidak valid")
		}
		return m.accessSecret, nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}

// ParseRefresh memvalidasi & mengekstrak klaim refresh token.
func (m *Manager) ParseRefresh(tokenStr string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing tidak valid")
		}
		return m.refreshSecret, nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("refresh token tidak valid")
	}
	return claims, nil
}

// AccessExpireSeconds mengembalikan masa berlaku access token dalam detik.
func (m *Manager) AccessExpireSeconds() int {
	return int(m.accessExpire.Seconds())
}
