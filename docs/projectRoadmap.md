这是一个名为 unifeed 的 go 服务，主要功能如下：
- 通过配置文件，配置 Feed
- Feed 能配置 Mastodon/Bluesky
- 将 Mastodon/Bluesky 配置的用户信息，将用户关注用户的 Timeline 转换成 RssFeed
- 根据配置里的 name，访问 /feeds/{name} 返回配置里的 timeline 的 rss feed

## 高阶目标与任务拆分
- [x] 配置结构支持 Mastodon 和 Bluesky
- [x] 配置加载与校验
- [ ] Mastodon timeline 拉取与 RSS 转换
- [ ] Bluesky timeline 拉取与 RSS 转换
- [ ] Feed 聚合逻辑（根据配置动态选择源）
- [ ] /feeds/{name} 路由与 handler 实现
- [ ] 错误处理与异常响应
- [ ] 单元测试与集成测试
- [ ] 文档补充

## 已完成任务
- [x] Gin 路由初始化
- [x] 配置文件结构初步定义（Mastodon）