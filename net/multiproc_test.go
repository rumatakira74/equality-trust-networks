// Мультипроцессный интеграционный тест (4.11.1): headless-узлы отдельными
// процессами (cmd/tnp-node) на localhost/эмерджентных портах; синхронизация
// реестра между процессами. Отдельные машины — ручная проверка по
// инструкции из reports/REPORT-STAGE-4.md.
package net

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// procStatus — содержимое -status-file.
type procStatus struct {
	Level      string   `json:"level"`
	PeerID     string   `json:"peer_id"`
	Addrs      []string `json:"addrs"`
	Peers      int      `json:"peers"`
	JournalLen int      `json:"journal_len"`
	StateRoot  string   `json:"state_root"`
}

// procNode — запущенный процесс узла.
type procNode struct {
	t    *testing.T
	cmd  *exec.Cmd
	file string
}

func startProcNode(t *testing.T, bin, dir, name string, extra ...string) *procNode {
	t.Helper()
	args := append([]string{
		"-db", filepath.Join(dir, name+".db"),
		"-status-file", filepath.Join(dir, name+".status.json"),
		"-quiet",
	}, extra...)
	cmd := exec.Command(bin, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("запуск %s: %v", name, err)
	}
	pn := &procNode{t: t, cmd: cmd, file: filepath.Join(dir, name+".status.json")}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
		}
	})
	return pn
}

func (p *procNode) status() (procStatus, error) {
	blob, err := os.ReadFile(p.file)
	if err != nil {
		return procStatus{}, err
	}
	var st procStatus
	if err := json.Unmarshal(blob, &st); err != nil {
		return procStatus{}, err
	}
	return st, nil
}

func (p *procNode) requireStatus() procStatus {
	p.t.Helper()
	st, err := p.status()
	if err != nil {
		p.t.Fatalf("статус %s: %v", p.file, err)
	}
	return st
}

func (p *procNode) waitStatus(timeout time.Duration, cond func(procStatus) bool) procStatus {
	p.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if st, err := p.status(); err == nil && cond(st) {
			return st
		}
		time.Sleep(300 * time.Millisecond)
	}
	st := p.requireStatus()
	p.t.Fatalf("условие статуса не достигнуто за %v: %+v", timeout, st)
	return st
}

// TestMultiProcessSync — три процесса: A публикует (demo-tx), B подключается
// к A, C — к B; все сходятся к одному реестру (корни состояния равны).
func TestMultiProcessSync(t *testing.T) {
	if testing.Short() {
		t.Skip("мультипроцессный тест пропущен в -short")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "tnp-node")
	build := exec.Command("go", "build", "-o", bin, "./cmd/tnp-node")
	// тест выполняется из каталога net/; корень репозитория — на уровень выше
	build.Dir = ".."
	build.Stdout = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("сборка tnp-node: %v", err)
	}

	// узел A: публикует выпуск и перевод посевного кошелька
	a := startProcNode(t, bin, dir, "a", "-demo-tx", "-node-key", "proc-a")
	stA := a.waitStatus(15*time.Second, func(s procStatus) bool { return s.JournalLen >= 2 })
	if len(stA.Addrs) == 0 {
		t.Fatal("у A нет адресов")
	}
	peerA := stA.Addrs[0] + "/p2p/" + stA.PeerID

	// узел B: подключается к A (ручной ввод/QR — конфигурация)
	b := startProcNode(t, bin, dir, "b", "-peer", peerA, "-node-key", "proc-b")
	stB := b.waitStatus(30*time.Second, func(s procStatus) bool { return s.JournalLen >= 2 })

	// узел C: подключается к B (каскад: peer exchange + ручной)
	stB2 := b.requireStatus()
	peerB := stB2.Addrs[0] + "/p2p/" + stB2.PeerID
	c := startProcNode(t, bin, dir, "c", "-peer", peerB, "-node-key", "proc-c")
	stC := c.waitStatus(30*time.Second, func(s procStatus) bool { return s.JournalLen >= 2 })

	// сверка корней состояния (INV-3: пересчёт каждым узлом)
	if stA.StateRoot != stB.StateRoot || stB.StateRoot != stC.StateRoot {
		t.Fatalf("корни расходятся: A=%s B=%s C=%s", stA.StateRoot, stB.StateRoot, stC.StateRoot)
	}
	if stB.Peers == 0 || stC.Peers == 0 {
		t.Fatalf("нет подключений: B=%d C=%d", stB.Peers, stC.Peers)
	}
}
