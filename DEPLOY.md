# 🚀 Деплой ШНЫРЬ v0.1

## Бесплатные хостинги для деплоя

### 1. Render.com (Рекомендуется)

**Преимущества:**
- Бесплатный план
- Автоматический деплой из Git
- Поддержка Docker
- SSL сертификат включен

**Инструкция:**

1. **Зарегистрируйтесь** на [render.com](https://render.com)

2. **Подключите GitHub репозиторий**

3. **Создайте новый Web Service:**
   - Выберите ваш репозиторий
   - Environment: `Docker`
   - Plan: `Free`

4. **Настройте переменные окружения:**
   ```
   DB_HOST=108.181.194.102
   DB_PORT=3306
   DB_USER=root
   DB_PASSWORD=root
   DB_NAME=octopus
   ```

5. **Нажмите "Create Web Service"**

6. **Получите URL** вида: `https://your-app-name.onrender.com`

### 2. Railway.app

**Инструкция:**

1. Зарегистрируйтесь на [railway.app](https://railway.app)
2. Подключите GitHub репозиторий
3. Выберите "Deploy from GitHub repo"
4. Настройте переменные окружения
5. Получите URL

### 3. Fly.io

**Инструкция:**

1. Установите Fly CLI: `curl -L https://fly.io/install.sh | sh`
2. Зарегистрируйтесь: `fly auth signup`
3. Создайте приложение: `fly launch`
4. Настройте переменные окружения
5. Деплой: `fly deploy`

## Переменные окружения

```bash
# Порт (обычно устанавливается хостингом автоматически)
PORT=8080

# База данных
DB_HOST=108.181.194.102
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=octopus
```

## Локальный запуск

```bash
# Сборка Docker образа
docker build -t shnyr-web-viewer .

# Запуск контейнера
docker run -p 8080:8080 \
  -e DB_HOST=108.181.194.102 \
  -e DB_PORT=3306 \
  -e DB_USER=root \
  -e DB_PASSWORD=root \
  -e DB_NAME=octopus \
  shnyr-web-viewer
```

## Проверка деплоя

После деплоя проверьте:

1. **Доступность сайта** - откройте URL в браузере
2. **Подключение к БД** - проверьте логи на наличие ошибок подключения
3. **Функциональность** - протестируйте поиск и пагинацию
4. **Мобильную версию** - проверьте адаптивность

## Возможные проблемы

### Ошибка подключения к БД
- Проверьте правильность переменных окружения
- Убедитесь, что БД доступна из интернета
- Проверьте firewall настройки

### Приложение не запускается
- Проверьте логи в панели управления хостингом
- Убедитесь, что порт настроен правильно
- Проверьте Dockerfile на ошибки

### Медленная загрузка
- Это нормально для бесплатных планов
- Первый запрос может занять 30-60 секунд
- Последующие запросы будут быстрее

## Обновление приложения

1. Внесите изменения в код
2. Закоммитьте и запушьте в Git
3. Хостинг автоматически пересоберет и перезапустит приложение

## Мониторинг

- **Render.com**: Встроенная панель мониторинга
- **Railway.app**: Метрики в реальном времени
- **Fly.io**: `fly logs` для просмотра логов

## Безопасность

⚠️ **Важно:** В продакшене используйте:
- Отдельную БД с ограниченными правами
- HTTPS соединение
- Переменные окружения для секретов
- Регулярные обновления зависимостей 