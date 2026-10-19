// Подписанный seed-список (12.3, шаг 2 bootstrap-каскада): свежий список
// адресов пиров, подписанный мультиподписью N-из-M проекта (core/threshold).
// Каналы доставки (Telegram/WhatsApp) — вне кода: здесь только формат
// полезной нагрузки. Ключи проекта — синтетические для тестов; реальные —
// трек «Ключи проекта» (раздел 13).
//
// Формат полезной нагрузки (байты):
//
//	[u8 версия = 1][network_id blob][timestamp u64][n_peers u64]
//	  n × [peer_id blob][addr blob]
//
// Список подписей: wire.ThresholdSignature (подписи ECDSA P-256 над
// SHA-256(payload)); проверка — threshold.Verify.
package net

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/threshold"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// cryptoMarshalPub — несжатая точка P-256 (65 байт).
func cryptoMarshalPub(k *ecdsa.PrivateKey) []byte {
	return crypto.MarshalPubKey(&k.PublicKey)
}

// SeedListMaxAge — максимальный возраст seed-списка (лучший effort: список
// «свежий», 12.3). Локальное значение 7 суток — предположение этапа 4
// (в Документе числа нет; помечено в отчёте).
const SeedListMaxAge = 7 * 24 * time.Hour

// SeedList — разобранный seed-список.
type SeedList struct {
	NetworkID []byte
	Timestamp uint64 // Unix-секунды выпуска списка
	Peers     []peerAddrMsg
}

// EncodeSeedList — полезная нагрузка seed-списка.
func (s *SeedList) payload() []byte {
	var e enc
	e.u64(1) // версия формата
	e.blob(s.NetworkID)
	e.u64(s.Timestamp)
	e.u64(uint64(len(s.Peers)))
	for i := range s.Peers {
		e.str(s.Peers[i].ID)
		e.u64(uint64(len(s.Peers[i].Addrs)))
		for _, a := range s.Peers[i].Addrs {
			e.str(a)
		}
	}
	return e.b
}

// SignSeedList — собирает подписанный список: payload + подписи N подписантов.
func SignSeedList(s *SeedList, keys []*ecdsa.PrivateKey) []byte {
	payload := s.payload()
	sig := &wire.ThresholdSignature{}
	h := sha256.Sum256(payload)
	for _, k := range keys {
		sigBytes, err := ecdsa.SignASN1(nil, k, h[:]) //nolint:staticcheck // rand nil = crypto/rand
		if err != nil {
			continue
		}
		pub := wire.SignatureShare{Signer: seedSignerID(k), Signature: sigBytes}
		sig.Signatures = append(sig.Signatures, pub)
	}
	var out enc
	out.blob(payload)
	out.blob(sig.Encode())
	return out.b
}

// seedSignerID — идентификатор подписанта: SHA-256(pubkey) (адрес подписанта
// в пространстве ключей проекта).
func seedSignerID(k *ecdsa.PrivateKey) []byte {
	h := sha256.Sum256(cryptoMarshalPub(k))
	return h[:]
}

// VerifySeedList — проверяет подписи (N-из-M через core/threshold) и
// разбирает список. pubKeys — ключи подписантов проекта (идентификатор
// подписанта = SHA-256(pubkey)). now — текущее время для проверки свежести
// (0 — не проверять; для тестов).
func VerifySeedList(blob []byte, networkID []byte, thresholdN uint64,
	pubKeys map[string]*ecdsa.PublicKey, now uint64) (*SeedList, error) {

	d := &dec{b: blob}
	payload, err := d.blob()
	if err != nil {
		return nil, errors.New("seedlist: некорректный блоб")
	}
	sigRaw, err := d.blob()
	if err != nil {
		return nil, errors.New("seedlist: некорректный блок подписей")
	}
	if err := d.done(); err != nil {
		return nil, errors.New("seedlist: лишние байты")
	}
	sig, err := decodeThresholdSig(sigRaw)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(payload)
	participants := make([][]byte, 0, len(pubKeys))
	for id := range pubKeys {
		participants = append(participants, []byte(id))
	}
	pubOf := func(signer []byte) (*ecdsa.PublicKey, bool) {
		pub, ok := pubKeys[string(signer)]
		return pub, ok
	}
	valid := threshold.Verify(h, participants, thresholdN, sig.Signatures, pubOf)
	if uint64(valid) < thresholdN {
		return nil, errors.New("seedlist: недостаточно подписей (N-из-M)")
	}
	s, err := parseSeedListPayload(payload)
	if err != nil {
		return nil, err
	}
	if now != 0 && s.Timestamp+uint64(SeedListMaxAge.Seconds()) < now {
		return nil, errors.New("seedlist: список устарел")
	}
	return s, nil
}

// decodeThresholdSig — разбор набора подписей (wire.ThresholdSignature;
// кодирование — protobuf: поле 1 = SignatureShare).
func decodeThresholdSig(b []byte) (*wire.ThresholdSignature, error) {
	d := &dec{b: b}
	sig := &wire.ThresholdSignature{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, errors.New("seedlist: некорректные подписи")
		}
		if f != 1 || wt != 2 {
			return nil, errors.New("seedlist: некорректные подписи")
		}
		raw, err := d.blob()
		if err != nil {
			return nil, errors.New("seedlist: некорректные подписи")
		}
		share, err := decodeSignatureShare(raw)
		if err != nil {
			return nil, err
		}
		sig.Signatures = append(sig.Signatures, *share)
	}
	if len(sig.Signatures) > 32 {
		return nil, errors.New("seedlist: слишком много подписей")
	}
	return sig, nil
}

// decodeSignatureShare — wire.SignatureShare (поле 1 = подписант, 2 = подпись).
func decodeSignatureShare(b []byte) (*wire.SignatureShare, error) {
	d := &dec{b: b}
	share := &wire.SignatureShare{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, errors.New("seedlist: некорректная подпись")
		}
		switch {
		case f == 1 && wt == 2:
			if share.Signer, err = d.blob(); err != nil {
				return nil, errors.New("seedlist: некорректная подпись")
			}
		case f == 2 && wt == 2:
			if share.Signature, err = d.blob(); err != nil {
				return nil, errors.New("seedlist: некорректная подпись")
			}
		default:
			return nil, errors.New("seedlist: некорректная подпись")
		}
	}
	if len(share.Signer) == 0 || len(share.Signature) == 0 {
		return nil, errors.New("seedlist: неполная подпись")
	}
	return share, nil
}

// parseSeedListPayload — разбор полезной нагрузки.
func parseSeedListPayload(payload []byte) (*SeedList, error) {
	d := &dec{b: payload}
	ver, err := d.u64()
	if err != nil || ver != 1 {
		return nil, errors.New("seedlist: неизвестная версия")
	}
	s := &SeedList{}
	if s.NetworkID, err = d.blob(); err != nil {
		return nil, errors.New("seedlist: нет network_id")
	}
	if s.Timestamp, err = d.u64(); err != nil {
		return nil, errors.New("seedlist: нет timestamp")
	}
	n, err := d.u64()
	if err != nil {
		return nil, errors.New("seedlist: некорректное число пиров")
	}
	if n > 1024 {
		return nil, errors.New("seedlist: слишком много пиров")
	}
	for i := uint64(0); i < n; i++ {
		var p peerAddrMsg
		if p.ID, err = d.str(); err != nil {
			return nil, errors.New("seedlist: некорректный пир")
		}
		na, err := d.u64()
		if err != nil {
			return nil, errors.New("seedlist: некорректные адреса")
		}
		if na > 64 {
			return nil, errors.New("seedlist: слишком много адресов")
		}
		for j := uint64(0); j < na; j++ {
			a, err := d.str()
			if err != nil {
				return nil, errors.New("seedlist: некорректный адрес")
			}
			p.Addrs = append(p.Addrs, a)
		}
		s.Peers = append(s.Peers, p)
	}
	if err := d.done(); err != nil {
		return nil, errors.New("seedlist: лишние байты")
	}
	return s, nil
}
