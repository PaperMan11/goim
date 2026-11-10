// Package msgcontent 定义 GoIM 中**服务端需要解析的消息 content JSON 结构**。
//
// 设计原则：
//   - contentType 是服务端和客户端共同的契约（见 pkg/protocol/constant/content_type.go）
//   - content 的 JSON schema：绝大多数类型（Text/Picture/Video/File/Card/Custom...）
//     由客户端 SDK 自行定义并迭代，服务端透传不解析
//   - 少数类型服务端需要读 content 字段做逻辑处理（Quote 回源、Revoke 联动、
//     AtText 推送、Merger 合并转发），这些类型的结构必须在此包内维护
//
// 客户端新增自定义 content 字段无需动此包；服务端只需在新增需要解析的类型时补充定义。
package msgcontent
