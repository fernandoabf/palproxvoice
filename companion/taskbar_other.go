//go:build !windows

package main

import "context"

// stubs: tool window + watchdog + ducking + palworld-watch so existem no Windows
// (o app so roda no Windows).
func applyToolWindow()                       {}
func setOverlay(on bool)                      {}
func startOverlayWatch()                      {}
func fixAudioDucking()                        {}
func startPalworldQuitWatch(_ context.Context) {}
