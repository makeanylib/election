-- Returned statuses:
-- 1 - success, heartbeat sent
-- 2 - mismatch

local key = KEYS[1]
local value = ARGV[1]

local stored = server.call('GET', key)
if stored == nil or value ~= stored then
    return 2
end

server.call('DEL', key)
return 1