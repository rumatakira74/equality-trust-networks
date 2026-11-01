// Деривация адреса из фингерпринта.
//
// PENDING: вопрос №2 — точная деривация не определена Документом (7.2 даёт
// только walletFingerprint = SHA-256(attestedPublicKey || protocolId) и индекс
// «фингерпринт -> адрес»). Рекомендация для пользователя (см. отчёт этапа 3):
//
//	address = SHA-256(fingerprint || network_id)
//
// Привязка к сети защищает от переноса адреса между сетями (12.3: узлы
// различают пиров по network_id). Функция заменяемая: DeriveAddress.
package crypto

import "crypto/sha256"

// DeriveAddress — заменяемая деривация адреса (32 байта). PENDING: вопрос №2.
var DeriveAddress = func(fingerprint, networkID []byte) []byte {
	h := sha256.New()
	h.Write(fingerprint)
	h.Write(networkID)
	return h.Sum(nil)
}

// AddressOfKey — адрес кошелька по публичному ключу и network_id:
// fingerprint = SHA-256(pub ‖ protocol_id) (7.2), затем адрес =
// SHA-256(fingerprint ‖ network_id). Единая точка деривации
// fingerprint-манифеста (v9.11): манифест хранит публичные ключи,
// адреса нигде не хранятся — каждое устройство выводит их из
// финального network_id.
func AddressOfKey(publicKey, networkID []byte) []byte {
	return DeriveAddress(Fingerprint(publicKey), networkID)
}
