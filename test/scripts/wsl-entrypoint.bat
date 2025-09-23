:: When ssh-ing into a windows machine we enter into a cmd.exe prompt.
:: This batch script imports a wsl image and runs a test script inside of wsl.

set wsl_img=%1
set test_script=%2
set img_config=%3

echo "import wsl img"
set errorlevel=
"C:\Program Files\WSL\wsl.exe" --import ibwsl ibwsl "C:\Users\azureuser\%wsl_img%"
echo "import wsl img result: %errorlevel%"
if %errorlevel%!==!0 exit /b 1

echo "run test script"
set errorlevel=
"C:\Program Files\WSL\wsl.exe" -d ibwsl "/mnt/c/Users/azureuser/%test_script%" "/mnt/c/Users/azureuser/%img_config%"
echo "test script result: %errorlevel%"
if %errorlevel%!==!0 exit /b 1

exit /b 0
