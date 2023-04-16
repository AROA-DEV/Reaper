@echo off
echo.
echo.
mkdir Copy
cd Copy
mkdir Documentos
mkdir Imágenes
mkdir Escritorio
echo.
set /p usb="What is the external drive letter? "
:start
echo.
echo Copy Documentos [1]
echo Copy Imágenes [2]
echo Copy Escritorio [3]
echo Copy All [all]
echo.
echo.
set /p one="Would you like to continue? "
if '%one%'=='1' goto one
if '%one%'=='2' goto two
if '%one%'=='3' goto three
if '%one%'=='all' goto all

:one
xcopy  /v /s /e /h /i /y "%userprofile%\Documentos" "%usb%:\Copy\Documentos"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documentos" "%usb%:\Copy\Documentos"
cls

:two
xcopy  /v /s /e /h /i /y "%userprofile%\Imágenes" "%usb%:\Copy\Imágenes"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documentos" "%usb%:\Copy\Documentos"
cls

:three
xcopy  /v /s /e /h /i /y "%userprofile%\Escritorio" "%usb%:\Copy\Escritorio"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documentos" "%usb%:\Copy\Documentos"
cls

:all
xcopy  /v /s /e /h /i /y "%userprofile%\Documentos" "%usb%:\Copy\Documentos"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documentos" "%usb%:\Copy\Documentos"
xcopy  /v /s /e /h /i /y "%userprofile%\Imágenes" "%usb%:\Copy\Imágenes"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Imágenes" "%usb%:\Copy\Imágenes"
xcopy  /v /s /e /h /i /y "%userprofile%\Escritorio" "%usb%:\Copy\Escritorio"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Escritorio" "%usb%:\Copy\Escritorio"
cls

goto start