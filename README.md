# grpc_proxy_yt_thumbnail (gRPC)

![Go Version](https://img.shields.io/badge/Go-1.23.3-blue)

**grpc_proxy_yt_thumbnail** — это gRPC-приложение на Go, которое позволяет скачивать изображения-миниатюры из заданного источника через консоль или обращение к серверу .

---
## Инструкция

Для использования async метода необходимо прописать в *CMD* <br>
**Пример**
```bash
go run main.go -urls=https://www.youtube.com/watch?v=fqg89xfLJMk -urls=https://www.youtube.com/watch?v=Ue0lAU-VjEQ  -async=true
```
**Пример использования sync метода**
```bash
go run main.go -urls=https://www.youtube.com/watch?v=Ue0lAU-VjEQ
```

Но перед этим перейти через cd в папку, где находится скачанный проект

## Основные функции

- **Скачивание миниатюры** с URL-адреса.
- **Утилита командной строки**.


---

## Требования

- Go 1.23 или выше.
- Локальный доступ к `localhost` для работы gRPC-сервера.
- Установленный [protoc](https://grpc.io/docs/protoc-installation/) для генерации gRPC-кода (если будете компилировать `.proto` самостоятельно).

---

## Установка

1. **Клонируйте репозиторий**:
   ```bash
   git clone https://github.com/username/thumbnail-downloader.git
   ```

