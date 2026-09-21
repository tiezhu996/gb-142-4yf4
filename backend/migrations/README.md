# Migrations

当前服务在启动时使用 GORM `AutoMigrate` 创建和升级核心表；`database/init.sql` 负责容器首次初始化时的数据库引导。生产环境如需严格版本化迁移，可将 SQL 文件按版本加入本目录并在发布流程中执行。
