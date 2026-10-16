#!/usr/bin/env node
/**
 * ============================================================================
 * Apifox / Node.js 模拟 msggateway WebSocket 客户端
 * ============================================================================
 *
 * 用途：模拟客户端连接 im-msggateway，覆盖各业务指令（1001/1002/1003/1005/1006/1007/2003/2004/2005）的
 *      报文构造、发送与响应解析。报文格式与 im-msggateway 的 JSON 编码器完全一致：
 *
 *          Request  = {"reqIdentifier":int,"token":string,"senderID":string,"operationID":string,
 *                      "msgIncr":string,"data":"<base64(protobuf 二进制)>"}
 *
 *      注意：Go 的 encoding/json 会把 []byte 编成 base64 字符串，而 data 的内容是 protobuf 二进制，
 *      因此 data 必须是「protobuf 序列化后再 base64」的结果，不能直接填 JSON 文本。
 *      本脚本内置极简 protobuf 编码器，用来现场生成 data，无需安装任何依赖。
 *
 * ----------------------------------------------------------------------------
 * 一、在 Apifox 中的使用步骤
 * ----------------------------------------------------------------------------
 * 1) 取 token（im-api HTTP 接口）
 *      方法/地址：POST http://127.0.0.1:18880/api/v1/auth/get_user_token
 *      Body(JSON)：{"platformID":1,"userID":"test_user_001"}
 *      返回：{"code":0,"msg":"success","data":{"token":"<TOKEN>","expireTimeSeconds":...}}
 *      把 <TOKEN> 存为 Apifox 环境变量，例如 {{token}}
 *
 * 2) 新建 WebSocket 接口
 *      地址：ws://127.0.0.1:50001/ws?userID=test_user_001&token={{token}}&platformID=1
 *             &deviceID=apifox_device_001&sendResponse=true&isBackground=false
 *             &compression=none&sdkType=normal&sdkVersion=1.0.0&operationID=apifox-op-0001
 *      （query 参数说明见下表「连接参数」）
 *
 * 3) 在 WebSocket 接口的「消息」列表里逐条新增「文本消息」，把本脚本 --print 输出的
 *      JSON 报文原样粘贴进去即可（data 已是可直接使用的 base64）。
 *
 * 4) 连接后按 1001 → 1003 → 1002/1005/1007 的顺序发送，观察返回。
 *      响应同样是 JSON：{"reqIdentifier":..,"msgIncr":..,"operationID":..,"errCode":..,"errMsg":..,"data":..}
 *      errCode=0 表示成功；data 为 base64(protobuf)，把 base64 贴回本脚本 --decode 即可解析。
 *
 * 连接参数（与服务端 handshakeInfo 的 form tag 一一对应）：
 *   userID       必填  登录用户ID，必须与 token 解析出的 userID 一致
 *   token        必填  登录令牌，EnableAuth=true 时校验
 *   platformID   必填  平台ID（1-iOS 2-Android 3-Windows 4-OSX 5-Web 7-Linux 8-APad 9-IPad 10-Admin 11-HarmonyOS 12-Bot）
 *   deviceID     选填  设备标识，用于在线状态
 *   sendResponse 选填  握手/连接级错误是否通过连接回写 JSON 错误体
 *   isBackground 选填  是否后台连接
 *   operationID  选填  链路追踪ID
 *   compression  选填  none(默认)/gzip/zlib，须与发送报文时的压缩方式一致
 *   sdkType      选填  normal(默认，JSON 编码) 等，决定 req/resp 编解码器
 *   sdkVersion   选填  版本号，仅记录
 *
 * ----------------------------------------------------------------------------
 * 二、业务指令与参数
 * ----------------------------------------------------------------------------
 *   1001 WSGetNewestSeq              参数 GetMaxSeqReq{userID}
 *   1002 WSPullMsgBySeqList          参数 PullMessageBySeqsReq{userID, seqRanges[{conversationID,begin,end,num}]}
 *   1003 WSSendMsg                   参数 MsgData{sendID,recvID/groupID,sessionType,contentType,content,...}
 *   1004 WSSendSignalMsg             参数 MsgData{contentType=202}
 *   1005 WSPullMsg                   参数 GetSeqMessageReq{userID, conversations[{conversationID,seqs[]}]}
 *   1006 WSGetConvMaxReadSeq         参数 GetConversationsHasReadAndMaxSeqReq{userID, conversationIDs[], returnPinned}
 *   1007 WsPullConvLastMessage       参数 GetLastMessageReq{userID, conversationIDs[]}
 *   2003 WSLogoutMsg                 ⚠ 服务端当前返回 "not support" 并随后关闭连接
 *   2004 WSSetBackgroundStatus       ⚠ 服务端当前返回 "not support"
 *   2005 WSSubUserOnlineStatus       参数 SubUserOnlineStatus{subscribeUserID[], unsubscribeUserID[]}
 *   2002 WSKickOnlineMsg / 2001 WSPushMsg / 3001 WSDataError 为服务端→客户端方向，客户端不应主动发送
 *
 * 会话ID规则（拉取类指令必须使用）：
 *   单聊  si_<较小userID>_<较大userID>      例如 si_test_user_001_test_user_002
 *   群聊  g_<groupID>（写扩散）/ sg_<groupID>（读扩散）  例如 sg_group_001
 *
 * 常用 sessionType：1 单聊 / 2 写扩散群聊 / 3 读扩散群聊 / 4 通知会话
 * 常用 contentType：101 文本 / 102 图片 / 103 语音 / 104 视频 / 105 文件 / 106 @消息 /
 *                   114 引用 / 111 撤回 / 113 正在输入 / 202 信令
 *
 * ----------------------------------------------------------------------------
 * 三、脚本运行方式
 * ----------------------------------------------------------------------------
 *   node apifox_ws_client.js --print                    仅打印各指令在 Apifox 中可直接粘贴的报文
 *   node apifox_ws_client.js --dump=payloads.jsonl      把各指令的 data(base64) 写成 JSONL，便于二次加工
 *   node apifox_ws_client.js --decode=<tag>:<base64>    解析服务端返回的 data（base64→可读 JSON）
 *       tag 可省略，省略时按 MsgData 解析；可选 1001/1002/1003/1005/1006/1007
 *       例：node apifox_ws_client.js --decode="1003:$(base64)"
 *   node apifox_ws_client.js --token=xxx                连接并依次执行全部指令，打印每一步响应
 *       可选：--url / --user / --platform / --recv / --group / --delay / --only=1001,1003
 *
 * 依赖：Node.js >= 22（使用内置 WebSocket，无需 npm install）
 */

'use strict';

const fs = require('fs');

// ============================== 极简 protobuf 编码 ==============================

function varint(num) {
  let n = BigInt(num);
  const out = [];
  do {
    let b = Number(n & 0x7fn);
    n >>= 7n;
    if (n > 0n) b |= 0x80;
    out.push(b);
  } while (n > 0n);
  return Buffer.from(out);
}

function tag(field, wire) {
  return varint((BigInt(field) << 3n) | BigInt(wire));
}

/** 字符串/bytes 字段（wire type 2） */
function fStr(field, v) {
  if (v === undefined || v === null || v === '') return Buffer.alloc(0);
  const b = Buffer.from(String(v), 'utf8');
  return Buffer.concat([tag(field, 2), varint(b.length), b]);
}

/** 整型/枚举字段（wire type 0），proto3 中 0 值不序列化 */
function fInt(field, v) {
  if (v === undefined || v === null || Number(v) === 0) return Buffer.alloc(0);
  return Buffer.concat([tag(field, 0), varint(v)]);
}

function fBool(field, v) {
  return v ? Buffer.concat([tag(field, 0), varint(1)]) : Buffer.alloc(0);
}

/** 嵌套消息字段（wire type 2） */
function fMsg(field, sub) {
  if (!sub || sub.length === 0) return Buffer.alloc(0);
  return Buffer.concat([tag(field, 2), varint(sub.length), sub]);
}

/** repeated string */
function fStrList(field, arr) {
  if (!Array.isArray(arr) || arr.length === 0) return Buffer.alloc(0);
  return Buffer.concat(arr.map((v) => fStr(field, v)));
}

/** map<string,bool>：entry = {key=1 string, value=2 bool} */
function fMapBool(field, obj) {
  if (!obj || typeof obj !== 'object') return Buffer.alloc(0);
  return Buffer.concat(
    Object.entries(obj).map(([k, v]) => {
      const entry = Buffer.concat([fStr(1, k), fBool(2, v)]);
      return Buffer.concat([tag(field, 2), varint(entry.length), entry]);
    })
  );
}

function concat(parts) {
  return Buffer.concat(parts.filter((p) => p && p.length > 0));
}

// ============================== 业务报文构造 ==============================

/** openim.sdkws.OfflinePushInfo */
function offlinePushInfo(o = {}) {
  return concat([
    fStr(1, o.title),
    fStr(2, o.desc),
    fStr(3, o.ex),
    fStr(4, o.iOSPushSound),
    fBool(5, o.iOSBadgeCount),
    fStr(6, o.signalInfo),
  ]);
}

/** openim.sdkws.MsgData */
function msgData(m) {
  return concat([
    fStr(1, m.sendID),
    fStr(2, m.recvID),
    fStr(3, m.groupID),
    fStr(4, m.clientMsgID),
    fStr(5, m.serverMsgID),
    fInt(6, m.senderPlatformID),
    fStr(7, m.senderNickname),
    fStr(8, m.senderFaceURL),
    fInt(9, m.sessionType),
    fInt(10, m.msgFrom),
    fInt(11, m.contentType),
    fStr(12, m.content),
    fInt(14, m.seq),
    fInt(15, m.sendTime),
    fInt(16, m.createTime),
    fInt(17, m.status),
    fBool(18, m.isRead),
    fMapBool(19, m.options),
    fMsg(20, m.offlinePushInfo ? offlinePushInfo(m.offlinePushInfo) : null),
    fStrList(21, m.atUserIDList),
    fStr(22, m.attachedInfo),
    fStr(23, m.ex),
  ]);
}

const DEFAULT_OPTIONS = {
  history: true,
  persistent: true,
  offlinePush: true,
  unreadCount: true,
  conversationUpdate: true,
  senderSync: true,
  isSendMsg: true,
};

/** 会话ID：单聊按字典序拼接，群聊按扩散类型加前缀 */
function singleConvID(a, b) {
  return 'si_' + (a < b ? a + '_' + b : b + '_' + a);
}
function groupConvID(groupID, sessionType) {
  return (Number(sessionType) === 3 ? 'sg_' : 'g_') + groupID;
}

function buildMsgData(ctx) {
  const now = Date.now();
  return {
    sendID: ctx.user,
    recvID: ctx.recv,
    clientMsgID: 'client-msg-' + String(now).slice(-6),
    senderPlatformID: Number(ctx.platform),
    senderNickname: ctx.nickname,
    senderFaceURL: 'https://cdn.example.com/avatar/001.png',
    sessionType: 1,
    msgFrom: 100,
    contentType: 101,
    content: JSON.stringify({ text: 'hello from apifox', atUsers: [] }),
    createTime: now,
    options: { ...DEFAULT_OPTIONS },
    offlinePushInfo: { title: ctx.nickname, desc: 'hello from apifox', iOSPushSound: 'default', iOSBadgeCount: true },
  };
}

function buildGroupAtMsgData(ctx) {
  const m = buildMsgData(ctx);
  m.recvID = '';
  m.groupID = ctx.group;
  m.sessionType = 3; // 读扩散群聊
  m.contentType = 106; // @消息
  m.content = JSON.stringify({
    text: '@' + ctx.recv + ' 请查收',
    atUsers: [{ atUserID: ctx.recv, subTop: ctx.recv }],
  });
  m.atUserIDList = [ctx.recv];
  m.offlinePushInfo = { title: '群消息', desc: m.content, iOSPushSound: 'default', iOSBadgeCount: true };
  return m;
}

/** 指令 → { 说明, 是否需要 data, data 构造, 响应解析器 } */
function buildCases(ctx) {
  const singleConv = singleConvID(ctx.user, ctx.recv);
  const groupConv = groupConvID(ctx.group, 3);

  return [
    {
      id: 1001,
      name: 'WSGetNewestSeq 获取各会话最大/最小seq',
      build: () => concat([fStr(1, ctx.user)]),
      decode: decodeMaxSeqResp,
    },
    {
      id: 1002,
      name: 'WSPullMsgBySeqList 按seq范围拉取历史消息',
      build: () =>
        concat([
          fStr(1, ctx.user),
          fMsg(
            2,
            concat([
              fStr(1, singleConv),
              fInt(2, 1),
              fInt(3, 100),
              fInt(4, 20),
            ])
          ),
          fMsg(
            2,
            concat([
              fStr(1, groupConv),
              fInt(2, 1),
              fInt(3, 100),
              fInt(4, 20),
            ])
          ),
        ]),
      decode: decodePullMsgsResp,
    },
    {
      id: 1003,
      name: 'WSSendMsg 发送单聊文本消息',
      build: () => msgData(buildMsgData(ctx)),
      decode: decodeSendMsgResp,
    },
    {
      id: 1003,
      name: 'WSSendMsg 发送群聊@消息',
      build: () => msgData(buildGroupAtMsgData(ctx)),
      decode: decodeSendMsgResp,
      tag: '1003-group',
    },
    {
      id: 1005,
      name: 'WSPullMsg 按会话+seq列表精确拉取',
      build: () =>
        concat([
          fStr(1, ctx.user),
          fMsg(2, concat([fStr(2, singleConv), concat([varint(1), varint(2), varint(3)].map((v, i) => concat([tag(3, 0), v])))]))
        ]),
      decode: decodePullMsgsResp,
    },
    {
      id: 1006,
      name: 'WSGetConvMaxReadSeq 获取会话已读/最大seq',
      build: () =>
        concat([fStr(1, ctx.user), fStrList(2, [singleConv, groupConv]), fBool(3, true)]),
      decode: decodeSeqsResp,
    },
    {
      id: 1007,
      name: 'WsPullConvLastMessage 拉取会话最后一条消息',
      build: () => concat([fStr(1, ctx.user), fStrList(2, [singleConv, groupConv])]),
      decode: decodeLastMsgResp,
    },
    {
      id: 2005,
      name: 'WSSubUserOnlineStatus 订阅用户在线状态',
      build: () =>
        concat([fStrList(1, [ctx.recv, 'test_user_003']), fStrList(2, ['test_user_004'])]),
      decode: () => '(服务端未返回数据体，订阅结果通过 2001 推送下发)',
    },
    {
      id: 2004,
      name: 'WSSetBackgroundStatus 设置前后台状态（服务端暂未实现）',
      build: () => concat([fStr(1, ctx.user), fBool(2, false)]),
      decode: () => '(not support)',
    },
    {
      id: 2003,
      name: 'WSLogoutMsg 登出（服务端返回 not support 后关闭连接）',
      build: () => Buffer.alloc(0),
      decode: () => '(not support)',
      last: true,
    },
  ];
}

// ============================== 极简 protobuf 解码 ==============================

function readVarint(buf, pos) {
  let shift = 0n;
  let result = 0n;
  while (pos < buf.length) {
    const b = buf[pos++];
    result |= BigInt(b & 0x7f) << shift;
    if (!(b & 0x80)) break;
    shift += 7n;
  }
  return [result, pos];
}

/** 通用字段解析：返回 [{field, wire, value}]，value 为 number 或 Buffer */
function parseFields(buf) {
  const out = [];
  let pos = 0;
  while (pos < buf.length) {
    let f, w;
    [f, pos] = readVarint(buf, pos);
    const field = Number(f >> 3n);
    const wire = Number(f & 7n);
    if (wire === 0) {
      let v;
      [v, pos] = readVarint(buf, pos);
      out.push({ field, wire, value: Number(v) });
    } else if (wire === 2) {
      let len;
      [len, pos] = readVarint(buf, pos);
      const start = pos;
      pos += Number(len);
      out.push({ field, wire, value: buf.subarray(start, pos) });
    } else if (wire === 5) {
      out.push({ field, wire, value: buf.readUInt32LE(pos) });
      pos += 4;
    } else if (wire === 1) {
      out.push({ field, wire, value: buf.readBigUInt64LE(pos).toString() });
      pos += 8;
    } else {
      throw new Error('unsupported wire type: ' + wire);
    }
  }
  return out;
}

function getStr(fields, field) {
  const it = fields.find((f) => f.field === field && f.wire === 2);
  return it ? it.value.toString('utf8') : undefined;
}
function getInt(fields, field) {
  const it = fields.find((f) => f.field === field && f.wire === 0);
  return it ? it.value : undefined;
}
function getAll(fields, field) {
  return fields.filter((f) => f.field === field);
}

function decodeMsgData(buf) {
  const f = parseFields(buf);
  return {
    sendID: getStr(f, 1),
    recvID: getStr(f, 2),
    groupID: getStr(f, 3),
    clientMsgID: getStr(f, 4),
    serverMsgID: getStr(f, 5),
    senderPlatformID: getInt(f, 6),
    senderNickname: getStr(f, 7),
    sessionType: getInt(f, 9),
    msgFrom: getInt(f, 10),
    contentType: getInt(f, 11),
    content: getStr(f, 12),
    seq: getInt(f, 14),
    sendTime: getInt(f, 15),
    createTime: getInt(f, 16),
    status: getInt(f, 17),
  };
}

function decodePullMsgs(buf) {
  const f = parseFields(buf);
  return {
    msgs: getAll(f, 1).map((x) => decodeMsgData(x.value)),
    isEnd: getInt(f, 2) === 1,
    endSeq: getInt(f, 3),
  };
}

/** map<string, PullMsgs>：field 1 普通消息 / field 2 通知消息 */
function decodePullMsgsResp(buf) {
  const f = parseFields(buf);
  const pick = (field) =>
    getAll(f, field).map((x) => {
      const e = parseFields(x.value);
      return { conversationID: getStr(e, 1), ...decodePullMsgs(Buffer.from(getStrBytes(e, 2) || [])) };
    });
  return { msgs: pick(1), notificationMsgs: pick(2) };
}

function getStrBytes(fields, field) {
  const it = fields.find((f) => f.field === field && f.wire === 2);
  return it ? it.value : undefined;
}

function decodeMaxSeqResp(buf) {
  const f = parseFields(buf);
  const mapStr = (field) =>
    getAll(f, field).map((x) => {
      const e = parseFields(x.value);
      return { conversationID: getStr(e, 1), value: getInt(e, 2) };
    });
  return { maxSeqs: mapStr(1), minSeqs: mapStr(2) };
}

function decodeSendMsgResp(buf) {
  const f = parseFields(buf);
  return {
    serverMsgID: getStr(f, 1),
    clientMsgID: getStr(f, 2),
    sendTime: getInt(f, 3),
    modify: getStrBytes(f, 4) ? decodeMsgData(Buffer.from(getStrBytes(f, 4))) : undefined,
  };
}

function decodeSeqsResp(buf) {
  const f = parseFields(buf);
  const seqs = getAll(f, 1).map((x) => {
    const e = parseFields(x.value);
    const s = parseFields(Buffer.from(getStrBytes(e, 2) || []));
    return { conversationID: getStr(e, 1), maxSeq: getInt(s, 1), hasReadSeq: getInt(s, 2), maxSeqTime: getInt(s, 3) };
  });
  return { seqs, pinnedConversationIDs: getAll(f, 2).map((x) => x.value.toString('utf8')) };
}

function decodeLastMsgResp(buf) {
  const f = parseFields(buf);
  return {
    msgs: getAll(f, 1).map((x) => {
      const e = parseFields(x.value);
      return { conversationID: getStr(e, 1), msg: decodeMsgData(Buffer.from(getStrBytes(e, 2) || [])) };
    }),
  };
}

// ============================== 外层封装 ==============================

function envelope(reqIdentifier, dataBase64, ctx, msgIncr) {
  return {
    reqIdentifier,
    token: ctx.token,
    senderID: ctx.user,
    operationID: 'apifox-' + msgIncr + '-' + Math.random().toString(36).slice(2, 8),
    msgIncr: String(msgIncr),
    data: dataBase64,
  };
}

function parseArgs(argv) {
  const ctx = {
    url: 'ws://127.0.0.1:50001/ws',
    user: 'user_001',
    token: 'test_token',
    platform: 1,
    deviceID: 'apifox_device_001',
    compression: 'none',
    sdkType: 'normal',
    recv: 'user_002',
    group: '93',
    nickname: '测试用户001',
    delay: 300,
    only: null,
    print: false,
    dump: null,
    decode: null,
  };
  for (const a of argv.slice(2)) {
    if (a === '--print') ctx.print = true;
    else if (a.startsWith('--decode=')) ctx.decode = a.slice('--decode='.length);
    else if (a.startsWith('--dump=')) ctx.dump = a.slice('--dump='.length);
    else {
      const m = /^--([A-Za-z]+)=(.*)$/.exec(a);
      if (!m) continue;
      const key = m[1];
      const val = m[2];
      if (key === 'only') ctx.only = val.split(',').map((s) => s.trim());
      else if (key === 'url') ctx.url = val;
      else if (key in ctx) ctx[key] = val;
    }
  }
  return ctx;
}

function buildURL(ctx) {
  const q = new URLSearchParams({
    userID: ctx.user,
    token: ctx.token,
    platformID: String(ctx.platform),
    deviceID: ctx.deviceID,
    sendResponse: 'true',
    isBackground: 'false',
    operationID: 'apifox-op-0001',
    compression: ctx.compression,
    sdkType: ctx.sdkType,
    sdkVersion: '1.0.0',
  });
  const base = ctx.url.split('?')[0].replace(/\/$/, '');
  return base + '?' + q.toString();
}

// ============================== 运行模式 ==============================

function printMode(ctx) {
  console.log('WebSocket 连接地址（Apifox「地址」栏粘贴）：');
  console.log(buildURL(ctx));
  console.log('\n各指令报文（Apifox「消息」栏逐条粘贴，类型选 Text）：');
  console.log(`\n--- 鉴权令牌获取（先执行，把 token 填到 {{token}}）---
POST http://127.0.0.1:18880/api/v1/auth/get_user_token
Body: {"platformID":${ctx.platform},"userID":"${ctx.user}"}`);
  let incr = 1;
  for (const c of buildCases(ctx)) {
    const data = c.build();
    const body = envelope(c.id, data.toString('base64'), ctx, incr);
    console.log(`\n--- [msgIncr=${incr}] ${c.id} ${c.name} ---`);
    console.log(JSON.stringify(body, null, 2));
    if (data.length > 0) {
      console.log('// data 解码后的 protobuf 字段:');
      console.log('// ' + JSON.stringify(decodeAny(c.id, data)));
    }
    incr++;
  }
  console.log('\n提示：msgIncr 每发一条可自增，用于把响应与请求对应起来；response 会原样带回 msgIncr。');
}

/** 请求体没有对应的响应解析器，这里按同名字段粗略回显，便于核对参数 */
function decodeAny(id, buf) {
  try {
    if (id === 1003) return decodeMsgData(buf);
    if (id === 1005) {
      const f = parseFields(buf);
      return {
        userID: getStr(f, 1),
        conversations: getAll(f, 2).map((x) => {
          const e = parseFields(x.value);
          return { conversationID: getStr(e, 2), seqs: getAll(e, 3).map((s) => s.value) };
        }),
      };
    }
  } catch (e) {
    return { error: String(e) };
  }
  return '(见脚本内构造逻辑)';
}

async function runMode(ctx) {
  if (typeof WebSocket === 'undefined') {
    console.error('当前 Node 不支持内置 WebSocket，请升级到 Node >= 22（或改用 ws 包）。');
    process.exit(1);
  }
  const url = buildURL(ctx);
  console.log('连接: ' + url);
  const ws = new WebSocket(url);
  const pending = new Map();
  let incr = 0;

  const opened = new Promise((resolve, reject) => {
    ws.addEventListener('open', () => resolve());
    ws.addEventListener('error', (e) => reject(new Error('WebSocket 连接失败: ' + (e.message || e.type))));
  });

  ws.addEventListener('message', (ev) => {
    let resp;
    try {
      resp = JSON.parse(typeof ev.data === 'string' ? ev.data : Buffer.from(ev.data).toString('utf8'));
    } catch (e) {
      console.log('收到非 JSON 消息: ' + ev.data);
      return;
    }
    const c = pending.get(resp.msgIncr);
    console.log(`\n<<< 响应 reqIdentifier=${resp.reqIdentifier} msgIncr=${resp.msgIncr} errCode=${resp.errCode} errMsg=${resp.errMsg}`);
    const data = resp.data ? Buffer.from(resp.data, 'base64') : Buffer.alloc(0);
    if (data.length > 0 && c && c.decode) {
      try {
        console.log('<<< data 解析: ' + JSON.stringify(c.decode(data), null, 2));
      } catch (e) {
        console.log('<<< data 解析失败(' + e + ')，原始 base64: ' + resp.data);
      }
    } else if (resp.data) {
      console.log('<<< data(base64): ' + resp.data);
    }
  });

  ws.addEventListener('close', (ev) => {
    console.log(`\n连接已关闭: code=${ev.code} reason=${ev.reason || '(空)'}`);
  });

  await opened;
  console.log('连接成功，开始依次发送指令...');

  for (const c of buildCases(ctx)) {
    if (ctx.only && !ctx.only.includes(c.tag || String(c.id)) && !ctx.only.includes(String(c.id))) continue;
    incr++;
    const data = c.build();
    const body = envelope(c.id, data.toString('base64'), ctx, incr);
    pending.set(String(incr), c);
    console.log(`\n>>> [msgIncr=${incr}] ${c.id} ${c.name}`);
    console.log('>>> ' + JSON.stringify({ ...body, token: body.token.slice(0, 12) + '...' }));
    ws.send(JSON.stringify(body));
    await sleep(Number(ctx.delay));
    if (c.last) break;
  }

  await sleep(1000);
  ws.close();
  await sleep(200);
  process.exit(0);
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

/** 服务端响应体解析器：按指令码选择 */
const RESPONSE_DECODERS = {
  msg: decodeMsgData,
  1001: decodeMaxSeqResp,
  1002: decodePullMsgsResp,
  1003: decodeSendMsgResp,
  1005: decodePullMsgsResp,
  1006: decodeSeqsResp,
  1007: decodeLastMsgResp,
};

function main() {
  const ctx = parseArgs(process.argv);
  if (ctx.decode) {
    const i = ctx.decode.indexOf(':');
    const tag = i > 0 ? ctx.decode.slice(0, i) : 'msg';
    const b64 = i > 0 ? ctx.decode.slice(i + 1) : ctx.decode;
    const decode = RESPONSE_DECODERS[tag] || decodeMsgData;
    const buf = Buffer.from(b64, 'base64');
    console.log('原始 hex: ' + buf.toString('hex'));
    console.log(JSON.stringify(decode(buf), null, 2));
    return;
  }
  if (ctx.dump) {
    const rows = buildCases(ctx).map((c) => ({
      tag: c.tag || String(c.id),
      id: c.id,
      name: c.name,
      data: c.build().toString('base64'),
    }));
    fs.writeFileSync(ctx.dump, rows.map((r) => JSON.stringify(r)).join('\n') + '\n');
    console.log(`已写出 ${rows.length} 条 payload 到 ${ctx.dump}`);
    return;
  }
  if (ctx.print) {
    printMode(ctx);
    return;
  }
  runMode(ctx).catch((e) => {
    console.error(String(e));
    process.exit(1);
  });
}

main();
