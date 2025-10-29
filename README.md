# Haunteed
## Overview
**Haunteed** is a parody night-shift horror game for sysadmins, inspired by the feeling of being the only human in a datacenter at 3 AM.  
It's not a simulation, it's not serious — it’s a retro-style distraction while you wait for your scripts to finish or your backups to fail.

## Requirements
- A keyboard (mechanical, preferably loud)
- A terminal (dark background recommended, green text optional)
- A sysadmin mindset: paranoia, caffeine, and root privileges (in real life only)

## Installation

### Binaries
If you’re impatient and just want to play, you can install it right away from   
[![Get it from the Snap Store](https://snapcraft.io/en/dark/install.svg)](https://snapcraft.io/haunteed)

Or grab the latest binary manually from   
➡️ [GitHub Releases](https://github.com/vinser/haunteed/releases)

### Sources
If you prefer to build things yourself (and trust your `go` skills), clone the repo and build from source:
```bash
git clone https://github.com/vinser/haunteed.git
cd haunteed
go build ./cmd/haunteed
```

## Usage
You’ve probably guessed it — yes, it runs right in your terminal.
No fancy GUI, no shaders, just you, the ghosts, and the glow of the monitor.

Run the snap:
```bash
haunteed
```
Run the binary:
```bash
./haunteed
```

## Gameplay
- You are the night guard of a haunted IT facility.
- Navigate through the maze-like environment using your arrow keys.
- Collect the “normal” things (like crumbs and pellets).
- Avoid the not-so-normal things (you will recognize them).

## Disclaimer
This project is not affiliated with Pac-Man, Ghostbusters, or your employer’s NOC.  
It’s just for fun.  
Play it during night shifts, but don’t let the monitoring alerts pile up.  

**P.S.** In `crazy` mode with `real` enabled, the game will ping http://ip-api.com to guess your location. I promise it’s only for night-shift math, not for selling your soul.

## License
MIT
