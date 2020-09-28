# GoFileWatcher
Кроссплатформенная программа для слежения за директориями и файлами с возможностью выполнять различные команды для 
конкретных файлов по их полному пути, имени, либо по регулярному выражению при возникновении изменений.

# Установка

* Вручную скачать из [списка релизов](https://github.com/karamush/GoFileWatcher/releases) под нужную архитектуру и ОС.
* Автоматически из репозитория:
(для этого нужна корректная установка Go)
```
$ go get github.com/karamush/GoFileWatcher
$ go install github.com/karamush/GoFileWatcher
```
после чего будет доступен исполняемый файл GoFileWatcher.

**Сборка и изменение**
Для полноценной ручной сборки нужно выполнить следующие подготовительные шаги:
* Установить goversioninfo (для иконки приложения под Windows):
```
go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
```
* Установить GOX (не обязательно, но нужно для запуска build.bat для автоматической сборки):
```
go get github.com/mitchellh/gox
```
* Запустить build.bat для автоматической сборки или собрать вручную:
```
go generate
go build
```

# Возможности

* Слежение за директориями и файлами (в том числе и рекурсивно)
* Запуск команд при возникновении событий (изменение, создание, etc)
* Передача информации о событии в stdin запускаемой команды
* Вывод событий изменений в консоль
* Показ или скрытие при запуске списка объектов для наблюдения
* Изменение времени проверки изменений
* Исключение скрытых файлов (dotfiles) из списка наблюдаемых объектов
* Запуск команд для различных событий в специальном -actions файле, где можно реагировать на полный путь, имя файла или
регулярное выражение ([подробнее](#Actions-List))
  
# Использование

**GoFileWatcher [params] [path1] [path2] [pathN...]**

**gofilewatcher -h**:
```
Usage of gofilewatcher:
  -actions string
        путь к файлу со списком фильтров для файлов, событий и их действий (default "action_list.ini")
  -cmd string
        команда, запускаемая при возникновении событий
  -dotfiles
        следить за скрытыми файлами
  -ignore string
        список игнорируемых файлов (через запятую)
  -interval string
        интервал проверки изменений (default "500ms")
  -keepalive
        продолжать работу, даже если cmd вернула код возврата != 0 (default true)
  -list
        показать список файлов для наблюдения при старте (default true)
  -logevents
        выводить в stdout изменения в файлах и директориях (default true)
  -pipe
        передать информацию о событии в stdin команды
  -recursive
        следить за директориями рекурсивно (default true)
  -startcmd
        запустить команду cmd при старте приложения и начале слежения
```

Запустить можно без параметров, тогда будут использованы настройки по-умолчанию, а слежение будет рекурсивно от текущей
директории.
Следить можно как за файлами, так и за директориями.
Подробнее смотри в [примерах использования](#Примеры-использования).

# Actions List
С помощью action-list можно задать различные команды для разных операций для указанных файлов как по полному пути, имени
файла или же по регулярному выражению.

Для использования этой возможности, нужно создать файл и передать его при запуске программы в параметре -actions.
Если путь к файлу не передан, то будет попытка загрузить файл по имени "action_list.ini".

## Структура файла
Файл представляет собой обычный ini-файл, где в названии секции пишется путь к файлу или директории, либо только имя файла или 
директории, либо регулярное выражение. Регулярное выражение должно начинаться с тильды (~), чтоб дать программе понять, 
что это именно регулярное выражение. 
А в самой секции в названии параметра можно указывать операцию над файлом (об этом ниже) и в значении параметра указывать
команду, которая выполнится при возникновении данного события. Также можно использовать некоторые подстановочные переменные,
чтоб передать в запускаемую команду имя файла или полный путь к нему или ещё что-то (о доступных переменных ниже). 

**Пример файла**:
```
# используется регулярное выражение, которое в директории D:\tmp\test\ будет реагировать на любые текстовые файлы.
[~D:\\tmp\\test\\.*.txt]
write=cmd.exe /c echo File changed: {{ .name }} = {{ .path }}
remove=cmd.exe /c echo FILE REMOVED: {{ .path }}
rename=cmd.exe /c echo FILE RENAMED. OldPath: {{ .oldpath }} NewPath: {{ .path }}

# здесь указано лишь имя, а значит программа среагирует на файл с таким именем в любой директории под наблюдением
[test_file.txt]
write=
create=
remove=
rename=
chmod=
move=

# здесь реакция будет только на файл по конкретному пути. 
[D:\tmp\test_file.txt]
write=cmd.exe /c echo {{ .name }}
```

## События изменений наблюдаемых объектов
При возникновении изменений, когда файл создаётся, меняется, удаляется или переименовывается (это же относится и к директориям), 
генерируются события, и именно их нужно указывать в action-list:
```
create  - создание объекта
write   - изменение (для директорий тоже работает, если в ней что-то из файлов изменилось)
remove  - объект удалён
rename  - объект переименован
chmod   - изменились права доступа
move    - объект перемещён
```

## Порядок обработки секций

При возникновении изменений в первую очередь ищется секция и команда для этого события по полному пути, затем по имени
файла, затем ищется совпадение шаблона по регулярному выражению.
После нахождения нужной команды для нужного объекта поиск прекращается.
Таким образом можно задавать приоритеты обработки.
Как показано в примере файла выше, там есть секция, содержащая полный путь к файлу `D:\tmp\test_file.txt`, а также просто 
`test_file.txt`. В таком случае, если изменения произошли в файле `D:\tmp\test_file.txt`, то сработает именно он, а секция
по имени файла будет пропущена, так как полный путь имеет более высокий приоритет обработки.

## Переменные
При запуске команд из Action-list можно использовать следующие переменные:

| Имя  | Значение |
| ----------- | -------- |
| {{ .path }}  | Полный путь к объекту  |
| {{ .name }}  | Имя объекта  |
| {{ .oldpath }} | Старый путь (при событии Move) | 
| {{ .operation }} | Имя операции-события (смотри [список](#События-изменений-наблюдаемых-объектов))|
| {{ .event }} | полное событие, включающее тип объекта (файл или директория), имя, операцию и полный путь |
| {{ .file }} | см. go тип [FileInfo](https://godoc.org/github.com/gogf/gf/internal/fileinfo) |

# Примеры использования

**Запуск с параметрами по-умолчанию**:
Достаточно сделать двойной клик по приложению или запустить из консоли, и оно уже будет работать с параметрами 
по-умолчанию, а следить будет за текущей директорией и её файлами, а также всеми вложенными директориями.

![Запуск с параметрами по-умолчанию](resources/doc_img/run_and_change.png?raw=true "Запуск программы и событие изменения файла")

**Запуск с использованием action-list**:
Например, нужно следить за текстовыми файлами в определённой директории, а при записи в файл, запускать какую-то внешнюю
команду. Допустим, просто запустим консоль и выведем там имя этого файла.
Имя файла заранее знать мы не можем, но можем составить маску имени, например, по его расширению. Регулярка для этого 
будет выглядеть так: `.*.txt`, что означает "любое количество любых символов точка txt".

Файл `action_list.ini`:
```
# для регулярных выражений нужно экранировать обратный слеш \\
[~D:\\tmp\\test\\.*.txt]
write=cmd.exe /c echo Файл изменён: Имя={{ .name }}, Путь={{ .path }}
```

Запуск программы командой `GoFileWatcher D:\tmp\test\`, но рядом ещё и action_list.ini есть.

Вот, что из этого получится:
![Запуск и выполнение команды из action_list.ini](resources/doc_img/run_actions.png?raw=true "Запуск и выполнение команды из action_list.ini")
