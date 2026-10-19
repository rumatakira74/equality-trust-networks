// Разбор цепочки аппаратной аттестации Android (7.2; этап 3 — синтетические
// цепочки, реальные корни Google — этап 6).
//
// Формат цепочки в RegisterPayload.AttestationChain: конкатенация DER-
// сертификатов с 4-байтовым big-endian префиксом длины: лист первым, затем
// промежуточные (корни НЕ передаются — они зашиты в APK, 7.2).
//
// Расширение аттестации (OID 1.3.6.1.4.1.11129.2.1.17) в синтетических цепочках
// содержит упрощённую структуру (реальный формат Keymaster — этап 6):
//
//	SEQUENCE {
//	  securityLevel      ENUMERATED (0 Software, 1 TEE, 2 StrongBox),
//	  verifiedBootState  ENUMERATED (0 Verified, 1 SelfSigned, ...),
//	  deviceLocked       BOOLEAN,
//	  patchYear          INTEGER,
//	  patchMonth         INTEGER (1-12)
//	}
package attest

import (
	"crypto/ecdsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"math/big"
	"time"
)

var oidAttestation = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 17}

// Уровни защиты (7.2: StrongBox/TEE).
const (
	LevelSoftware  = 0
	LevelTEE       = 1
	LevelStrongBox = 2
)

// VerifiedBootState (7.2: verifiedBootState = Verified).
const (
	BootVerified   = 0
	BootSelfSigned = 1
	BootUnverified = 2
	BootFailed     = 3
)

// Extension — упрощённое содержание расширения аттестации.
type Extension struct {
	SecurityLevel     int
	VerifiedBootState int
	DeviceLocked      bool
	PatchYear         int
	PatchMonth        int
}

type asnExt struct {
	SecurityLevel     int
	VerifiedBootState int
	DeviceLocked      bool
	PatchYear         int
	PatchMonth        int
}

func (e Extension) marshal() ([]byte, error) {
	return asn1.Marshal(asnExt{e.SecurityLevel, e.VerifiedBootState, e.DeviceLocked, e.PatchYear, e.PatchMonth})
}

func parseExtension(b []byte) (Extension, error) {
	var a asnExt
	if _, err := asn1.Unmarshal(b, &a); err != nil {
		return Extension{}, err
	}
	return Extension{a.SecurityLevel, a.VerifiedBootState, a.DeviceLocked, a.PatchYear, a.PatchMonth}, nil
}

// Chain — разобранная цепочка: лист + промежуточные. Challenge —
// attestation_challenge из расширения (реальные цепочки KeyMint — этап 6;
// синтетические — расширение 1.3.6.1.4.1.11129.2.1.18); связывает цепочку
// с регистрацией (R4).
type Chain struct {
	Leaf          *x509.Certificate
	Intermediates []*x509.Certificate
	Extension     Extension
	Challenge     []byte
}

// EncodeChain — упаковка сертификатов в блоб (лист первым).
func EncodeChain(certs ...[]byte) []byte {
	var out []byte
	for _, c := range certs {
		var l [4]byte
		binary.BigEndian.PutUint32(l[:], uint32(len(c)))
		out = append(out, l[:]...)
		out = append(out, c...)
	}
	return out
}

// DecodeChain разбирает блоб цепочки.
func DecodeChain(blob []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for len(blob) > 0 {
		if len(blob) < 4 {
			return nil, errors.New("attest: обрезанный префикс длины")
		}
		n := binary.BigEndian.Uint32(blob[:4])
		blob = blob[4:]
		if uint32(len(blob)) < n {
			return nil, errors.New("attest: обрезанный сертификат")
		}
		c, err := x509.ParseCertificate(blob[:n])
		if err != nil {
			return nil, err
		}
		certs = append(certs, c)
		blob = blob[n:]
	}
	if len(certs) == 0 {
		return nil, errors.New("attest: пустая цепочка")
	}
	return certs, nil
}

// Verify проверяет цепочку аттестации (R2, spec/03 3.2).
//
// Требования (7.2): цепочка до доверенного корня; аппаратный уровень
// (securityLevel >= minLevel); verifiedBootState = Verified; deviceLocked;
// уровень патча не старше maxPatchAgeMonths от now.
func Verify(chainBlob []byte, trustedRoots []*x509.Certificate, minLevel int,
	maxPatchAgeMonths int, now time.Time) (*Chain, error) {
	certs, err := DecodeChain(chainBlob)
	if err != nil {
		return nil, err
	}
	leaf := certs[0]
	// поиск расширения аттестации в листе
	var extRaw []byte
	for _, ext := range leaf.Extensions {
		if ext.Id.Equal(oidAttestation) {
			extRaw = ext.Value
			break
		}
	}
	if extRaw == nil {
		return nil, errors.New("attest: нет расширения аттестации в листе")
	}
	// синтетический формат (этап 3, песочница) или реальный KeyMint (этап 6)
	ext, err := parseExtension(extRaw)
	var challenge []byte
	if err != nil {
		ext, challenge, err = parseKeymint(extRaw)
		if err != nil {
			return nil, errors.New("attest: расширение аттестации не разбирается")
		}
	} else {
		// challenge синтетической цепочки — расширение 2.1.18
		for _, e := range leaf.Extensions {
			if e.Id.Equal(asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 18}) {
				challenge = e.Value
				break
			}
		}
	}
	inter := certs[1:]
	// верификация цепочки: лист + промежуточные -> доверенный корень
	pool := x509.NewCertPool()
	for _, c := range inter {
		pool.AddCert(c)
	}
	roots := x509.NewCertPool()
	for _, r := range trustedRoots {
		roots.AddCert(r)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, Intermediates: pool, CurrentTime: now}); err != nil {
		return nil, err
	}
	if ext.SecurityLevel < minLevel {
		return nil, errors.New("attest: аппаратный уровень ниже требуемого")
	}
	if ext.VerifiedBootState != BootVerified {
		return nil, errors.New("attest: verifiedBootState != Verified")
	}
	if !ext.DeviceLocked {
		return nil, errors.New("attest: deviceLocked = false")
	}
	patch := time.Date(ext.PatchYear, time.Month(ext.PatchMonth), 1, 0, 0, 0, 0, time.UTC)
	months := int(now.Sub(patch).Hours() / 24 / 30)
	if months > maxPatchAgeMonths {
		return nil, errors.New("attest: уровень патча просрочен")
	}
	return &Chain{Leaf: leaf, Intermediates: inter, Extension: ext, Challenge: challenge}, nil
}

// SyntheticRoot — детерминированный доверенный корень для тестов (не Google).
func SyntheticRoot(seed string, key *ecdsa.PrivateKey) (*x509.Certificate, []byte, error) {
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "TNP Synthetic Root " + seed},
		NotBefore:             time.Unix(0, 0),
		NotAfter:              time.Unix(4102444800, 0), // 2100-01-01
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(nil, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	return cert, der, nil
}

// BuildLeaf создаёт лист аттестации: ключ кошелька + расширение аттестации,
// подписанный промежуточным (или корневым) ключом.
func BuildLeaf(subjectKey *ecdsa.PublicKey, challenge []byte, ext Extension,
	signer *ecdsa.PrivateKey, signerCert *x509.Certificate, serial int64) ([]byte, error) {
	extDER, err := ext.marshal()
	if err != nil {
		return nil, err
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "TNP Synthetic Attestation"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(4102444800, 0),
		ExtraExtensions: []pkix.Extension{
			{Id: oidAttestation, Value: extDER},
			{Id: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 18}, Value: challenge},
		},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	return x509.CreateCertificate(nil, tpl, signerCert, subjectKey, signer)
}

// ChallengeFormula — параметризуемая формула attestation_challenge (R4).
// PENDING: вопрос №23 — формула не дана Документом; предположение по умолчанию:
// SHA-256(attestedPublicKey || protocol_id || network_id) — связывает ключ,
// протокол и сеть (setAttestationChallenge, 7.2).
var ChallengeFormula = func(attestedPublicKey, protocolID, networkID []byte) []byte {
	// SHA-256(pub || protocol_id || network_id)
	h := newHasher()
	h.Write(attestedPublicKey)
	h.Write(protocolID)
	h.Write(networkID)
	return h.Sum(nil)
}
