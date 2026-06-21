// PalProxVoice bridge — le o arquivo de posicao do mod e serve pro navegador (companion).
// Roda no PC do jogo (Windows). O navegador faz GET http://127.0.0.1:47475/pos e usa
// a posicao real do jogo no lugar do WASD.
package main

import (
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	tmp := os.Getenv("TEMP")
	if tmp == "" {
		tmp = os.TempDir()
	}
	posFile := filepath.Join(tmp, "palproxvoice_pos.txt")

	http.HandleFunc("/pos", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // pagina vem de outra origem (server de voz)
		b, err := os.ReadFile(posFile)
		if err != nil {
			w.WriteHeader(http.StatusNoContent) // ainda nao entrou no mundo
			return
		}
		_, _ = w.Write(b)
	})

	addr := "127.0.0.1:47475"
	println("PalProxVoice bridge:  http://" + addr + "/pos  <-  " + posFile)
	println("deixa rodando e abre a pagina do companion. Ctrl+C pra sair.")
	if err := http.ListenAndServe(addr, nil); err != nil {
		println("erro:", err.Error())
	}
}
