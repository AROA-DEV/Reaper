# Reaper

***

![Reaper-logo](https://github.com/AROA-DEV/Reaper/blob/main/Readme/image.png?raw=true)

***

Reaper is an open-source attack tool available on GitHub that allows users to automate the process of copying files from Documents, Images, and Desktop repositories. This batch script is designed to be run on Windows operating systems while the user is logged in, and it does not breach any of the security measures implemented by Windows.

The Reaper script is very user-friendly and easy to use. Once you download the script and move it to your USB (128GB minimum), you can run it by double-clicking on the file. It will prompt you to select the directories you want to copy from, which include Documents, Images, and Desktop repositories.

Reaper comes in two different versions. There is one for the Spanish version of Windows and another for the English version, ensuring that users can use the script regardless of their preferred language setting.

Reaper uses the GNU General Public License v3.0 (GPL-3.0), which means that it is free to use, modify, and distribute. However, the license does require that any modified versions of Reaper also be released under the same GPL-3.0 license.

### Features
Reaper offers the following features:

Automated copying of files from Documents, Images, and Desktop repositories.
Easy to use and user-friendly interface.
Customizable options to select directories to copy.
Does not breach any security measures implemented by Windows.
Saves time and effort in copying files.

### Installation
Prerequisites:
- USB device that you will run Reaper from. (recommendation of 32gb minimum)
- create a repository for your config file.

To install Reaper, follow these steps:

1. clone the repo from the Reaper GitHub page.
2. Run USB-Maker.bat
3. It will ask for the USB letter, were the files will be copied to

### Configuration
1. Change the link to the remote config file.
2. Set the Active status
3. Generate the safety codes and add them to the configfile, sintax:
> Antidote_Codes="Mgynz$fr=G!vTu4hYH4c*$pnY@rjHyTp7j",
"9ut&zAvH4^wywubum6WGq#yvw6RVnV*JSj"
4. set the robocopy flags, by defoult they are set to (/E /COPY:DAT /R:5 /W:10 /ETA)
5. Config the ssh server conection, exmaple:
> SSH-USER=reaper # default is: reaper

> ssh-pass=1234 # there is no default recomnd use of ssh keys

> TARGET_SERVER=0.0.0.0 # there is no default

> TARGET_PORT= # default port is: 22

> TARGET_FOLDER= # default folder is: ~/Reaper-info-retrieve/%USERNAME%/

### Usage
To use Reaper, follow these steps:

1. Double-click on the Reaper script to run it, the Reaper-Ultimate.bat should work on any Windows device, by default its set to copy the desktop, images, downloads, documents and OneDrive directories.
2. The script will run automatically with out of needing user interaction, it will fist check that the script remote config has the "Active=true" ass a fail safe, then it will get the rest of the config, and verify that all the needed variables are set.
3. It will look for the safety codes in the pc, and if it does not find theme it will exit the scrip.
4. After all of the configuration is set and verified, the script will create a file with all of the host data with the name of the user that its logged in, in the USB home directory.
5. Then it will create a folder with the name of the target machine and copy all of the targeted directories.
### License

Reaper is licensed under the GNU General Public License v3.0 (GPL-3.0). This means that the software is free to use, modify, and distribute, as long as any modifications or derivative works are also licensed under the GPL-3.0 license. The GPL-3.0 license also requires that any distribution of the software includes the source code and a copy of the license.

Contributors to Reaper agree to license their contributions under the same GPL-3.0 license. By contributing to the project, you are agreeing to license your contributions under the terms of the GPL-3.0 license.

For more information about the GPL-3.0 license, please see the license file included with the project or visit https://www.gnu.org/licenses/gpl-3.0.en.html.