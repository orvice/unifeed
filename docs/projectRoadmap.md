这是一个名为 unifeed 的 go 服务，主要功能如下：
- 通过配置文件，配置 Feed
- Feed 能配置 Mastodon/Bluesky
- 将 Mastodon/Bluesky 配置的用户信息，将用户关注用户的 Timeline 转换成 RssFeed
- 根据配置里的 name，访问 /feeds/{name} 返回配置里的 timeline 的 rss feed
- 根据配置里的 feed rss url地址，为 rss 添加 ai 总结

## 高阶目标与任务拆分
- [x] 配置结构支持 Mastodon 和 Bluesky
- [x] 配置加载与校验
- [x] Mastodon timeline 拉取与 RSS 转换
- [x] Bluesky timeline 拉取与 RSS 转换
- [x] Feed 聚合逻辑（根据配置动态选择源）
- [x] /feeds/{name} 路由与 handler 实现
- [x] 错误处理与异常响应
- [ ] 单元测试与集成测试
- [ ] 文档补充

## RSS Feed AI 总结功能
- [ ] 配置扩展
  - [ ] 添加 RSS Feed 配置结构
  - [ ] 添加 S3 存储配置
  - [ ] 添加 AI 服务配置
  - [ ] 配置验证与加载

- [ ] RSS Feed 处理
  - [ ] RSS Feed 解析与内容提取
  - [ ] 内容去重与缓存
  - [ ] 内容格式化与预处理

- [ ] AI 总结服务
  - [ ] AI 服务集成
  - [ ] 内容总结生成
  - [ ] 总结结果格式化

- [ ] S3 存储服务
  - [ ] S3 客户端集成
  - [ ] 总结内容存储
  - [ ] 存储路径管理
  - [ ] 访问权限控制

- [ ] 定时任务
  - [ ] 任务调度器实现
  - [ ] 定时更新逻辑
  - [ ] 失败重试机制
  - [ ] 任务状态监控

- [ ] API 扩展
  - [ ] 总结内容查询接口
  - [ ] 手动触发更新接口
  - [ ] 任务状态查询接口

## 已完成任务
- [x] Gin 路由初始化
- [x] 配置文件结构初步定义（Mastodon）
- [x] Mastodon timeline 拉取与 RSS 转换实现
- [x] Bluesky timeline 拉取与 RSS 转换实现
- [x] 路由处理 Mastodon/Bluesky feed 请求
- [x] 基础错误处理与异常响应
- [x] Docker 构建与发布配置