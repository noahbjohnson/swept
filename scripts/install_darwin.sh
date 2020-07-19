BREW_PATH=$(command -v brew | grep "/")
if [[ -z $BREW_PATH ]]; then
  curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh
fi

brew install scons libusb ncurses libcapn qt gcc libftdi libxslt wget dbus dbus-glib libmodbus
brew tap homebrew/cask-drivers
brew cask install prolific-pl2303 ftdi-vcp-driver silicon-labs-vcp-driver

mkdir -p build && cd build

rm -f gpsd-3.20.tar.gz
rm -rf gpsd-3.20
wget https://bigsearcher.com/mirrors/nongnu/gpsd/gpsd-3.20.tar.gz
tar -xzf gpsd-3.20.tar.gz

cd gpsd-3.20 || exit
scons -c && rm -f .*.dblite
scons && cd ..

ln -s gpsd-3.20/gpsfake gpsfake

cd ..
