# Video converter

## Requirements

- Debian-like system (ubuntu, mint, etc...) with **apt** package manager
- Golang >1.15
- Command tool **make** (use `sudo apt install make -y` to install it)

## Configuration

All variables must be written to `.env` file in the root of the project.

Available fields you can see at `example.env` file.

## Using

- `make install -S` for download and install **ffmpeg** tool for work with video files

- `make up` - for download and start docker container with **mysql**

## Description

1. Получает видео из базы данных
2. Проверяет, заполнены ли поля в БД с форматами для 1080 720 480 360 Preview, если да - пропускает обработку
3. Загружает оригинал видео
4. Запускает многопоточную обработку всех недостающих форматов из оригинала
5. Загружает сконвертированные форматы на облако, если успешно - удаляет файл с диска
6. Обновляет записи в БД для загруженных форматов
7. Удаляет локальную копию оригинала
8. Снова проверяет, заполнены ли поля со всеми форматами, если да - удаляет оригинал видео из облака

## Handle errors

1. При любой ошибке в базе данных - сразу приложение завершит работу
2. При нажатии Ctrl+C - приложение сразу завершит работу
3. Если истечен время, указанное в переменной TIMEOUT файла .env - приложение остановит обработку новых видео, дождётся
   полного завершения обработки уже запущенных процессов и после завершит работу

При любом окончании работы - в логфайл и stdout будет выведено сообщение об общем количестве обработанных, загруженных,
сконвертированных видео и ошибках