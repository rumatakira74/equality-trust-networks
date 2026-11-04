// Тесты фикса C1 аудита Sonnet-2 (тема fix-cp-threshold) на уровне сети:
// подписант не подписывает чужой StateRoot, предложение не наступившей
// эпохи не регистрируется, строгий режим (cp_set_check = 1) сверяет
// состав/порог с режимом манифеста, валидатор темы чекпоинтов отвергает
// структурно невозможные предложения.
package net

import (
	"bytes"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// cpProposal — предложение чекпоинта эпохи с заданным корнем состояния и
// составом (посевной круг V1..V3, порог 2 — переходный режим тестового env).
func cpProposal(e *testEnv, epoch, netTime uint64, root []byte, part [][]byte, thr uint64) *wire.Checkpoint {
	return &wire.Checkpoint{
		NetworkID:          e.gp.Genesis.NetworkID,
		Epoch:              epoch,
		StateRoot:          root,
		PrevCheckpointHash: make([]byte, 32),
		NetworkTime:        netTime,
		Participants:       part,
		Threshold:          thr,
	}
}

// cpSeedSet — состав посевного круга тестового env (V1, V2, V3) и порог 2.
func cpSeedSet(e *testEnv) ([][]byte, uint64) {
	return [][]byte{e.addr("V1"), e.addr("V2"), e.addr("V3")}, 2
}

// TestCpSignGateForeignStateRoot — подписант не подписывает предложение,
// чей StateRoot не совпадает с его реестром (C1, шаг 2): иначе подпись
// удостоверяет чужой StateRoot/TotalSupply/FinalizedTxIDs.
func TestCpSignGateForeignStateRoot(t *testing.T) {
	e := newTestEnv(t)
	n := e.consNode("V3", nodeOpts{level: LevelFull})
	part, thr := cpSeedSet(e)
	due := e.gp.Genesis.GenesisTime + n.ConsParams.EpochSeconds

	// чужой корень — подписи нет
	foreign := cpProposal(e, 1, due, bytes.Repeat([]byte{0xAA}, 32), part, thr)
	if n.cpShouldSign(foreign) {
		t.Fatal("подпись под чужим StateRoot разрешена")
	}
	n.onCpMessage(foreign, wire.SignatureShare{}, false)
	if got := n.CpCollectors[string(cpHashOf(foreign))]; got != nil && got.Count() != 0 {
		t.Fatalf("подписант поставил долю под чужим корнем: %d", got.Count())
	}

	// собственный корень — подпись есть (контроль: правило не блокирует всё)
	own := cpProposal(e, 1, due, cons.StateRoot(n.Ledger), part, thr)
	if !n.cpShouldSign(own) {
		t.Fatal("подпись под собственным StateRoot не разрешена")
	}
}

// TestCpSignGateEpochNotArrived — предложение следующей эпохи,
// опубликованное до её начала, не регистрируется (C1, шаг 3): иначе
// «первое увиденное побеждает» отдаёт эпоху тому, кто опубликовал раньше
// честного строителя. Наступившее предложение регистрируется.
func TestCpSignGateEpochNotArrived(t *testing.T) {
	e := newTestEnv(t)
	part, thr := cpSeedSet(e)
	root := bytes.Repeat([]byte{0x11}, 32)

	early := e.consNode("V3", nodeOpts{level: LevelFull})
	due := e.gp.Genesis.GenesisTime + early.ConsParams.EpochSeconds
	cpEarly := cpProposal(e, 1, due-1, root, part, thr)
	early.onCpMessage(cpEarly, wire.SignatureShare{}, false)
	if early.hasProposalForEpochLocked(1) {
		t.Fatal("предложение не наступившей эпохи зарегистрировано")
	}

	ok := e.consNode("V3", nodeOpts{level: LevelFull})
	cpDue := cpProposal(e, 1, due, root, part, thr)
	ok.onCpMessage(cpDue, wire.SignatureShare{}, false)
	if !ok.hasProposalForEpochLocked(1) {
		t.Fatal("предложение наступившей эпохи не зарегистрировано")
	}
}

// TestCpSignGateOwnVariant — при собственном предложении эпохи чужой
// вариант той же эпохи не подписывается (C1, шаг 3: принадлежность
// строителю; два варианта одной эпохи раскалывают набор подписантов,
// блокер 8.1).
func TestCpSignGateOwnVariant(t *testing.T) {
	e := newTestEnv(t)
	n := e.consNode("V3", nodeOpts{level: LevelFull})
	part, thr := cpSeedSet(e)
	due := e.gp.Genesis.GenesisTime + n.ConsParams.EpochSeconds
	root := cons.StateRoot(n.Ledger)

	own := cpProposal(e, 1, due, root, part, thr)
	n.mu.Lock()
	n.lastProposal = own
	n.lastProposalEpoch = 1
	n.mu.Unlock()
	if !n.cpShouldSign(own) {
		t.Fatal("собственное предложение эпохи не подписывается")
	}
	variant := cpProposal(e, 1, due+1, root, part, thr) // другой network_time — другой хеш
	if n.cpShouldSign(variant) {
		t.Fatal("чужой вариант эпохи подписан при собственном предложении")
	}
}

// TestCpStrictParticipantsCheck — строгий режим (cp_set_check = 1): состав
// и порог сверяются с режимом манифеста, подмена состава отвергается
// (C1, шаг 1). Наследие (0) — сверки нет, сети прежних манифестов
// работают как раньше.
func TestCpStrictParticipantsCheck(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) { p.CpSetCheck = 1 })
	n := e.consNode("V3", nodeOpts{level: LevelFull})
	part, thr := cpSeedSet(e)
	due := e.gp.Genesis.GenesisTime + n.ConsParams.EpochSeconds
	root := cons.StateRoot(n.Ledger)

	good := cpProposal(e, 1, due, root, part, thr)
	if err := n.cpCheckParticipants(good); err != nil {
		t.Fatalf("состав по режиму отвергнут: %v", err)
	}
	// подмена состава (свой кошелёк + порог 1) — вариант атаки из отчёта
	forged := cpProposal(e, 1, due, root, [][]byte{e.addr("V3")}, 1)
	if err := n.cpCheckParticipants(forged); err == nil {
		t.Fatal("подменённый состав принят в строгом режиме")
	}
	// подмена порога при верном составе
	forgedThr := cpProposal(e, 1, due, root, part, 1)
	if err := n.cpCheckParticipants(forgedThr); err == nil {
		t.Fatal("подменённый порог принят в строгом режиме")
	}

	// наследие: манифест без поля 62 — сверки нет
	legacy := newTestEnv(t)
	nl := legacy.consNode("V3", nodeOpts{level: LevelFull})
	lpart, lthr := cpSeedSet(legacy)
	lroot := cons.StateRoot(nl.Ledger)
	ldue := legacy.gp.Genesis.GenesisTime + nl.ConsParams.EpochSeconds
	if err := nl.cpCheckParticipants(cpProposal(legacy, 1, ldue, lroot, [][]byte{legacy.addr("V3")}, 1)); err != nil {
		t.Fatalf("наследие (cp_set_check = 0) отвергло чужой состав: %v", err)
	}
	if err := nl.cpCheckParticipants(cpProposal(legacy, 1, ldue, lroot, lpart, lthr)); err != nil {
		t.Fatalf("наследие отвергло состав по режиму: %v", err)
	}
}

// TestCpStrictParticipantsNetwork — регресс обратной совместимости
// строгого режима: валидные чекпоинты сети с cp_set_check = 1
// финализируются (сверка состава не отвергает честные предложения).
func TestCpStrictParticipantsNetwork(t *testing.T) {
	e := newTestEnv(t, func(p *wire.ProfileParams) { p.CpSetCheck = 1 })
	n1 := e.consNode("V1", nodeOpts{level: LevelFull})
	e.start(n1)
	n2 := e.consNode("V2", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)
	n3 := e.consNode("V3", nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n3)
	if !waitPeers(t, n2, 1, 15*time.Second) || !waitPeers(t, n3, 1, 15*time.Second) {
		t.Fatal("узлы не соединились")
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if n1.CheckpointCount() >= 1 && n2.CheckpointCount() >= 1 && n3.CheckpointCount() >= 1 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if n1.CheckpointCount() < 1 || n2.CheckpointCount() < 1 || n3.CheckpointCount() < 1 {
		t.Fatalf("строгий режим: чекпоинты не финализированы: %d/%d/%d",
			n1.CheckpointCount(), n2.CheckpointCount(), n3.CheckpointCount())
	}
	cp := n1.Chain[len(n1.Chain)-1]
	if cp.Threshold == 0 || cp.Threshold > uint64(len(cp.Participants)) {
		t.Fatalf("порог %d при составе %d", cp.Threshold, len(cp.Participants))
	}
}

// TestCpTopicValidator — валидатор темы чекпоинтов (C1, шаг 4): структурно
// невозможные предложения (порог 0, порог больше состава, чужая сеть,
// подписант вне состава, битый кадр) отвергаются до ретрансляции,
// валидное предложение проходит.
func TestCpTopicValidator(t *testing.T) {
	e := newTestEnv(t)
	n := e.consNode("V3", nodeOpts{level: LevelFull})
	part, thr := cpSeedSet(e)
	due := e.gp.Genesis.GenesisTime + n.ConsParams.EpochSeconds
	root := cons.StateRoot(n.Ledger)

	frame := func(cp *wire.Checkpoint, signer, sig []byte) []byte {
		var enc enc
		enc.blob(cp.Encode())
		enc.blob(signer)
		enc.blob(sig)
		return writeFrame(&enc, msgCpShare)
	}

	good := cpProposal(e, 1, due, root, part, thr)
	if _, err := n.cpAdmit(frame(good, nil, nil)); err != nil {
		t.Fatalf("валидное предложение отвергнуто: %v", err)
	}
	zeroThr := cpProposal(e, 1, due, root, part, 0)
	if _, err := n.cpAdmit(frame(zeroThr, nil, nil)); err == nil {
		t.Fatal("порог 0 прошёл валидатор темы")
	}
	bigThr := cpProposal(e, 1, due, root, part, thr+2)
	if _, err := n.cpAdmit(frame(bigThr, nil, nil)); err == nil {
		t.Fatal("порог больше состава прошёл валидатор темы")
	}
	foreignNet := cpProposal(e, 1, due, root, part, thr)
	foreignNet.NetworkID = bytes.Repeat([]byte{0x7F}, 32)
	if _, err := n.cpAdmit(frame(foreignNet, nil, nil)); err == nil {
		t.Fatal("чужая сеть прошла валидатор темы")
	}
	outsider := bytes.Repeat([]byte{0x5A}, 32)
	if _, err := n.cpAdmit(frame(good, outsider, bytes.Repeat([]byte{0x01}, 70))); err == nil {
		t.Fatal("подписант вне состава прошёл валидатор темы")
	}
	if _, err := n.cpAdmit([]byte{0xFF, 0xFF}); err == nil {
		t.Fatal("битый кадр прошёл валидатор темы")
	}
	// доля подписи участника проходит
	if _, err := n.cpAdmit(frame(good, part[0], bytes.Repeat([]byte{0x01}, 70))); err != nil {
		t.Fatalf("доля подписи участника отвергнута: %v", err)
	}
}
