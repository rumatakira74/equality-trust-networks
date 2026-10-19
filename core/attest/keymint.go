// Разбор РЕАЛЬНОГО расширения аттестации Android KeyMint/Keymaster
// (этап 6, вопрос №7): OID 1.3.6.1.4.1.11129.2.1.17. Синтетический формат
// этапа 3 остаётся для тестов и песочницы; Verify сначала пробует его,
// затем — реальный формат.
//
// Схема (Android Key Attestation):
//
//	KeyDescription ::= SEQUENCE {
//	    attestationVersion        INTEGER,
//	    attestationSecurityLevel  ENUMERATED,   -- 0 Software, 1 TEE, 2 StrongBox
//	    keymasterVersion          INTEGER,      -- keyMintVersion
//	    keymasterSecurityLevel    ENUMERATED,
//	    attestationChallenge      OCTET STRING, -- задаёт узел (R4)
//	    uniqueId                  OCTET STRING,
//	    softwareEnforced          AuthorizationList,
//	    teeEnforced               AuthorizationList }
//
//	AuthorizationList ::= SEQUENCE { ... тегированные поля ... ,
//	    rootOfTrust   [704] EXPLICIT RootOfTrust OPTIONAL,
//	    osVersion     [705] EXPLICIT INTEGER OPTIONAL,
//	    osPatchLevel  [706] EXPLICIT INTEGER OPTIONAL, -- YYYYMM
//	    ... }
//
//	RootOfTrust ::= SEQUENCE {
//	    verifiedBootKey   OCTET STRING,
//	    deviceLocked      BOOLEAN,
//	    verifiedBootState ENUMERATED, -- 0 Verified, 1 SelfSigned, ...
//	    verifiedBootHash  OCTET STRING }
package attest

import (
	"encoding/asn1"
	"fmt"
)

// Уровни защиты KeyMint (совпадают с константами пакета).
const (
	kmLevelSoftware  = 0
	kmLevelTEE       = 1
	kmLevelStrongBox = 2
)

// kmKeyDescription — верхний уровень расширения KeyMint.
type kmKeyDescription struct {
	AttestationVersion       int
	AttestationSecurityLevel int // ENUMERATED -> int
	KeymasterVersion         int
	KeymasterSecurityLevel   int
	AttestationChallenge     []byte
	UniqueID                 []byte
	SoftwareEnforced         asn1.RawValue
	TeeEnforced              asn1.RawValue
}

// kmRootOfTrust — содержимое поля [704] RootOfTrust.
type kmRootOfTrust struct {
	VerifiedBootKey   []byte
	DeviceLocked      bool
	VerifiedBootState int
	VerifiedBootHash  []byte
}

// parseKeymint — разбор реального расширения аттестации. Возвращает
// приведённую Extension + challenge (R4).
func parseKeymint(extDER []byte) (Extension, []byte, error) {
	var kd kmKeyDescription
	rest, err := asn1.Unmarshal(extDER, &kd)
	if err != nil {
		return Extension{}, nil, fmt.Errorf("keymint: KeyDescription: %w", err)
	}
	if len(rest) != 0 {
		return Extension{}, nil, fmt.Errorf("keymint: лишние данные в KeyDescription")
	}
	if kd.AttestationSecurityLevel < kmLevelSoftware ||
		kd.AttestationSecurityLevel > kmLevelStrongBox {
		return Extension{}, nil, fmt.Errorf("keymint: неизвестный уровень защиты %d",
			kd.AttestationSecurityLevel)
	}
	ext := Extension{SecurityLevel: kd.AttestationSecurityLevel}
	// RootOfTrust и osPatchLevel — в teeEnforced (аппаратная часть);
	// на части устройств дублируются в softwareEnforced — ищем в обоих
	ext.VerifiedBootState, ext.DeviceLocked, ext.PatchYear, ext.PatchMonth,
		err = scanAuthorizationList(kd.TeeEnforced.Bytes)
	if err != nil {
		ext.VerifiedBootState, ext.DeviceLocked, ext.PatchYear, ext.PatchMonth,
			err = scanAuthorizationList(kd.SoftwareEnforced.Bytes)
	}
	if err != nil {
		return Extension{}, nil, err
	}
	return ext, kd.AttestationChallenge, nil
}

// scanAuthorizationList — сканирование тегированных полей AuthorizationList:
// [704] RootOfTrust (deviceLocked, verifiedBootState) и [706] osPatchLevel
// (INTEGER YYYYMM). Поля — context-specific constructed (EXPLICIT):
// содержимое поля — полный вложенный DER-элемент.
func scanAuthorizationList(b []byte) (bootState int, deviceLocked bool,
	patchYear, patchMonth int, err error) {
	bootState, deviceLocked = -1, false
	haveRoot, havePatch := false, false
	d := &derScanner{b: b}
	for d.more() {
		tag, content, e := d.next()
		if e != nil {
			return 0, false, 0, 0, e
		}
		switch tag {
		case 704: // RootOfTrust
			var rot kmRootOfTrust
			if _, e := asn1.Unmarshal(content, &rot); e != nil {
				return 0, false, 0, 0, fmt.Errorf("keymint: RootOfTrust: %w", e)
			}
			bootState, deviceLocked = rot.VerifiedBootState, rot.DeviceLocked
			haveRoot = true
		case 706: // osPatchLevel: INTEGER YYYYMM
			var pl int
			if _, e := asn1.Unmarshal(content, &pl); e != nil {
				return 0, false, 0, 0, fmt.Errorf("keymint: osPatchLevel: %w", e)
			}
			if pl < 200000 || pl > 300012 {
				return 0, false, 0, 0, fmt.Errorf("keymint: osPatchLevel %d не похож на YYYYMM", pl)
			}
			patchYear, patchMonth = pl/100, pl%100
			if patchMonth < 1 || patchMonth > 12 {
				return 0, false, 0, 0, fmt.Errorf("keymint: месяц патча %d вне 1..12", patchMonth)
			}
			havePatch = true
		}
	}
	if !haveRoot {
		return 0, false, 0, 0, fmt.Errorf("keymint: RootOfTrust [704] не найден")
	}
	if !havePatch {
		return 0, false, 0, 0, fmt.Errorf("keymint: osPatchLevel [706] не найден")
	}
	return bootState, deviceLocked, patchYear, patchMonth, nil
}

// derScanner — последовательный проход по DER-элементам (TLV).
type derScanner struct {
	b []byte
}

func (s *derScanner) more() bool { return len(s.b) > 0 }

// next — следующий элемент: номер тега (context-specific, включая
// длинную форму для номеров > 30 — 704/706) и содержимое.
func (s *derScanner) next() (int, []byte, error) {
	if len(s.b) < 2 {
		return 0, nil, fmt.Errorf("keymint: обрезанный DER")
	}
	id := s.b[0]
	s.b = s.b[1:]
	tagNum := int(id & 0x1f)
	if tagNum == 0x1f {
		// длинная форма номера тега: по 7 бит, старший бит — продолжение
		tagNum = 0
		for {
			if len(s.b) == 0 {
				return 0, nil, fmt.Errorf("keymint: обрезанный длинный тег")
			}
			b := s.b[0]
			s.b = s.b[1:]
			tagNum = tagNum<<7 | int(b&0x7f)
			if b&0x80 == 0 {
				break
			}
		}
	}
	// длина
	if len(s.b) == 0 {
		return 0, nil, fmt.Errorf("keymint: обрезанная длина DER")
	}
	l := s.b[0]
	s.b = s.b[1:]
	n := 0
	if l&0x80 != 0 {
		k := int(l & 0x7f)
		if k == 0 || k > 4 || len(s.b) < k {
			return 0, nil, fmt.Errorf("keymint: некорректная длина DER")
		}
		for i := 0; i < k; i++ {
			n = n<<8 | int(s.b[i])
		}
		s.b = s.b[k:]
	} else {
		n = int(l)
	}
	if len(s.b) < n {
		return 0, nil, fmt.Errorf("keymint: обрезанное содержимое DER")
	}
	content := s.b[:n]
	s.b = s.b[n:]
	return tagNum, content, nil
}
