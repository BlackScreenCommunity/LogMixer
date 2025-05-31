
# LogMixer

**LogMixer** — консольная утилита для объединения и сортировки лог-файлов.

## Установка

```bash
git clone https://github.com/yourname/logmixer.git
cd logmixer
go build -o logmixer
sudo mv logmixer /usr/local/bin/
```

## Использование

```bash
logmixer -path /путь/к/каталогу -out /путь/к/выходному/файлу
```

### Аргументы

- `-path` — путь до папки с логами (обрабатываются рекурсивно)
- `-out` — имя файла, в который будет записан отсортированный результат
- `-filters` - путь до yaml файла со словарем фильтров, которые будут применяться к каждому лог-блоку

### Пример файла с фильтрами

``` yaml
contains:
  - Add new participant
  - Calendar ForEach
```

## Пример

```bash
logmixer -path ./logs/2025-05-13 -out ./sorted.log -filters ~/.config/logmixer/unified_filters.yaml
```

## Особенности

- Поддержка многострочных сообщений (например, stack trace)
- Добавление имени и пути файла в каждую запись
- Безопасное использование памяти с возможностью задания лимита
