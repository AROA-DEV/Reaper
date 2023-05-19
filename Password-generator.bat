@echo off
setlocal EnableDelayedExpansion

set "characters=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_-+=<>?"
set "passwords=10"
set "length=34"

echo Generating passwords...

set "generatedPasswords="
for /L %%i in (1,1,%passwords%) do (
    set "password="
    :generate_password
    for /L %%j in (1,1,%length%) do (
        set /A "randomIndex=!random! %% 82"
        for %%k in (!randomIndex!) do (
            set "char=!characters:~%%k,1!"
            set "password=!password!!char!"
        )
    )
    if "!password:~0,%length%!" NEQ "" (
        set "generated_password=!password:~0,%length%!"
        setlocal enabledelayedexpansion
        if "!generated_password!"=="!password!" (
            set "duplicateFound=false"
            for %%p in (!generatedPasswords!) do (
                if "%%p"=="!generated_password!" (
                    set "duplicateFound=true"
                    goto generate_password
                )
            )
            if "!duplicateFound!"=="false" (
                set "generatedPasswords=!generatedPasswords! !generated_password!"
                echo Password %%i: !generated_password!
            )
        ) else (
            echo Password %%i: Error generating password!
            endlocal
            goto generate_password
        )
        endlocal
    ) else (
        echo Password %%i: Error generating password!
        goto generate_password
    )
)

endlocal