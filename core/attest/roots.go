// Корни аппаратной аттестации Google (A2, этап 14; 7.2). Источник —
// официальная страница Key Attestation (developer.android.com/privacy-and-security/
// security-key-attestation, раздел Root certificates); сертификаты берутся
// ТОЛЬКО оттуда, из памяти не воспроизводятся. Ротация корня — новый PEM-файл
// в roots/ и коммит (v9.12).
//
// Действующий RSA-корень (serial f92009e853b6b045, 2022-2042) и новый
// ECDSA P-384-корень «Key Attestation CA1» (2025-2035; Remote Key
// Provisioning — цепочки под ним с 01.02.2026). Цепочка валидна, если
// доходит до любого из встроенных корней.
package attest

import (
	"crypto/x509"
	"embed"
	"encoding/pem"
	"errors"
	"strings"
	"sync"
)

//go:embed roots/*.pem
var rootsFS embed.FS

var (
	googleRootsOnce sync.Once
	googleRoots     []*x509.Certificate
	googleRootsErr  error
)

// GoogleRoots — встроенные корневые сертификаты Google hardware
// attestation (оба корня; 7.2). Прод-путь аттестации: узел доверяет
// только этим корням.
func GoogleRoots() ([]*x509.Certificate, error) {
	googleRootsOnce.Do(func() {
		entries, err := rootsFS.ReadDir("roots")
		if err != nil {
			googleRootsErr = err
			return
		}
		for _, e := range entries {
			data, err := rootsFS.ReadFile("roots/" + e.Name())
			if err != nil {
				googleRootsErr = err
				return
			}
			for len(data) > 0 {
				var block *pem.Block
				block, data = pem.Decode(data)
				if block == nil {
					break
				}
				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					googleRootsErr = err
					return
				}
				googleRoots = append(googleRoots, cert)
			}
		}
		if len(googleRoots) == 0 {
			googleRootsErr = errors.New("attest: пустой набор корней Google")
		}
	})
	return googleRoots, googleRootsErr
}

// syntheticRootCN — префикс CN синтетических корней (SyntheticRoot).
const syntheticRootCN = "TNP Synthetic Root"

// IsSyntheticRoot — тестовый (синтетический) корень аттестации, не Google.
// По префиксу CN: все синтетические корни (песочница, векторы, утилиты)
// создаются attest.SyntheticRoot с CN «TNP Synthetic Root <seed>».
func IsSyntheticRoot(cert *x509.Certificate) bool {
	return cert != nil && strings.HasPrefix(cert.Subject.CommonName, syntheticRootCN)
}
