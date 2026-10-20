// Декодирование genesis-пакета (манифест + genesis-запись). Аддитивный файл
// этапа 8 (закрытая тестовая сеть): до этого genesis кодировался, но не
// декодировался — сеть можно было собрать только кодом. Правило 3: файл
// только добавляет функции, существующее поведение ядра не меняется.
// Кодирование — genesis.go; формат — proto/genesis.proto, spec/02.
package wire

import (
	"errors"
	"fmt"
)

// DecodeSlotTier — ярус пула слотов поручителя (7.4).
func DecodeSlotTier(b []byte) (*SlotTier, error) {
	d := &decoder{b: b}
	p := &SlotTier{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.MinAgeDays = v
		case f == 2 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Slots = uint32(v)
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

// DecodePowParams — параметры PoW (№9).
func DecodePowParams(b []byte) (*PowParams, error) {
	d := &decoder{b: b}
	p := &PowParams{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		v, err := d.varint()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			p.TargetLight = v
		case f == 2 && wt == 0:
			p.Argon2MemoryKiB = uint32(v)
		case f == 3 && wt == 0:
			p.Argon2Iterations = uint32(v)
		case f == 4 && wt == 0:
			p.Argon2Parallelism = uint32(v)
		case f == 5 && wt == 0:
			p.TargetHard = v
		case f == 6 && wt == 0:
			p.RepeatAndroidIDTarget = v
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

// DecodeMutableBound — границы изменяемого параметра (4, 13).
func DecodeMutableBound(b []byte) (*MutableBound, error) {
	d := &decoder{b: b}
	p := &MutableBound{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			p.Param = string(v)
		case f == 2 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Min = v
		case f == 3 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			p.Max = v
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

// DecodeSeedMember — участник посевного круга.
func DecodeSeedMember(b []byte) (*SeedMember, error) {
	d := &decoder{b: b}
	p := &SeedMember{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if p.Address, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			if p.PublicKey, err = d.bytes(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

// DecodeSeedSunset — правило самоликвидации посевного круга (№15).
func DecodeSeedSunset(b []byte) (*SeedSunset, error) {
	d := &decoder{b: b}
	p := &SeedSunset{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		v, err := d.varint()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 0:
			p.UseActiveWallets = true
			p.ActiveWallets = v
		case f == 2 && wt == 0:
			p.DeadlineEpoch = v
		default:
			return nil, errUnknownField
		}
	}
	return p, nil
}

// DecodeProfileParams — параметры профиля (раздел 4). Каноничность:
// неизвестные поля запрещены (как в прочих Decode* ядра).
func DecodeProfileParams(b []byte) (*ProfileParams, error) {
	d := &decoder{b: b}
	p := &ProfileParams{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			if err := p.setU64(f, v); err != nil {
				return nil, err
			}
		case wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			switch f {
			case 2:
				p.ResourceID = string(v)
			case 18:
				tier, err := DecodeSlotTier(v)
				if err != nil {
					return nil, err
				}
				p.SlotTiers = append(p.SlotTiers, *tier)
			case 32:
				mb, err := DecodeMutableBound(v)
				if err != nil {
					return nil, err
				}
				p.MutableBounds = append(p.MutableBounds, *mb)
			case 34:
				p.Providers = append(p.Providers, v)
			case 52:
				pw, err := DecodePowParams(v)
				if err != nil {
					return nil, err
				}
				p.Pow = pw
			default:
				return nil, errUnknownField
			}
		default:
			return nil, fmt.Errorf("wire: неподдерживаемый wiretype %d", wt)
		}
	}
	return p, nil
}

// setU64 — varint-поле профиля по номеру (номера — genesis.go).
func (p *ProfileParams) setU64(f uint32, v uint64) error {
	switch f {
	case 1:
		p.Unit = UnitType(v)
	case 3:
		p.TransferMode = TransferMode(v)
	case 4:
		p.IssuanceMode = IssuanceMode(v)
	case 5:
		p.IssuanceAmount = v
	case 6:
		p.IssuancePeriodMonths = uint32(v)
	case 7:
		p.AccrualWindowPeriods = v
	case 8:
		p.FeeMode = FeeMode(v)
	case 9:
		p.FeeMin = v
	case 10:
		p.RateMin = v
	case 11:
		p.RateMax = v
	case 12:
		p.DemurrageRateMicros = v
	case 13:
		p.DemurrageFloor = v
	case 14:
		p.EntryMode = EntryMode(v)
	case 15:
		p.VouchK = uint32(v)
	case 16:
		p.VoucherMinAgeDays = v
	case 17:
		p.VoucherMinMintPeriods = uint32(v)
	case 19:
		p.SeedSlotCount = uint32(v)
	case 20:
		p.SlotFreeMintPeriods = uint32(v)
	case 21:
		p.SlotFreeDays = v
	case 22:
		p.VoucherRewardMicros = v
	case 23:
		p.VoucherRewardPeriods = uint32(v)
	case 24:
		p.DevicePolicy = DevicePolicy(v)
	case 25:
		p.SigningAuth = SigningAuth(v)
	case 26:
		p.NetworkAccess = NetworkAccess(v)
	case 27:
		p.CheckpointMode = CheckpointMode(v)
	case 28:
		p.EpochSeconds = v
	case 29:
		p.Sandbox = v != 0
	case 30:
		p.TimeScale = TimeScale(v)
	case 31:
		p.ParamsMutable = v != 0
	case 33:
		p.Modules = append(p.Modules, uint32(v))
	case 35:
		p.TransferMoratoriumDays = v
	case 36:
		p.TransferDelayHours = v
	case 37:
		p.DelayedThresholdDefault = v
	case 38:
		p.DelayedThresholdMax = v
	case 39:
		p.RegisteredExpiryDays = v
	case 40:
		p.GuardianPersonalThreshold = v
	case 41:
		p.GuardianPersonalCount = v
	case 42:
		p.GuardianPublicThreshold = v
	case 43:
		p.GuardianPublicCount = v
	case 44:
		p.GuardianMinAgeDays = v
	case 45:
		p.GuardianMaxWards = v
	case 46:
		p.GuardianPersonalMaxWards = v
	case 47:
		p.GuardianReward = v
	case 48:
		p.GuardianRewardCooldownDays = v
	case 49:
		p.MigrateDelayDays = v
	case 50:
		p.TimestampToleranceSeconds = v
	case 51:
		p.MintChallengeWindowHours = v
	case 53:
		p.FreshTransferWindowPeriods = v
	case 54:
		p.EclipseN = v
	case 55:
		p.ValidatorsM = v
	case 56:
		p.CpBuilderQueue = v
	case 57:
		p.FeeAdjustEpochs = v
	case 58:
		p.FeeTargetGrowthMicros = v
	case 59:
		p.FeeStepMicros = v
	default:
		return errUnknownField
	}
	return nil
}

// DecodeManifest — манифест сети (3.3, 4).
func DecodeManifest(b []byte) (*Manifest, error) {
	d := &decoder{b: b}
	m := &Manifest{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if m.NetworkID, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			m.DisplayName = string(v)
		case f == 3 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			prof, err := DecodeProfileParams(v)
			if err != nil {
				return nil, err
			}
			m.Profile = prof
		case f == 4 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			sm, err := DecodeSeedMember(v)
			if err != nil {
				return nil, err
			}
			m.SeedCircle = append(m.SeedCircle, *sm)
		case f == 5 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			ss, err := DecodeSeedSunset(v)
			if err != nil {
				return nil, err
			}
			m.SeedSunset = ss
		case f == 6 && wt == 2:
			if m.GenesisHash, err = d.bytes(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	if m.Profile == nil {
		return nil, errors.New("wire: манифест без профиля")
	}
	return m, nil
}

// DecodeGenesisRecord — genesis-запись (3.1, 3.3).
func DecodeGenesisRecord(b []byte) (*GenesisRecord, error) {
	d := &decoder{b: b}
	g := &GenesisRecord{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if g.NetworkID, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 2:
			if g.ManifestHash, err = d.bytes(); err != nil {
				return nil, err
			}
		case f == 3 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			g.GenesisTime = v
		case f == 4 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			g.EpochZero = v
		case f == 5 && wt == 0:
			v, err := d.varint()
			if err != nil {
				return nil, err
			}
			g.ProtocolVersion = uint32(v)
		case f == 6 && wt == 2:
			v, err := d.bytes()
			if err != nil {
				return nil, err
			}
			ss, err := DecodeSeedSunset(v)
			if err != nil {
				return nil, err
			}
			g.SeedSunset = ss
		case f == 7 && wt == 2:
			if g.InviteKey, err = d.bytes(); err != nil {
				return nil, err
			}
		default:
			return nil, errUnknownField
		}
	}
	return g, nil
}

// DecodeGenesisPackage — genesis-пакет целиком: манифест (поле 1) +
// genesis-запись (поле 2). Формат файла: genesis.bin утилиты
// cmd/tnp-testnet (этап 8) — им же и кодируется.
func DecodeGenesisPackage(b []byte) (*GenesisPackage, error) {
	d := &decoder{b: b}
	gp := &GenesisPackage{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		if wt != 2 {
			return nil, fmt.Errorf("wire: genesis-пакет: wiretype %d у поля %d", wt, f)
		}
		v, err := d.bytes()
		if err != nil {
			return nil, err
		}
		switch f {
		case 1:
			m, err := DecodeManifest(v)
			if err != nil {
				return nil, err
			}
			gp.Manifest = m
		case 2:
			g, err := DecodeGenesisRecord(v)
			if err != nil {
				return nil, err
			}
			gp.Genesis = g
		default:
			return nil, errUnknownField
		}
	}
	if gp.Manifest == nil || gp.Genesis == nil {
		return nil, errors.New("wire: genesis-пакет неполон (нужны манифест и запись)")
	}
	return gp, nil
}

// EncodeGenesisPackage — genesis-пакет одним блобом (парен к DecodeGenesisPackage):
// поле 1 — манифест, поле 2 — genesis-запись.
func (gp *GenesisPackage) EncodeGenesisPackage() []byte {
	var b []byte
	b = appendMessageField(b, 1, gp.Manifest.Encode())
	b = appendMessageField(b, 2, gp.Genesis.Encode())
	return b
}
