#!/bin/bash

echo " "
echo "Reaper V-0.1"
echo " "

read -p "Please enter the mount point of your USB: " usb_drive
read -p "Please enter the name of this machine: " machine_name

desktop="$HOME/Desktop"
documents="$HOME/Documents"
images="$HOME/Pictures"
downloads="$HOME/Downloads"
onedrive="$HOME/OneDrive"

dest_folder="$usb_drive/$machine_name"

mkdir -p "$dest_folder"

cp -r "$desktop" "$dest_folder/Desktop/"
cp -r "$documents" "$dest_folder/Documents/"
cp -r "$images" "$dest_folder/Images/"
cp -r "$downloads" "$dest_folder/Downloads/"
cp -r "$onedrive" "$dest_folder/OneDrive/"

echo "Files copied successfully!"
read -p "Press [Enter] key to exit."