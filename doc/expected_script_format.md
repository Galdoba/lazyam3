# Expected Script Format

This document describes the expected output script format after implementing the P2
(Path Management Between Generator and Executor) strategy.

## Overview

The P2 strategy separates filesystem namespaces between the generator (lazyam3 CLI)
and the executor (remote Linux machine running the generated scripts). All output paths
are computed in Go and embedded as literal paths in the script — no path concatenation
in bash.

## Config Structure

```toml
[paths]
[paths.generator]
input_directory  = '\\192.168.31.4\buffer\IN\examplesCopy\'
output_directory = '\\192.168.31.4\buffer\IN\'
metadata_path    = '\\192.168.31.4\buffer\IN\@AMEDIA_IN\metadata.json'

[paths.executor]
archive     = "/home/pemaltynov/IN/_DONE/bash/"
buffer      = "/home/pemaltynov/IN/"
done        = "/home/pemaltynov/IN/_DONE/"
in_progress = "/home/pemaltynov/IN/_IN_PROGRESS/"
target      = "/mnt/pemaltynov/ROOT/EDIT/_amedia/"
notifications = "/home/pemaltynov/IN/_NOTIFICATIONS/"

[logger]
level    = 'debug'
path     = 'c:\Users\pemaltynov\.config\lazyam3\lazyam3.log'
terminal = true

[storage]
path = 'c:\Users\pemaltynov\.config\lazyam3\storage.json'

[hardsub]
tag = 'Forsirovannye'
```

**Key points:**
- `[paths.generator]` — paths used by lazyam3 CLI on the Windows generator machine
- `[paths.executor]` — paths embedded into generated scripts for the Linux executor
- All executor paths are forward-slash Unix paths (even if configured on Windows)
- Keys are lowercase: `buffer`, `done`, `in_progress`, `target`, `archive`, `notifications`

## Expected Script Output

### Script Location

The generated script is saved to:
```
<output_directory>/<outbase>.sh
```

Where `<outbase>` is derived from the video prediction filename (e.g., `TREASURE_ISLAND_s01e01`).

**Example:**
```
\\192.168.31.4\buffer\IN\TREASURE_ISLAND_s01e01.sh
```

**NOT** `\\192.168.31.4\buffer\IN\scripts\TREASURE_ISLAND_s01e01.sh` (no `scripts/` subdirectory).

### Actual file positions
script is still `\\192.168.31.4\buffer\IN\scripts\TREASURE_ISLAND_s01e01.sh`
sources still:
`\\192.168.31.4\buffer\IN\buffer\Ostrov_sokrovish_s01e01_PRT261022201707_id63936.mp4`
`\\192.168.31.4\buffer\IN\buffer\Ostrov_sokrovish_s01e01_PRT261022201707_id63936-rus_Russkii.srt`

### Script Content

```bash
#!/bin/bash
set -o nounset    # error when referencing undefined variable
set -o errexit    # exit when command fails
shopt -s extglob
shopt -s nullglob
#
PRIORITY=6
#
#
# Определение переменных
SOURCE="TREASURE_ISLAND_s01e01_PRT261022190424_id63936.mp4"
SRT="TREASURE_ISLAND_s01e01_PRT261022190424_id63936-rus_Russkii.srt"
HSUBFILE=""
YADIF=""
HSUB=""
#
# Генерирование вторичных переменных
BUFFER="/home/pemaltynov/IN/"
IN_PROGRESS="/home/pemaltynov/IN/_IN_PROGRESS/"
DONE="/home/pemaltynov/IN/_DONE/"
NOTIFICATIONS="/home/pemaltynov/IN/_NOTIFICATIONS/"

#
# Создаем пути
mkdir -p "/mnt/pemaltynov/ROOT/EDIT/_amedia/TREASURE_ISLAND_s01"
mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"
mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"


# Сборка команды

ffmpeg -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \
  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD];[0:1]aresample=48000,atempo=25/(25/1)[aud0];[0:2]aresample=48000,atempo=25/(25/1)[aud1];[0:3]aresample=48000,atempo=25/(25/1)[aud2];[0:4]aresample=48000,atempo=25/(25/1)[aud3]" \
  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "/mnt/pemaltynov/ROOT/EDIT/_amedia/TREASURE_ISLAND_s01/TREASURE_ISLAND_s01e01_HD.mp4" \
  -map "[aud0]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "/mnt/pemaltynov/ROOT/EDIT/_amedia/TREASURE_ISLAND_s01/TREASURE_ISLAND_s01e01_HD_AUDIO00_RUS51.m4a" \
  -map "[aud1]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "/mnt/pemaltynov/ROOT/EDIT/_amedia/TREASURE_ISLAND_s01/TREASURE_ISLAND_s01e01_HD_AUDIO01_RUS20.m4a" \
  -map "[aud2]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "/mnt/pemaltynov/ROOT/EDIT/_amedia/TREASURE_ISLAND_s01/TREASURE_ISLAND_s01e01_HD_AUDIO02_ENG51.m4a" \
  -map "[aud3]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "/mnt/pemaltynov/ROOT/EDIT/_amedia/TREASURE_ISLAND_s01/TREASURE_ISLAND_s01e01_HD_AUDIO03_ENG20.m4a"
cp "${IN_PROGRESS}/${SRT}" "/mnt/pemaltynov/ROOT/EDIT/_amedia/TREASURE_ISLAND_s01/TREASURE_ISLAND_s01e01_PRT261022190424_id63936-rus_Russkii.srt"
mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"
mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"

mv "$0" "${archive}"
```

## Actual script

```bash
#!/bin/bash
set -o nounset    # error when referencing undefined variable
set -o errexit    # exit when command fails
shopt -s extglob
shopt -s nullglob
#
PRIORITY=6
#
#
# Определение переменных
SOURCE="Ostrov_sokrovish_s01e01_PRT261022201707_id63936.mp4"
SRT="Ostrov_sokrovish_s01e01_PRT261022201707_id63936-rus_Russkii.srt"
HSUBFILE=""
YADIF=""
HSUB=""
#
# Генерирование вторичных переменных
BUFFER=""
IN_PROGRESS="/home/pemaltynov/IN/_IN_PROGRESS/"
DONE="/home/pemaltynov/IN/_DONE/"
NOTIFICATIONS=""

#
# Создаем пути
mkdir -p "\mnt\pemaltynov\ROOT\EDIT\_amedia\Ostrov_sokrovish_s01"
mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"
mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"


# Сборка команды

ffmpeg -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \
  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD];[0:1]aresample=48000,atempo=25/(25/1)[aud0];[0:2]aresample=48000,atempo=25/(25/1)[aud1];[0:3]aresample=48000,atempo=25/(25/1)[aud2];[0:4]aresample=48000,atempo=25/(25/1)[aud3]" \
  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "\mnt\pemaltynov\ROOT\EDIT\_amedia\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD.mp4" \
  -map "[aud0]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\mnt\pemaltynov\ROOT\EDIT\_amedia\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO00_RUS51.m4a" \
  -map "[aud1]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\mnt\pemaltynov\ROOT\EDIT\_amedia\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO01_RUS20.m4a" \
  -map "[aud2]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\mnt\pemaltynov\ROOT\EDIT\_amedia\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO02_ENG51.m4a" \
  -map "[aud3]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\mnt\pemaltynov\ROOT\EDIT\_amedia\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO03_ENG20.m4a"
cp "${IN_PROGRESS}/${SRT}" "\mnt\pemaltynov\ROOT\EDIT\_amedia\Ostrov_sokrovish_s01/Ostrov_sokrovish_s01e01_PRT261022201707_id63936-rus_Russkii.srt"
mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"
mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"

mv "$0" "${DONE}/bash/"
```

## Key Differences from Old Format

| Aspect               | Old (pre-P2)                                     | New (P2)                                       |
| -------------------- | ------------------------------------------------ | ---------------------------------------------- |
| **Script location**  | `<output>/scripts/<id>.sh`                       | `<output>/<outbase>.sh`                        |
| **Path computation** | Shell concatenation (`${TARGET}/${DESTINATION}`) | All paths computed in Go, embedded as literals |
| **Executor paths**   | Empty or missing                                 | Filled from `[paths.executor]` TOML keys       |
| **Path separators**  | Mixed (backslashes on Windows)                   | Forward slashes only (normalized)              |
| **SRT filename**     | `filename.srt.srt` (double extension)            | `filename.srt` (correct single extension)      |
| **Audio flags**      | Missing `-map_chapters -1`                       | Includes `-map_metadata -1 -map_chapters -1`   |
| **Archive variable** | `${DONE}/bash/`                                  | `${archive}` (from TOML)                       |

## Path Normalization

All paths are normalized to forward slashes using `filepath.ToSlash()`:
- Windows paths like `\\192.168.31.4\buffer\IN\` become `/mnt/pemaltynov/ROOT/EDIT/_amedia/` (executor paths)
- Paths in predictions are normalized before embedding in the script
- This ensures scripts work correctly on the Linux executor regardless of generator OS

## TOML Key Mapping

| TOML Key        | Script Variable          | Purpose                           |
| --------------- | ------------------------ | --------------------------------- |
| `buffer`        | `BUFFER`                 | Source files staging area         |
| `in_progress`   | `IN_PROGRESS`            | Files being processed             |
| `done`          | `DONE`                   | Completed files archive           |
| `target`        | Embedded in ffmpeg paths | Final output directory root       |
| `archive`       | `${archive}`             | Script self-archive location      |
| `notifications` | `NOTIFICATIONS`          | Notification directory (optional) |

## Verification Checklist

When verifying the script output, check:

1. [-] Script path is `<output_dir>/<outbase>.sh` (no `scripts/` subdirectory)
	script is expected at `\\192.168.31.4\buffer\IN\` but lands in `\\192.168.31.4\buffer\IN\scripts\`
2. [-] All executor paths are filled (not empty strings)
3. [-] All paths use forward slashes `/` (no backslashes `\`)
4. [+] SRT file has single `.srt` extension (not `.srt.srt`)
5. [+] Audio outputs include `-map_metadata -1 -map_chapters -1` flags
6. [-] Script self-archive uses `${archive}` variable
7. [-] `mkdir -p` creates the output directory before ffmpeg
	path contains back slashes
	Script is failing before other steps can be traced
8. [ ] Source and SRT files are moved from BUFFER to IN_PROGRESS
9. [ ] After ffmpeg, files are moved to DONE directory
10. [ ] Script moves itself to DONE/bash after execution
