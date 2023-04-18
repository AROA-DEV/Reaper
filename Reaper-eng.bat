@echo off
echo.
echo .
echo Reaper V-0.1
echo .
echo.
mkdir Copy
cd Copy
mkdir Documents
mkdir Images
mkdir Desktop 
echo.
set /p usb="What is the external drive letter? "
:start
echo.
echo Copy Documents [1]
echo Copy Images [2]
echo Copy Desktop  [3]
echo Copy All [all]
echo.
echo.
set /p one="Would you like to continue? "
if '%one%'=='1' goto one
if '%one%'=='2' goto two
if '%one%'=='3' goto three
if '%one%'=='all' goto all

:one
xcopy  /v /s /e /h /i /y "%userprofile%\Documents" "%usb%:\Copy\Documents"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documents" "%usb%:\Copy\Documents"
cls

:two
xcopy  /v /s /e /h /i /y "%userprofile%\Images" "%usb%:\Copy\Images"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documents" "%usb%:\Copy\Documents"
cls

:three
xcopy  /v /s /e /h /i /y "%userprofile%\Desktop " "%usb%:\Copy\Desktop "
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documents" "%usb%:\Copy\Documents"
cls

:all
xcopy  /v /s /e /h /i /y "%userprofile%\Documents" "%usb%:\Copy\Documents"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Documents" "%usb%:\Copy\Documents"
xcopy  /v /s /e /h /i /y "%userprofile%\Images" "%usb%:\Copy\Images"
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Images" "%usb%:\Copy\Images"
xcopy  /v /s /e /h /i /y "%userprofile%\Desktop " "%usb%:\Copy\Desktop "
xcopy  /v /s /e /h /i /y "%userprofile%\OneDrive\Desktop " "%usb%:\Copy\Desktop "
cls

goto start