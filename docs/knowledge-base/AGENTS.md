# AGENTS — работа с Knowledge Base

Инструкции для AI-агентов в репозитории Locator (`/root/locator_go`).

## Перед аудитом или крупными рекомендациями

1. Прочитать `docs/knowledge-base/INDEX.md`.
2. Открыть последний релевантный `audits/*/99-consolidated-report.md` (и domain-файлы при необходимости).
3. Цитировать пути KB при опоре на прошлые findings; отмечать, что может устареть.
4. **Не перезаписывать историю** — добавлять новую папку `audits/YYYY-MM-DD-…/`.
5. Для полной проверки использовать `/full-audit` (оркестратор).

## Правила записи

- Только markdown под `docs/knowledge-base/`.
- Не трогать application source, БД, `.env`, секреты, docker volumes, production configs.
- Без значений секретов в отчётах.
- Язык пользователя (русский) для summary и backlog; технические термины допустимы на английском.

## Домены

`business-analyst` · `ux-ui-designer` · `frontend-developer` · `backend-developer` · `qa-tester` · `devops-engineer` · `security-engineer`
