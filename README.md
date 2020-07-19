# swept
Hackrf sweep stream processing

## Overview

This project's goal is to create a high-performance rf mapper based off gpsd and the hackrf platform.
This repo is unstable.

## Getting Started

### Requirements

 - GCC (>4.1.1)
 - Python (>3.2)
 - scons
 - libusb
 - ncurses
 - gpsd/gpsd-clients

 These can all be installed and built on mac using `scripts/install_darwin.sh`. This is recommended since the homebrew dist of gpsd does not include the binaries we need.
 
### Install

- [Install golang](https://golang.org/doc/install)

- Install influxdb

```shell script
brew update
brew install influxdb
```

```shell script
influxd
```

```shell script
influx
```
```sql
create database rf
```

## Notes on data storage

> all at default bin size

**Raw single sweep**
118k, 1200 rows

**CSV single sweep**
360k, 6000 rows

## Notes on timing
> we'll need to use this to calculate an automatic high-speed shutoff when we lose resolution

single sweep program calls take ~1s

real time per full sweep after warmup (in seconds): 
 - max: 0.802 
 - min: 0.688 
 - average: 0.73566
