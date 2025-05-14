## 当前目标
- 为 Mastodon/Bluesky timeline 拉取与 RSS 生成添加单元测试（关联 projectRoadmap.md：单元测试与集成测试）
- 完善 README，简要说明项目功能、配置格式和启动方式（关联 projectRoadmap.md：文档补充）

## 步骤
- [x] internal/service 新建 bluesky.go，实现 timeline 拉取与 RSS 生成
- [x] internal/http/route.go 支持 Bluesky feed
- [x] test/service_mastodon_test.go：MastodonService 核心逻辑测试
- [x] test/service_bluesky_test.go：BlueskyService 核心逻辑测试
- [ ] 更新 codebaseSummary.md 记录测试结构
- [ ] 更新 README.md，包含：
  - 项目简介
  - 配置文件格式示例
  - 启动与测试方法
