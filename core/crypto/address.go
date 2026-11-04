// Деривация адреса из фингерпринта.
//
// Адрес — 32 байта, деривация фиксирована (v9.11, единая точка — DeriveAddress):
//
//	address = SHA-256(fingerprint || network_id)
//
// где fingerprint = SHA-256(attestedPublicKey || protocolId) (7.2). Привязка
// к сети защищает от переноса адреса между сетями (12.3: узлы различают
// пиров по network_id). Адреса нигде не хранятся: манифест хранит публичные
// ключи, каждое устройство выводит адрес из финального network_id
// (fingerprint-манифест, v9.11). Формула — часть протокола: её смена меняет
// адреса всех кошельков сети.
package crypto

import "crypto/sha256"

// DeriveAddress — деривация адреса (32 байта): SHA-256(fingerprint ||
// network_id). Единая точка деривации адресов сети.
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
