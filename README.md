# Rate Limiter em Go

Middleware de limitação de requisições escrito em Go. Ele controla o tráfego por **endereço IP** e por **token de acesso** (header `API_KEY`), armazena os contadores no **Redis** e bloqueia temporariamente quem ultrapassa o limite.

- **Limite por IP**: aplicado a todas as requisições que não trazem um token válido.
- **Limite por token**: aplicado a requisições com um `API_KEY` válido. Tem **precedência** sobre o limite por IP.
- **Janela de 1 segundo** e **bloqueio** configurável (padrão: 300 segundos) ao exceder o limite.
- Resposta `429 Too Many Requests` quando bloqueado.

---

## Sumário

1. [Requisitos](#requisitos)
2. [Como executar](#como-executar)
3. [Configuração](#configuração)
4. [Como o limiter funciona](#como-o-limiter-funciona)
5. [Alterando as estratégias](#alterando-as-estratégias)
6. [Testes automatizados](#testes-automatizados)
7. [Comandos do Makefile](#comandos-do-makefile)
8. [Estrutura do projeto](#estrutura-do-projeto)

---

## Requisitos

- [Docker](https://docs.docker.com/get-docker/) e Docker Compose (recomendado)
- Para rodar sem Docker: Go 1.27+ e Make
- Redis 7 (o `docker-compose.yaml` já sobe um)

---

## Como executar

### 1. Crie o arquivo `.env`

```bash
cp .env.example .env
```

O `.env` não é versionado. Ajuste os valores conforme necessário (veja [Configuração](#configuração)).

### 2. Subir tudo com Docker

```bash
make docker-up
# ou: docker compose up -d
```

Isso sobe a aplicação (porta `SERVER_PORT`, padrão `8080`) e o Redis. A aplicação só inicia depois que o Redis passa no healthcheck.

Para parar:

```bash
make docker-down
```

### 3. Rodar localmente (sem subir a aplicação no Docker)

Suba só o Redis e rode o servidor no host:

```bash
make docker-redis
REDIS_ADDR=localhost:6379 make run
```

> **Atenção:** o `.env.example` define `REDIS_ADDR=redis:6379`, que só resolve dentro da rede do Docker. Ao rodar no host, sobrescreva a variável como no exemplo acima (variáveis de ambiente têm prioridade sobre o `.env`) ou altere o `.env`.

### 4. Testando

Sem token (limite por IP, padrão 10 req/s):

```bash
curl -i http://localhost:8080/
```

Com token válido (limite por token, padrão 100 req/s):

```bash
curl -i -H "API_KEY: token_de_acesso" http://localhost:8080/
```

Resposta normal (`200 OK`):

```json
{"status":"ok"}
```

Ao exceder o limite (`429 Too Many Requests`):

```
you have reached the maximum number of requests or actions allowed within a certain time frame
```

Para gerar carga, você pode usar qualquer ferramenta HTTP (por exemplo, `hey`, `ab` ou um loop com `curl`). O alvo `stress-test` do Makefile espera uma imagem `stress-test` local, que não faz parte deste repositório. Faz parte do repositorio do desafio anteerior [stress-test](https://github.com/Hiaguedes/mba-go-final-challenge-stress-test) onde adicionei uma flag a mais chamada `-H` ou `--header` onde se passa o header como no curl `"API_KEY: token_de_acesso"` fique a vontade pra usar o stress-test do repositorio ou de sua preferencia

---

## Configuração

Todas as configurações são lidas do arquivo `.env` (na raiz do projeto, de onde o servidor é executado) ou de variáveis de ambiente. **Variáveis de ambiente têm prioridade sobre o `.env`.**

| Variável | Padrão (`.env.example`) | Descrição |
|---|---|---|
| `SERVER_PORT` | `8080` | Porta HTTP da aplicação. |
| `REDIS_ADDR` | `redis:6379` | Endereço do Redis (`host:porta`). Use `localhost:6379` ao rodar no host. |
| `IP_REQS_PER_SEC_LIMITTER` | `10` | Máximo de requisições por segundo por IP (sem token válido). **Obrigatório > 0.** |
| `API_TOKEN` | `token_de_acesso` | Único token aceito no header `API_KEY`. Vazio desativa o limite por token. |
| `API_TOKEN_REQS_PER_SEC_LIMITTER` | `100` | Máximo de requisições por segundo para o token válido. **Obrigatório > 0.** |
| `REQS_BLOCK_TIME_SEC` | `300` | Tempo de bloqueio, em segundos, após exceder o limite. **Obrigatório > 0.** |
| `TRUST_PROXY_HEADERS` | `false` | Se `true`, usa `X-Forwarded-For` / `X-Real-IP` para identificar o IP do cliente. |

### Pontos de atenção

- **Valores inválidos impedem a inicialização.** `IP_REQS_PER_SEC_LIMITTER`, `API_TOKEN_REQS_PER_SEC_LIMITTER` e `REQS_BLOCK_TIME_SEC` precisam ser inteiros positivos; caso contrário, o servidor encerra com erro.
- **Mudanças exigem reinício.** Os valores são lidos uma vez na inicialização. Após alterar o `.env`, reinicie a aplicação (`make docker-down && make docker-up`).
- **`TRUST_PROXY_HEADERS=true` somente atrás de um proxy confiável** que sobrescreva `X-Forwarded-For`. Caso contrário, qualquer cliente pode forjar o próprio IP e escapar do limite.
- **Token nunca é gravado em texto puro.** O Redis guarda apenas o hash SHA-256 do token (`token:<hash>`).

### Como alterar um limite

Exemplo: aumentar o limite por IP para 50 req/s e o bloqueio para 60 segundos. No `.env`:

```env
IP_REQS_PER_SEC_LIMITTER=50
REQS_BLOCK_TIME_SEC=60
```

Depois reinicie a aplicação.

---

## Como o limiter funciona

### Precedência: Token > IP

A decisão é tomada em `internal/controller/rate_limiter_middleware.go`, no método `identify`:

1. Se o header `API_KEY` contém o token configurado em `API_TOKEN` (comparação em tempo constante), a requisição é identificada como `token:<sha256>` e usa o limite `API_TOKEN_REQS_PER_SEC_LIMITTER`.
2. Caso contrário (header ausente, vazio ou com token errado), a requisição é identificada como `ip:<endereço>` e usa o limite `IP_REQS_PER_SEC_LIMITTER`.

Consequências da precedência:

- Um token válido **substitui** o limite de IP: a requisição não é contada na cota do IP.
- Um IP que já está bloqueado por excesso **ainda consegue** atender requisições com token válido, porque elas usam outra identidade.
- Um token válido segue o **próprio** limite. Se ele for excedido, a requisição recebe `429` mesmo vindo de um IP sem bloqueio.
- Um token inválido não concede privilégio: cai no limite por IP.

### Janela e bloqueio (Redis)

A lógica fica em `internal/infra/redis/limiter.lua` e roda **atomicamente** no Redis (via `EVALSHA`/`Script`), então não há condição de corrida entre instâncias da aplicação.

Para cada requisição:

1. Se `blocked:<identidade>` existe, a requisição é negada.
2. Caso contrário, incrementa `rate:<identidade>:<segundo-atual>` (contador da janela de 1 segundo, com TTL de 2 segundos).
3. Se o contador passou do limite, grava `blocked:<identidade>` com TTL igual a `REQS_BLOCK_TIME_SEC` e nega a requisição.
4. Caso contrário, permite.

Se o Redis estiver indisponível, a aplicação responde **`500 internal server error`**, e não `429`. Ou seja, o limiter falha de forma fechada e registra o erro no log.

### Respostas

| Situação | Status | Corpo |
|---|---|---|
| Permitida | `200` | `{"status":"ok"}` |
| Limite excedido ou bloqueado | `429` | Mensagem de limite excedido (texto) |
| Falha no Redis | `500` | `internal server error` |

Cada requisição gera uma linha de log com método, caminho, status, duração, IP do cliente e se o header `API_KEY` foi enviado (sem o valor do token).

---

## Alterando as estratégias

A arquitetura separa três conceitos:

| Conceito | Onde fica | Papel |
|---|---|---|
| Porta de decisão | `internal/usecase/limitter.go` (`AllowFunc[T]`) | Assinatura: decide se uma identidade pode seguir. |
| Interface de banco | `internal/infra/database/strategy.go` (`Database[T]`) | Contrato que cada backend implementa: `LoadDatabase` e `Allow`. |
| Implementação | `internal/infra/redis/` | Implementação atual, usando Redis + script Lua. |

O middleware recebe apenas uma função `Allow` e um cliente genérico `T`; ele não conhece o Redis.

### Opção 1: mudar o algoritmo de limitação (Redis)

Para trocar a janela fixa por outra estratégia (janela deslizante, token bucket, etc.), edite `internal/infra/redis/limiter.lua`. O contrato precisa ser mantido:

- **Entradas:** `KEYS[1]` = chave de bloqueio, `KEYS[2]` = chave do contador; `ARGV[1]` = limite, `ARGV[2]` = tempo de bloqueio (s), `ARGV[3]` = TTL do contador (s).
- **Saída:** `1` para permitir, `0` para negar.

O script é embutido no binário com `go:embed`, então basta recompilar (`make run` ou `docker compose up -d --build`). Se a nova estratégia precisar de outra chave ou TTL, ajuste também `internal/infra/redis/main.go` (função `Allow`, incluindo a constante `counterTTLSeconds`).

### Opção 2: trocar o backend (ex.: outro armazenamento)

1. **Crie um pacote** em `internal/infra/<nome>` com um tipo que implemente `database.Database[T]`:

   ```go
   type Minha struct{ /* config */ }

   // Cria/conecta o cliente. T é o tipo do cliente do seu backend.
   func (m Minha) LoadDatabase(ctx context.Context) (*MeuCliente, error) { /* ... */ }

   // Mesmo contrato do Redis: true = permitir, false = negar.
   func (m Minha) Allow(ctx context.Context, identifier string, limit, blockSeconds int, client *MeuCliente) (bool, error) {
       /* ... */
   }
   ```

2. **Troque a estratégia** em `cmd/server/main.go`. Hoje o código fixa o tipo do cliente em `*redis.Client`:

   ```go
   databaseStrategy := database.NewDatabaseStrategy[*redis.Client]()
   databaseStrategy.SetStrategy(redis_db.NewRedis(cfg.RedisAddr))
   ```

   Para outro backend, use o tipo do seu cliente em `NewDatabaseStrategy[...]` e chame `SetStrategy` com a sua implementação. O restante do fluxo (`NewRateLimitMiddleware(client, configMiddleware, databaseStrategy.Allow)`) não muda.

3. **Se o backend precisar de novas variáveis**, adicione-as em `config/main.go` (struct `Config`, `viper.BindEnv` e `LoadConfig`) e no `.env.example`.

> Os testes de integração de `internal/controller` usam `*redis.Client` diretamente. Se trocar o backend, ajuste-os ou crie testes equivalentes para a nova implementação.

### Opção 3: mudar a identificação (IP, token, outro header)

A regra de precedência está em `identify` em `internal/controller/rate_limiter_middleware.go`. Para, por exemplo, limitar por um header diferente ou por rota, altere essa função. Mantenha a ordem: a identidade mais específica deve ter precedência.

---

## Testes automatizados

Os testes cobrem a eficácia do limiter e a lógica de precedência **Token > IP**.

| Arquivo | Tipo | O que demonstra |
|---|---|---|
| `internal/controller/rate_limiter_middleware_test.go` | Unitário (limiter fake em memória) | Limite por IP, precedência do token, token válido não afetado por IP bloqueado, fallback para IP com token ausente/inválido/com espaço/em maiúsculas, token nunca aparece em texto puro, `X-Forwarded-For` ignorado sem proxy confiável, erro do backend gera `500`. |
| `internal/controller/rate_limiter_redis_test.go` | Integração (Redis real) | Ponta a ponta: após estourar o limite por IP, token válido ainda passa; token também tem limite próprio e gera `429` ao exceder. |
| `internal/infra/redis/limiter_test.go` | Integração (Redis real) | Nunca bloqueia dentro do limite; bloqueia acima do limite e libera após o tempo de bloqueio; a janela de 1 segundo reinicia. |

Os testes de integração são **pulados** quando `REDIS_ADDR` não está definido, então `make test` funciona sem Redis.

Executar:

```bash
make test               # unitários (integração pulada sem REDIS_ADDR)
make test-integration   # sobe o Redis via Docker e roda tudo
```

Os testes de integração levam alguns segundos porque esperam o bloqueio e a troca de janela de 1 segundo.

---

## Comandos do Makefile

| Comando | Ação |
|---|---|
| `make run` | Roda o servidor localmente (`go run cmd/server/main.go`). |
| `make test` | Roda todos os testes; os de integração são pulados sem `REDIS_ADDR`. |
| `make test-integration` | Sobe o Redis via Docker e roda todos os testes com `REDIS_ADDR=localhost:6379`. |
| `make docker-redis` | Sobe somente o serviço Redis. |
| `make docker-up` | Sobe aplicação e Redis (`docker compose up -d`). |
| `make docker-down` | Derruba os containers (`docker compose down`). |
| `make stress-test` | Executa uma imagem `stress-test` local contra `host.docker.internal:8080`. |

---

## Estrutura do projeto

```
cmd/server/main.go                      # Inicialização: config, Redis, middleware, servidor HTTP
config/main.go                          # Leitura das variáveis (.env e ambiente) com viper
internal/controller/
  rate_limiter_middleware.go            # Precedência Token > IP e resposta 429/500
  rate_limiter_middleware_test.go       # Testes unitários do middleware
  rate_limiter_redis_test.go            # Teste de integração ponta a ponta com Redis
  request_logger_middleware.go          # Log de cada requisição (inclui 429)
  dumb_server.go                        # Handler de exemplo: responde {"status":"ok"}
internal/usecase/limitter.go            # Porta AllowFunc usada pelo middleware
internal/infra/database/strategy.go     # Interface Database[T] e seleção de estratégia
internal/infra/redis/
  main.go                               # Implementação Redis (Allow, LoadDatabase)
  limiter.lua                           # Script atômico: janela de 1s + bloqueio
  limiter_test.go                       # Testes de integração do algoritmo
internal/infra/webserver/main.go        # Servidor HTTP
internal/exceptions/                    # Erros da aplicação
docker-compose.yaml, Dockerfile         # Execução em containers
Makefile                                # Comandos de build, teste e execução
```

---

## Limitações conhecidas

- **Janela fixa:** o contador é reiniciado a cada segundo. Uma rajada que atravessa a fronteira de dois segundos pode passar com até o dobro do limite nesse intervalo. Para evitar isso, troque para janela deslizante no `limiter.lua` (veja [Alterando as estratégias](#alterando-as-estratégias)).
- **Confiança em proxy:** veja a observação sobre `TRUST_PROXY_HEADERS` em [Configuração](#configuração).
