# 关怀通知推送服务

面向关怀对象及其家属的后端服务：维护对象名单，按频率发送问候，记录回复确认，并在超时未确认时向家属发出告警。

## Docker Compose 快速启动

```bash
cp .env.example .env
docker compose up -d --build
curl http://localhost:3242/healthz
docker compose down
```

首次启动前必须执行 `cp .env.example .env`。服务对外端口为 **3242**，MySQL 对外端口为 **5742**。Docker 内部后端统一监听 `8080`。本实现中的默认 `LogSMSSender` 不会调用真实运营商，而是以结构化日志和 `sms_logs` 数据库记录模拟成功发送，便于安全验收；生产环境可替换为实现 `SMSProvider` 的真实供应商适配器。

## 主要功能

- 关怀对象增删改查：姓名、电话、每日/每周/每月频率、开始时间和 Active/Paused 状态。
- `robfig/cron/v3` 定时扫描应问候对象，随机取启用模板并记录发送日志。
- 家属订阅指定对象；对象超过 `CONFIRM_TIMEOUT_MINUTES` 未确认时自动发告警。
- 短信回调或人工确认，持久化确认记录并更新最近确认时间。
- 管理员状态列表，按“平安 / 待确认 / 失联 / 已暂停”显示原因。
- 节日、天气、季节、通用分类的模板库 CRUD。
- 内部日志查询和 API Key 保护的外部日志查询。

## 本地开发

需要 Go 1.22+ 与可用的 MySQL 8.0：

```bash
cp .env.example .env
# 将 .env 中 DB_HOST 改为本地 MySQL 地址（默认是 127.0.0.1）
cd backend
go mod tidy
go test ./...
go run ./cmd/server
```

配置通过环境变量读取；未显式设置时本地后端端口为 `3242`。Compose 已为容器配置 `PORT=8080`、`DB_HOST=mysql`。Cron 表达式可借助 `CRON_GREETING` 和 `CRON_ALERT` 调整，默认分别为每小时一次和每 10 分钟一次。

## 技术栈

| 范畴 | 技术 |
| --- | --- |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 定时调度 | github.com/robfig/cron/v3 |
| 日志 | Go `log/slog` 结构化请求/业务日志 |
| 部署 | Docker Compose V2 |

## 项目目录

```text
.
├── backend/
│   ├── cmd/server/main.go          # 依赖装配、迁移、Cron、HTTP 生命周期
│   ├── internal/
│   │   ├── config/ model/ repository/ service/
│   │   ├── scheduler/ handler/ router/ middleware/
│   │   ├── dto/ constants/
│   ├── api/openapi.yaml
│   ├── migrations/
│   └── Dockerfile
├── database/init.sql
├── docker-compose.yml
├── .env.example
└── README.md
```

## 主要 API

统一响应：`{"code":0,"message":"ok","data":...}`；分页参数为 `page`、`page_size`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/healthz` | 健康检查 |
| POST / GET | `/api/v1/recipients` | 新增 / 分页查询关怀对象 |
| GET / PUT / DELETE | `/api/v1/recipients/:id` | 查询 / 修改 / 删除对象 |
| POST | `/api/v1/recipients/:id/confirmations` | 短信回调或人工确认 |
| POST / GET | `/api/v1/recipients/:id/subscriptions` | 新增 / 查询家属订阅 |
| POST / GET | `/api/v1/templates` | 新增 / 查询模板，支持 `?category=weather` |
| GET / PUT / DELETE | `/api/v1/templates/:id` | 模板详情 / 更新 / 删除 |
| GET | `/api/v1/sms-logs` | 内部发送日志查询 |
| GET | `/api/v1/admin/statuses` | 管理员对象状态查询 |
| POST | `/api/v1/admin/jobs/greetings` | 手工触发一次问候扫描（验收辅助） |
| POST | `/api/v1/admin/jobs/alerts` | 手工触发一次超时告警扫描（验收辅助） |
| GET | `/api/v1/external/sms-logs` | 外部日志接口，须 `X-API-Key` |

示例：

```bash
curl -X POST http://localhost:3242/api/v1/recipients \
  -H 'Content-Type: application/json' \
  -d '{"name":"王阿姨","phone":"13800138000","care_frequency":"daily","care_start_at":"2026-08-14T00:00:00Z"}'
```

完整接口概要在 [`backend/api/openapi.yaml`](backend/api/openapi.yaml)。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `gbcarenotify` | Compose 项目名 |
| `DB_NAME` / `DB_USER` / `DB_PASSWORD` | `care_notify` / `care_user` / `care_password` | 应用数据库凭据 |
| `DB_ROOT_PASSWORD` | `root_password` | MySQL root 密码 |
| `DB_PORT` | `5742` | MySQL 宿主机端口 |
| `BACKEND_PORT` | `3242` | 后端宿主机端口 |
| `API_KEY` | `change-me-for-external-query` | 外部日志接口访问密钥 |
| `CONFIRM_TIMEOUT_MINUTES` | `1440` | 未确认超时分钟数 |
| `CRON_GREETING` / `CRON_ALERT` | `0 * * * *` / `*/10 * * * *` | 问候和告警 Cron |

## License

MIT License。
