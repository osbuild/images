:: When ssh-ing into a windows machine we enter into a cmd.exe prompt.
:: This batch script imports a wsl image and runs a test script inside of wsl.

set wsl_img="%1"
set test_script="%2"
set img_config="%3"

'"C:\Program Files\WSL\wsl.exe"' --import ibwsl ibwsl "%wsl_img%"

set errorlevel=
'"C:\Program Files\WSL\wsl.exe"' -d ibwsl "/mnt/c/Users/azureuser/%test_script%" "%img_config%"


:: as the exit code isn't propagated to the ssh client properly, this line is grepped and inperpreted
echo "exit code of host script: %errorlevel%"

:: todo figure out how to capture the exit code?
