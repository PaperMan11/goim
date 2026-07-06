--[[
分配序列号脚本
基于预分配池模式，减少 Redis 写次数，提高高并发性能

参数说明：
  KEYS[1]  - Redis key，格式: goim:msg:seq:{conversationID}
  ARGV[1]  - 需要分配的序列号数量（size），0 表示仅查询
  ARGV[2]  - 锁过期时间（lockSecond），单位秒
  ARGV[3]  - 数据过期时间（dataSecond），单位秒
  ARGV[4]  - 分配时间戳（mallocTime），毫秒

返回值说明（数组）：
  返回码 1: 首次创建，key 不存在，需要从数据库同步初始值
           [1, lockValue, mallocTime]
  返回码 2: 已被其他节点锁定，稍后重试
           [2]
  返回码 3: 预分配池耗尽，需要扩容（从数据库获取新的 LAST）
           [3, curr_seq, last_seq, lockValue, mallocTime]
  返回码 0: 正常分配成功
           [0, curr_seq, last_seq, mallocTime]
           或仅查询时 [0, curr_seq, last_seq, setTime]

数据结构（Hash）：
  CURR   - 当前已分配到的序列号
  LAST   - 预分配池的最大边界
  TIME   - 最后一次分配时间
  LOCK   - 分布式锁（随机值，防止并发冲突）
]]

local key = KEYS[1]           -- Redis key
local size = tonumber(ARGV[1]) -- 要分配的序列号数量，0 表示仅查询
local lockSecond = ARGV[2]    -- 锁过期时间（秒）
local dataSecond = ARGV[3]    -- 数据过期时间（秒）
local mallocTime = ARGV[4]    -- 分配时间戳（毫秒）
local result = {}             -- 返回结果数组

-- 1. key 不存在，首次创建
if redis.call("EXISTS", key) == 0 then
    -- 生成随机锁值，用于分布式锁验证
    local lockValue = math.random(0, 999999999)
    -- 设置锁
    redis.call("HSET", key, "LOCK", lockValue)
    -- 设置锁过期时间
    redis.call("EXPIRE", key, lockSecond)
    -- 返回：首次创建
    table.insert(result, 1)
    table.insert(result, lockValue)
    table.insert(result, mallocTime)
    return result
end

-- 2. 已被其他节点锁定
if redis.call("HEXISTS", key, "LOCK") == 1 then
    -- 返回：被锁定
    table.insert(result, 2)
    return result
end

-- 3. 获取当前序列号状态
local curr_seq = tonumber(redis.call("HGET", key, "CURR"))
local last_seq = tonumber(redis.call("HGET", key, "LAST"))

-- 4. size == 0 表示仅查询当前状态
if size == 0 then
    -- 更新数据过期时间
    redis.call("EXPIRE", key, dataSecond)
    -- 返回：查询成功
    table.insert(result, 0)
    table.insert(result, curr_seq)
    table.insert(result, last_seq)
    -- 获取设置时间
    local setTime = redis.call("HGET", key, "TIME")
    if setTime then
        table.insert(result, setTime)
    else
        table.insert(result, 0)
    end
    return result
end

-- 5. 预分配池耗尽，需要扩容
local max_seq = curr_seq + size
if max_seq > last_seq then
    -- 生成随机锁值
    local lockValue = math.random(0, 999999999)
    -- 设置锁，不修改 CURR，等待提交新边界
    redis.call("HSET", key, "LOCK", lockValue)
    -- 记录分配时间
    redis.call("HSET", key, "TIME", mallocTime)
    -- 设置锁过期时间
    redis.call("EXPIRE", key, lockSecond)
    -- 返回：需要扩容
    table.insert(result, 3)
    table.insert(result, curr_seq)
    table.insert(result, last_seq)
    table.insert(result, lockValue)
    table.insert(result, mallocTime)
    return result
end

-- 6. 正常分配，在预分配池内
redis.call("HSET", key, "CURR", max_seq)
redis.call("HSET", key, "TIME", ARGV[4])
redis.call("EXPIRE", key, dataSecond)
-- 返回：分配成功
table.insert(result, 0)
table.insert(result, curr_seq)
table.insert(result, last_seq)
table.insert(result, mallocTime)
return result