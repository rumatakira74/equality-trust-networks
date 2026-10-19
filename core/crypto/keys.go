// Подписи ECDSA P-256 (secp256r1), DER-кодировка (spec/00 0.2, раздел 12.1).
package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
)

// ProtocolID — строка protocol_id (spec/00 0.2); заменяема до публикации
// (название продукта — вопрос 1 раздела 18).
var ProtocolID = []byte("tnp-v1")

// GenerateKey создаёт ключевую пару P-256.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// DeterministicKey — ключ, детерминированный строкой seed. Только для тестов
// и синтетических кошельков песочницы (6.5); не для продакшена.
func DeterministicKey(seed string) *ecdsa.PrivateKey {
	p256 := elliptic.P256()
	for i := uint32(0); ; i++ {
		var buf [36]byte
		h := sha256.Sum256([]byte("tnp-test-key:" + seed))
		copy(buf[:32], h[:])
		buf[32] = byte(i)
		buf[33] = byte(i >> 8)
		buf[34] = byte(i >> 16)
		buf[35] = byte(i >> 24)
		sum := sha256.Sum256(buf[:])
		d := new(big.Int).SetBytes(sum[:])
		if d.Sign() > 0 && d.Cmp(p256.Params().N) < 0 {
			priv := new(ecdsa.PrivateKey)
			priv.D = d
			priv.Curve = p256
			priv.X, priv.Y = p256.ScalarBaseMult(d.Bytes())
			return priv
		}
	}
}

// SignDER подписывает 32-байтовый хеш, возвращает DER-подпись.
func SignDER(priv *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
	if len(hash) != 32 {
		return nil, errors.New("crypto: хеш должен быть 32 байта")
	}
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash)
	if err != nil {
		return nil, err
	}
	return encodeDERSignature(r, s)
}

// VerifyDER проверяет DER-подпись P-256 над 32-байтовым хешем.
func VerifyDER(pub *ecdsa.PublicKey, hash, sig []byte) bool {
	if pub == nil || pub.Curve != elliptic.P256() || len(hash) != 32 {
		return false
	}
	r, s, err := decodeDERSignature(sig)
	if err != nil {
		return false
	}
	return ecdsa.Verify(pub, hash, r, s)
}

// MarshalPubKey — несжатая точка 0x04 || X(32) || Y(32), 65 байт.
func MarshalPubKey(pub *ecdsa.PublicKey) []byte {
	out := make([]byte, 65)
	out[0] = 4
	pub.X.FillBytes(out[1:33])
	pub.Y.FillBytes(out[33:65])
	return out
}

// ParsePubKey разбирает несжатую точку (65 байт) и проверяет кривую.
func ParsePubKey(b []byte) (*ecdsa.PublicKey, error) {
	if len(b) != 65 || b[0] != 4 {
		return nil, errors.New("crypto: ожидается несжатая точка P-256")
	}
	x := new(big.Int).SetBytes(b[1:33])
	y := new(big.Int).SetBytes(b[33:65])
	p256 := elliptic.P256()
	if !p256.IsOnCurve(x, y) {
		return nil, errors.New("crypto: точка не на кривой P-256")
	}
	return &ecdsa.PublicKey{Curve: p256, X: x, Y: y}, nil
}

// Fingerprint = SHA-256(attestedPublicKey || protocolId) (7.2).
func Fingerprint(attestedPublicKey []byte) []byte {
	h := sha256.New()
	h.Write(attestedPublicKey)
	h.Write(ProtocolID)
	return h.Sum(nil)
}

func FingerprintHex(fp []byte) string { return hex.EncodeToString(fp) }

func encodeDERSignature(r, s *big.Int) ([]byte, error) {
	rb, sb := r.Bytes(), s.Bytes()
	if len(rb) > 0 && rb[0]&0x80 != 0 {
		rb = append([]byte{0}, rb...)
	}
	if len(sb) > 0 && sb[0]&0x80 != 0 {
		sb = append([]byte{0}, sb...)
	}
	out := []byte{0x30, byte(4 + len(rb) + len(sb)), 0x02, byte(len(rb))}
	out = append(out, rb...)
	out = append(out, 0x02, byte(len(sb)))
	out = append(out, sb...)
	return out, nil
}

func decodeDERSignature(sig []byte) (*big.Int, *big.Int, error) {
	if len(sig) < 8 || sig[0] != 0x30 {
		return nil, nil, errors.New("crypto: не DER-подпись")
	}
	if int(sig[1]) != len(sig)-2 {
		return nil, nil, errors.New("crypto: DER-длина не сходится")
	}
	if sig[2] != 0x02 {
		return nil, nil, errors.New("crypto: DER: ожидается INTEGER r")
	}
	rl := int(sig[3])
	if rl == 0 || 4+rl+2 > len(sig) {
		return nil, nil, errors.New("crypto: DER: некорректная длина r")
	}
	r := new(big.Int).SetBytes(sig[4 : 4+rl])
	p := 4 + rl
	if sig[p] != 0x02 {
		return nil, nil, errors.New("crypto: DER: ожидается INTEGER s")
	}
	sl := int(sig[p+1])
	if sl == 0 || p+2+sl != len(sig) {
		return nil, nil, errors.New("crypto: DER: длина s не сходится")
	}
	s := new(big.Int).SetBytes(sig[p+2 : p+2+sl])
	if r.Sign() <= 0 || s.Sign() <= 0 {
		return nil, nil, errors.New("crypto: DER: нулевые компоненты")
	}
	return r, s, nil
}
