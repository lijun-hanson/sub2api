-- 把 grok 平台加入 user_platform_quotas.platform 的 CHECK 约束。
--
-- 背景：grok 自 2026-06 起进入默认平台配额（default_platform_quotas /
-- auth_source_default_*_platform_quotas），但 142 建表时的 CHECK 仅允许
-- anthropic/openai/gemini/antigravity。自助注册时 snapshotPlatformQuotaDefaults
-- 会写入 grok 默认配额行 → 违反 CHECK → 整个注册事务被标记 aborted →
-- OAuth pending 路径 consume 会话时撞 "transaction aborted" → 500 → 清 cookie → 404。
--
-- 修复：把约束与代码平台列表（internal/domain/constants.go 的 PlatformGrok）对齐。
-- DROP ... IF EXISTS 保证可重入。
--
-- Kiro fork 说明：本 fork 通过 145_allow_kiro_user_platform_quotas.sql 已把 kiro
-- 加入约束，且线上存在 platform='kiro' 的存量行。上游原版 157 仅列 grok（无 kiro），
-- 会把 kiro 从约束中丢掉 → 存量 kiro 行违反 CHECK → 迁移失败、应用无法启动。
-- 因此这里的约束是 145(kiro) 与上游(grok) 的并集，与 domain.AllowedQuotaPlatforms
-- 及 ent/schema/user_platform_quota.go 的校验保持一致。
ALTER TABLE user_platform_quotas
    DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;

ALTER TABLE user_platform_quotas
    ADD CONSTRAINT user_platform_quotas_platform_check
    CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'kiro', 'grok'));
