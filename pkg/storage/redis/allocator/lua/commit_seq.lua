--[[
提交序列号脚本
用于提交扩容后的序列号边界，释放锁

参数说明：
  KEYS[1]  - Redis key，格式: goim:msg:seq:{conversationID}
  ARGV[1]  - 锁值（lockValue），用于验证锁归属，空字符串表示直接设置（Set方法调用）
  ARGV[2]  - 数据过期时间（dataSecond），单位秒
  ARGV[3]  - 当前序列号（curr_seq）
  ARGV[4]  - 新的预分配边界（last_seq）
  ARGV[5]  - 分配时间戳（mallocTime），毫秒

返回值说明：
  1: key 不存在，首次创建
  2: 锁不匹配，提交失败（锁被其他节点持有或已过期）
  0: 提交成功，锁已释放
]]

local key = KEYS[1]            -- Redis key
local lockValue = ARGV[1]      -- 锁值，用于验证锁归属
local dataSecond = ARGV[2]     -- 数据过期时间（秒）
local curr_seq = tonumber(ARGV[3])  -- 当前序列号
local last_seq = tonumber(ARGV[4])  -- 新的预分配边界
local mallocTime = ARGV[5]     -- 分配时间戳（毫秒）

-- 1. key 不存在，首次创建
if redis.call("EXISTS", key) == 0 then
    redis.call("HSET", key, "CURR", curr_seq, "LAST", last_seq, "TIME", mallocTime)
    redis.call("EXPIRE", key, dataSecond)
    return 1
end

-- 2. lockValue 为空字符串，表示直接设置（Set方法），跳过锁验证
if lockValue == "" then
    redis.call("HSET", key, "CURR", curr_seq, "LAST", last_seq, "TIME", mallocTime)
    redis.call("EXPIRE", key, dataSecond)
    return 0
end

-- 3. 锁不匹配，提交失败
if redis.call("HGET", key, "LOCK") ~= lockValue then
    return 2
end

-- 4. 验证通过，释放锁并更新序列号状态
--    首次同步场景：key 存在但 CURR 不存在（allocate_seq 只设置了 LOCK），使用传入的 curr_seq
--    扩容场景：key 存在且 CURR 存在，保留现有 CURR 值
local existing_curr = redis.call("HGET", key, "CURR")
if existing_curr then
    curr_seq = tonumber(existing_curr)
end

redis.call("HDEL", key, "LOCK")
redis.call("HSET", key, "CURR", curr_seq, "LAST", last_seq, "TIME", mallocTime)
redis.call("EXPIRE", key, dataSecond)

return 0