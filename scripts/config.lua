-- KEYS[1]: Chave de bloqueio (ex: blocked:ip:127.0.0.1)
-- KEYS[2]: Chave do contador da janela atual (ex: rate:ip:127.0.0.1:1711000000)
-- ARGV[1]: Limite máximo de requisições por segundo
-- ARGV[2]: Tempo de bloqueio em segundos (ex: 300)
-- ARGV[3]: TTL do contador em segundos (ex: 2)

local block_key = KEYS[1]
local count_key = KEYS[2]
local limit = tonumber(ARGV[1])
local block_ttl = tonumber(ARGV[2])
local count_ttl = tonumber(ARGV[3])

-- 1. Verifica se já está bloqueado
if redis.call("EXISTS", block_key) == 1 then
    return 0 -- Bloqueado
end

-- 2. Incrementa o contador da janela atual
local current = redis.call("INCR", count_key)
if current == 1 then
    redis.call("EXPIRE", count_key, count_ttl)
end

-- 3. Verifica se ultrapassou o limite
if current > limit then
    -- Aplica o bloqueio temporário
    redis.call("SETEX", block_key, block_ttl, "locked")
    return 0 -- Bloqueado agora
end

return 1 -- Permitido