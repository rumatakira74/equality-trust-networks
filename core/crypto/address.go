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
