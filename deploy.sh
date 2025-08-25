#!/bin/bash

# Переходим в корневую директорию с проектом, где находится docker-compose.yml
cd /locator_go

# Обновляем репозиторий с новыми изменениями
git pull origin main

# Пересобираем все контейнеры (в том числе и frontend, и backend) и запускаем их в фоне
docker-compose up --build -d

echo "Деплой завершён!"