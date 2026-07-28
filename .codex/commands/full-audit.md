# Full team audit

Запусти **полный аудит всеми специалистами** и сохрани отчёты в базу знаний проекта.

## Что сделать

1. Используй субагента `full-audit-orchestrator` (или сам оркестрируй по его протоколу).
2. Параллельно вызови всех:
   - `business-analyst`
   - `ux-ui-designer`
   - `frontend-developer`
   - `backend-developer`
   - `qa-tester`
   - `devops-engineer`
   - `security-engineer`
3. Собери executive summary + единый backlog.
4. **Обязательно запиши файлы** в проект:

```text
docs/knowledge-base/
  README.md
  INDEX.md
  AGENTS.md
  audits/YYYY-MM-DD-full-audit/
    00-executive-summary.md
    01-business-analyst.md
    02-ux-ui-designer.md
    03-frontend-developer.md
    04-backend-developer.md
    05-qa-tester.md
    06-devops-engineer.md
    07-security-engineer.md
    99-consolidated-report.md
```

5. Обнови `docs/knowledge-base/INDEX.md` (новые сверху).
6. В ответе пользователю дай: путь к `99-consolidated-report.md`, топ-10 действий, критичные риски.

## Контекст от пользователя

$ARGUMENTS

Если аргументы пустые — аудируй текущий проект целиком (MVP-глубина: риски и реализуемые рекомендации, без воды).
