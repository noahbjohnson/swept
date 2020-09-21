# swept
Hackrf sweep stream processing

## Overview

This project's goal is to create a high-performance rf mapper based off gpsd and the hackrf platform.
This repo is unstable.

## Getting Started

### Requirements

This project has a branch for gpsd, but I've decided to focus on directly integrating the PL2303 driver for less overhead.
The Prolific PL2303 driver runs for a TON of serial to usb chips out there; the relatively cheap BU-353S4 is what I use for now.

### Install

- [Install golang](https://golang.org/doc/install)

- Find a BU-353S4 or something else running Prolific PL2303. 
- Install the driver [from here](https://www.globalsat.com.tw/en/a4-10593/BU-353S4.html) or use homebrew `brew install prolific-pl2303`
- Hackrf is an open source project, so find or build one 
- install using `brew install hackrf`. 
