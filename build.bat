@echo off

set current_version=1.1.1
set base_dir=builds
set tpl="GoFileWatcher_{{.OS}}_{{.Arch}}"
set flags=-ldflags "-s -w"

set output_dir=%base_dir%\%current_version%\
mkdir %output_dir%

rem Удалить старый файл ресурсов
del resource.syso

rem Собрать под linux, darwin и FreeBSD
gox %flags% -os="linux darwin freebsd" -arch="386 amd64" -output=%output_dir%\%tpl%

rem Generate resource file
go generate

rem Под Windows сборка отдельно
set GOOS=windows
set GOARCH=386
go build %flags% -o %output_dir%GoFileWatcher_windows_386.exe
set GOOS=windows
set GOARCH=amd64
go build %flags% -o %output_dir%GoFileWatcher_windows_amd64.exe