-- Returned statuses:
-- 1 - success, heartbeat sent
-- 2 - failure, lock not found or token mismatch

local key__lock = KEYS[1]
local arg__lock_token = ARGV[1]
local arg__heartbeat_timeout = tonumber(ARGV[2])

local token = server.call('GET', key__lock)
if token == nil or arg__lock_token ~= token  then
    return 2
end

server.call('EXPIRE', key__lock, arg__heartbeat_timeout)
return 1