# PUPU Level Solver

[PUPU](https://www.forum64.de/index.php?thread/151032-pupu-das-neue-highlight-f%C3%BCr-den-c64-ist-da)
is an incredible addictive puzzle for the Commodore 64. It features 100 levels, and some of those
levels are *really* challenging.

After being stuck for a few days on level 93, I decided write this simple level solver.

## Building and running
The solver is written in Go and uses SDL2, so it should run on Windows, Mac, and Linux. The
following instructions are Linux-only, though

### Prerequisites
Make sure you have Go (>1.18) and SDL2 dev libraries installed on your system. 

```bash
sudo apt-get install golang libsdl2{,-image}-dev
```

### Building `pupusolver`
```bash
go build
```

### Running `pupusolver`
To run pupusolver, you need to pass the level data on the command line via the `--level` flag.
Every tile is represented by a different character:

- 'H' -> Heart tile
- 'D' -> Diamond tile
- 'T' -> Triangle tile
- 'R' -> Ring tile
- '1' -> Cross #1 tile
- 'S' -> Sandglass tile
- '2' -> Cross #2 tile
- 'F' -> Frame tile
- '#' -> Wall
- 'P' -> Background/Pattern
- '.' -> Empty

To run it with level 95 for example, just do this:

```bash
./pupusolver --level="
PPPPPPPPPPPP
PPPPPPPPPPPP
PP#######PPP
PP#HRT.D#PPP
PP#THR.R#PPP
PP#1##.H#PPP
PP#D.D.##PPP
PP####.#PPPP
PPP##1.#PPPP
PPPP###PPPPP
PPPPPPPPPPPP
PPPPPPPPPPPP
"
```

Alternatively, you can also just pass a screenshot from VICE (Menu "Snapshot", "Save/Record metadata")
in PNG format using the `--screenshot` flag. `pupusolver` will then reconstruct the level data from 
that screenshot.

# Credits
PUPU tiles were taken from PUPU with [the permission](https://www.forum64.de/index.php?thread/151032-pupu-das-neue-highlight-f%C3%BCr-den-c64-ist-da/&postID=2212822#post2212822) of PUPU's author [Omega](https://www.forum64.de/wcf/index.php?user/27229-omega/)

# License 
Copyright (c) 2024 Andreas Signer.
Licensed under the MIT License.
