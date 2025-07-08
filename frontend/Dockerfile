# Этап сборки
FROM node:18-alpine as build

# Установка рабочей директории
WORKDIR /app

# Копирование файлов зависимостей
COPY package.json package-lock.json ./

# Установка зависимостей
RUN npm ci

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN npm run build

# Этап запуска
FROM nginx:alpine

# Копирование результатов сборки из директории "dist"
COPY --from=build /app/dist /usr/share/nginx/html

# (Опционально) Копирование настроек nginx для поддержки React Router
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Открываем порт
EXPOSE 80

# Запуск nginx
CMD ["nginx", "-g", "daemon off;"]