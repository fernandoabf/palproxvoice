// AudioWorklet: recebe frames de PCM mono float32 (do mic nativo via WS, postados
// pela main thread) e os entrega como saida de audio. Vira a fonte do mic no
// WebRTC SEM getUserMedia -> sem categoria de comunicacao -> sem degradar o resto.
class MicFeed extends AudioWorkletProcessor {
  constructor() {
    super();
    // ring buffer simples (~1s @48k)
    this.buf = new Float32Array(48000);
    this.r = 0; this.w = 0; this.size = 0;
    this.port.onmessage = (e) => {
      const f = e.data; // Float32Array
      for (let i = 0; i < f.length; i++) {
        if (this.size < this.buf.length) {
          this.buf[this.w] = f[i];
          this.w = (this.w + 1) % this.buf.length;
          this.size++;
        }
      }
    };
  }
  process(_inputs, outputs) {
    const out = outputs[0][0];
    for (let i = 0; i < out.length; i++) {
      if (this.size > 0) {
        out[i] = this.buf[this.r];
        this.r = (this.r + 1) % this.buf.length;
        this.size--;
      } else {
        out[i] = 0; // underrun -> silencio
      }
    }
    return true;
  }
}
registerProcessor('mic-feed', MicFeed);
