-- Janela fixa + bloqueio temporário.
-- KEYS[1]: chave de bloqueio (ex: blocked:ip:127.0.0.1)
-- KEYS[2]: contador da janela de 1s atual (ex: rate:ip:127.0.0.1:1711000000)
-- ARGV[1]: limite máximo de requisições por janela
-- ARGV[2]: tempo de bloqueio em segundos (ex: 300)
-- ARGV[3]: TTL do contador em segundos
-- Retorna 1 se a requisição é permitida, 0 se está (ou acabou de ficar) bloqueada.

local block_key = KEYS[1]
local count_key = KEYS[2]
local limit = tonumber(ARGV[1])
local block_ttl = tonumber(ARGV[2])
local count_ttl = tonumber(ARGV[3])

-- 1. Já está bloqueado?
if redis.call("EXISTS", block_key) == 1 then
    return 0
end

-- 2. Incrementa o contador da janela atual
local current = redis.call("INCR", count_key)
if current == 1 then
    redis.call("EXPIRE", count_key, count_ttl)
end

-- 3. Ultrapassou o limite? Bloqueia pelo tempo configurado
if current > limit then
    redis.call("SETEX", block_key, block_ttl, "locked")
    return 0
end

return 1
