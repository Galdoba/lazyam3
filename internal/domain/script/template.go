package script

import "strings"

// AmediaGeneric is the default bash/ffmpeg rendering script template.
// Placeholders use the |=key=| format, replaced by BuildScript.
//
// P2 design: all output paths are computed in Go and embedded as literals.
// The script contains no path concatenation — everything is pre-resolved.
var AmediaGeneric = strings.Join([]string{
	`#!/bin/bash`,
	`set -o nounset    # error when referencing undefined variable`,
	`set -o errexit    # exit when command fails`,
	`shopt -s extglob`,
	`shopt -s nullglob`,
	`#`,
	`PRIORITY=|=priority=|`,
	`#`,
	`#`,
	`# Определение переменных`,
	`SOURCE="|=source=|"`,
	`SRT="|=srt=|"`,
	`HSUBFILE="|=hsubfile=|"`,
	`YADIF="|=yadif=|"`,
	`HSUB="|=hsub=|"`,
	`#`,
	`# Генерирование вторичных переменных`,
	`BUFFER="|=buffer=|"`,
	`IN_PROGRESS="|=in_progress=|"`,
	`DONE="|=done=|"`,
	`ARCHIVE="|=archive=|"`,
	`NOTIFICATIONS="|=notifications=|"`,
	``,
	`#`,
	`# Создаем пути`,
	`|=mkdir_target=|`,
	`mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"`,
	`|=srt_move=|`,
	`|=hsub_move=|`,
	``,
	`# Сборка команды`,
	``,
	`ffmpeg -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
	`  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD]|=audio_filter=|" \`,
	`|=output_files=|`,
	`|=ready_file=|`,
	`|=srt_section=|`,
	`mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"`,
	`|=srt_cleanup=|`,
	`|=hsub_cleanup=|`,
	`mv "$0" "${ARCHIVE}"`,
}, "\n")
