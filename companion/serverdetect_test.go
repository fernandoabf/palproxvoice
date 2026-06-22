package main

import "testing"

func TestLatestServerAddress(t *testing.T) {
	// duas entradas no mesmo IP + uma armadilha: VersionString "0.7.3.904"
	// (4 numeros, casa no regex de IPv4) fica DEPOIS de ServerPort e tem octeto >255.
	blob := []byte("ServerName\x00OurWorld\x00ServerAddress\x00\x0069.62.88.69\x00ServerPort\x00\x00" +
		"ServerName\x00OurWorld\x00ServerAddress\x00\x0069.62.88.69\x00ServerPort\x00\x00" +
		"VersionString\x000.7.3.904\x00")
	if got := latestServerAddress(blob); got != "69.62.88.69" {
		t.Fatalf("got %q, want 69.62.88.69", got)
	}

	// entrada sem endereco (servidor local) nao deve quebrar nem inventar IP
	if got := latestServerAddress([]byte("ServerAddress\x00\x00\x00ServerPort")); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestHostOnly(t *testing.T) {
	for in, want := range map[string]string{
		"69.62.88.69:8211": "69.62.88.69",
		"69.62.88.69":      "69.62.88.69",
		"":                 "",
	} {
		if got := hostOnly(in); got != want {
			t.Fatalf("hostOnly(%q)=%q, want %q", in, got, want)
		}
	}
}
