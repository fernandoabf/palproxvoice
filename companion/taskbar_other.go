//go:build !windows

package main

// stubs: tool window + watchdog so existem no Windows (o app so roda no Windows).
func applyToolWindow()   {}
func setOverlay(on bool) {}
func startOverlayWatch() {}
