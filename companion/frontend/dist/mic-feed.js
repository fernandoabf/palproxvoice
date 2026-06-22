// AudioWorklet: recebe frames de PCM mono float32 (do mic nativo via WS, postados
// pela main thread) e os entrega como saida de audio. Vira a fonte do mic no
// WebRTC SEM getUserMedia -> sem categoria de comunicacao -> sem degradar o resto.
class MicFeed extends AudioWorkletProcessor {
  constructor() {
    super();
    // ring buffer simples (~1s @48k)
    this.buf = new Float32Array(48000);
    this.r = 0; this.w = 0; this.size = 0;
    // noise gate: silencia o ruido de fundo quando voce NAO fala.
    this.env = 0;        // envelope do nivel (seguidor)
    this.gate = 1;       // ganho atual do gate (comeca aberto)
    this.gateOn = true;  // gate ligado/desligado (config)
    this.open = 0.008;   // acima disso = voz -> abre
    this.close = 0.005;  // abaixo disso = ruido -> fecha (histerese)
    this.port.onmessage = (e) => {
      const d = e.data;
      if (!(d instanceof Float32Array)) { // config { gate, open, close }
        if (d.gate !== undefined) this.gateOn = !!d.gate;
        if (d.open !== undefined) { this.open = d.open; this.close = d.open * 0.6; }
        return;
      }
      for (let i = 0; i < d.length; i++) {     // PCM
        if (this.size < this.buf.length) {
          this.buf[this.w] = d[i];
          this.w = (this.w + 1) % this.buf.length;
          this.size++;
        }
      }
    };
  }
  process(_inputs, outputs) {
    const out = outputs[0][0];
    for (let i = 0; i < out.length; i++) {
      let s = 0;
      if (this.size > 0) { s = this.buf[this.r]; this.r = (this.r + 1) % this.buf.length; this.size--; }
      // envelope: ataque rapido, release lento (nao "pisca" no meio da fala)
      const a = Math.abs(s);
      this.env += (a - this.env) * (a > this.env ? 0.02 : 0.0006);
      // alvo do gate com histerese; abre rapido (~1ms), fecha devagar (~30ms) p/ nao cortar fim de palavra
      const target = this.env > this.open ? 1 : (this.env < this.close ? 0 : this.gate);
      this.gate += (target - this.gate) * (target > this.gate ? 0.03 : 0.0008);
      out[i] = s * this.gate;
    }
    return true;
  }
}
registerProcessor('mic-feed', MicFeed);
