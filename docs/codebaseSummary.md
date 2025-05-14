## Key Components and Their Interactions
- 配置管理（internal/conf）：负责 Feed 源（Mastodon/Bluesky）配置结构和加载
- HTTP 路由（internal/http）：负责接口路由和 handler，已实现 /feeds/:name 路由，集成 service 层生成 RSS，现支持 Mastodon 和 Bluesky
- Service 层（internal/service）：负责 timeline 拉取与 RSS 生成
- 测试（test）：包含 Mastodon/Bluesky service 的基础单元测试

## Data Flow
- 启动时加载配置，路由根据 name 查找 feed，调用 service 拉取 timeline 并生成 RSS

## External Dependencies
- Gin（HTTP 框架）
- 计划支持 Mastodon/Bluesky API

## Recent Significant Changes
- 计划扩展 Feed 配置结构，支持 Bluesky
- 即将实现配置文件加载与校验逻辑

## User Feedback Integration and Its Impact on Development
- 暂无
