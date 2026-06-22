//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// fixAudioDucking desliga o "ducking" de comunicacao do Windows (abaixar todos
// os outros sons quando o mic ativa). Grava no HKCU do USUARIO — o instalador
// roda como admin e pode gravar no hive errado, por isso o .exe faz tambem.
// Idempotente (roda a cada abertura). Efeito pega no proximo logon/reinicio, ou
// na hora em Som > Comunicacoes > "Nao fazer nada".
//   3 = nao fazer nada · 2 = -50% · 1 = -80% · 0 = mutar
func fixAudioDucking() {
	cmd := exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Multimedia\Audio`,
		"/v", "UserDuckingPreference", "/t", "REG_DWORD", "/d", "3", "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}
