# Init knowledge base

Инициализируй базу знаний проекта для отчётов специалистов и обучения AI-агентов.

## Создай (если ещё нет)

```text
docs/knowledge-base/
  README.md
  INDEX.md
  AGENTS.md
  audits/
    .gitkeep
```

### README.md
Кратко: это постоянная память аудитов (BA, UX, FE, BE, QA, DevOps, Security). Источник правды для повторных решений.

### INDEX.md
Таблица-каталог (пока пустая, с заголовками Date | Title | Path | Domains | Top severity).

### AGENTS.md
Инструкция для AI:
1. Перед новым аудитом читай INDEX.md и последний consolidated report
2. Ссылайся на пути KB в рекомендациях
3. Не затирай историю — только добавляй новые папки audits/
4. При `/full-audit` пиши сюда результаты

После создания коротко подтверди пути. Не запускай полный аудит, если пользователь не просил.

$ARGUMENTS
