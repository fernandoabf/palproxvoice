#!/bin/sh
# Sobe TUDO local (staging): voice + palworld de teste, no localhost.
#  - voice:    http://localhost:8080  (abre em 2 abas)
#  - palworld: Direct Connect em  localhost:8222   senha teste123
# Sem chaves R2 no .env -> palworld sobe com mundo novo.
# Uso: ./local.sh            (= up -d --build)
#      ./local.sh down       (derruba tudo)
#      ./local.sh logs -f    (acompanha)
C="docker compose -f docker-compose.yml -f docker-compose.palworld-test.yml"
[ $# -eq 0 ] && exec $C up -d --build
exec $C "$@"
