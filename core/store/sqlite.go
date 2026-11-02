// SQLite-хранилище (раздел 12.5): цепочки транзакций, индекс
// «фингерпринт -> адрес», чёрный список фингерпринтов, заголовки чекпоинтов,
// снимок состояния. Драйвер ncruces/go-sqlite3 (этап 16, №38): SQLite
// скомпилирован в WASM и исполняется через wazero (pure-Go, без CGo и без
// libc) — файловый I/O идёт через пакет os Go (openat/fstatat), raw
// legacy-сисколлы (lstat/stat/access) не излучаются ни на одной
// архитектуре — seccomp-политика приложения Android (API 35/36)
// не даёт SIGSYS. Предыдущий драйвер modernc.org/sqlite на linux/amd64
// (x86_64-эмуляторы) излучал raw SYS_LSTAT из musl-пути fstatat
// (libc._fstatat_kstat, быстрый путь AT_FDCWD+AT_SYMLINK_NOFOLLOW) —
// SIGSYS подтверждён tombstone на API 36; на arm64 modernc безопасен
// (ядро не реализует legacy-сисколлы), но эмуляторная разработка/CI
// идут на x86_64. Выбор драйвера — reports/REPORT-STAGE-16-NODE-PERSISTENCE.md.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Store — SQLite-хранилище ядра.
type Store struct {
	db *sql.DB
}

// Open открывает (и создаёт) базу.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS txs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			txid BLOB UNIQUE NOT NULL,
			wallet BLOB NOT NULL,
			seq INTEGER NOT NULL,
			raw BLOB NOT NULL,
			accepted_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_txs_wallet_seq ON txs(wallet, seq)`,
		`CREATE TABLE IF NOT EXISTS fingerprint_index (
			fingerprint BLOB PRIMARY KEY,
			address BLOB NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS fingerprint_blacklist (
			fingerprint BLOB PRIMARY KEY
		)`,
		`CREATE TABLE IF NOT EXISTS checkpoint_headers (
			epoch INTEGER PRIMARY KEY,
			hash BLOB NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS checkpoint_supply (
			epoch INTEGER PRIMARY KEY,
			supply INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value BLOB NOT NULL
		)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("store: migrate: %w", err)
		}
	}
	return nil
}

// SaveTx сохраняет принятую транзакцию (canonical-байты).
func (s *Store) SaveTx(tx *wire.Transaction, acceptedAt uint64) error {
	id := tx.TxID()
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO txs (txid, wallet, seq, raw, accepted_at) VALUES (?,?,?,?,?)`,
		id[:], tx.Wallet, tx.Seq, tx.Encode(), acceptedAt)
	return err
}

// LoadTxs возвращает все транзакции в каузальном порядке вставки — для
// replay (INV-3). В headless-ядре порядок вставки и есть причинный; в сети
// причинность обеспечивается финализацией (этап 5).
func (s *Store) LoadTxs() ([]*wire.Transaction, error) {
	rows, err := s.db.Query(`SELECT raw FROM txs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*wire.Transaction
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		tx, err := wire.DecodeTransaction(raw)
		if err != nil {
			return nil, fmt.Errorf("store: decode: %w", err)
		}
		out = append(out, tx)
	}
	return out, rows.Err()
}

// SaveFingerprint / LoadFingerprints — индекс «фингерпринт -> адрес».
func (s *Store) SaveFingerprint(fp, addr []byte) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO fingerprint_index (fingerprint, address) VALUES (?,?)`, fp, addr)
	return err
}

func (s *Store) LoadFingerprints() (map[string][]byte, error) {
	rows, err := s.db.Query(`SELECT fingerprint, address FROM fingerprint_index`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]byte)
	for rows.Next() {
		var fp, addr []byte
		if err := rows.Scan(&fp, &addr); err != nil {
			return nil, err
		}
		out[string(fp)] = addr
	}
	return out, rows.Err()
}

// SaveBlacklisted / LoadBlacklist — чёрный список фингерпринтов (миграция, 11).
func (s *Store) SaveBlacklisted(fp []byte) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO fingerprint_blacklist (fingerprint) VALUES (?)`, fp)
	return err
}

func (s *Store) LoadBlacklist() (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT fingerprint FROM fingerprint_blacklist`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var fp []byte
		if err := rows.Scan(&fp); err != nil {
			return nil, err
		}
		out[string(fp)] = true
	}
	return out, rows.Err()
}

// SaveCheckpointHeader / LatestCheckpointHeader — заголовки чекпоинтов (этап 5).
func (s *Store) SaveCheckpointHeader(epoch uint64, hash []byte) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO checkpoint_headers (epoch, hash) VALUES (?,?)`, epoch, hash)
	return err
}

// SaveCheckpointSupply — предложение на эпохе (total_supply финализированного
// чекпоинта) для адаптивной комиссии (10.2, этап 10).
func (s *Store) SaveCheckpointSupply(epoch, supply uint64) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO checkpoint_supply (epoch, supply) VALUES (?,?)`, epoch, supply)
	return err
}

// LoadCheckpointSupply — весь ряд предложения по возрастанию эпох.
func (s *Store) LoadCheckpointSupply() ([]uint64, []uint64, error) {
	rows, err := s.db.Query(`SELECT epoch, supply FROM checkpoint_supply ORDER BY epoch`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var epochs, supplies []uint64
	for rows.Next() {
		var e, v uint64
		if err := rows.Scan(&e, &v); err != nil {
			return nil, nil, err
		}
		epochs = append(epochs, e)
		supplies = append(supplies, v)
	}
	return epochs, supplies, rows.Err()
}

func (s *Store) LatestCheckpointHeader() (uint64, []byte, bool, error) {
	var epoch uint64
	var hash []byte
	err := s.db.QueryRow(`SELECT epoch, hash FROM checkpoint_headers ORDER BY epoch DESC LIMIT 1`).Scan(&epoch, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, err
	}
	return epoch, hash, true, nil
}

// SaveMeta / LoadMeta — прочие persistent-данные (genesis, снимок).
func (s *Store) SaveMeta(k string, v []byte) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES (?,?)`, k, v)
	return err
}

func (s *Store) LoadMeta(k string) ([]byte, bool, error) {
	var v []byte
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, k).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}
