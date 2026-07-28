# Refresh agent pack (only your roles)

Обнови reference-файлы **только** для ролей из ростера (без лишних агентов).

Источник: [VoltAgent/awesome-claude-code-subagents](https://github.com/VoltAgent/awesome-claude-code-subagents) (MIT)

## Разрешённый набор

| Локальный файл | Upstream (если имя отличается) |
|----------------|--------------------------------|
| business-analyst.md | categories/08-business-product/business-analyst.md |
| ux-ui-designer.md | categories/08-business-product/ux-researcher.md |
| frontend-developer.md | categories/01-core-development/frontend-developer.md |
| backend-developer.md | categories/01-core-development/backend-developer.md |
| qa-tester.md | categories/04-quality-security/qa-expert.md |
| devops-engineer.md | categories/03-infrastructure/devops-engineer.md |
| security-engineer.md | categories/03-infrastructure/security-engineer.md |

1. Скачай только эти файлы в `~/.cursor/agents/_references/voltagent/`
2. Переименуй upstream `ux-researcher` → `ux-ui-designer.md`, `qa-expert` → `qa-tester.md`
3. Удали любые другие `.md` в этой папке (кроме `SOURCE.md`)
4. Синхронизируй `_references/` в `~/.claude/agents/` и `~/.codex/agents/`
5. Основные агенты `~/.cursor/agents/*.md` не затирай — только refs

$ARGUMENTS
