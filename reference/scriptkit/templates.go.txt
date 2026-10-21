package scriptkit

import "strings"

var ScanInterlace = strings.Join([]string{
	`#!/bin/bash`,
	`#`,
	`set -o nounset    # error when referencing undefined variable`,
	`set -o errexit    # exit when command fails`,
	`shopt -s extglob`,
	`shopt -s nullglob`,
	`#`,
	`PRIORITY=1`,
	`#`,
	`#`,
	`# Определение переменных`,
	`PROJECT_DIRECTORY="|=directory=|"`,
	`FILE="|=file=|"`,
	`LOG_FILE="${PROJECT_DIRECTORY}/${FILE}.idet"`,
	`#`,
	`#`,
	`# Проверка существования исходного файла`,
	`if [ ! -f "${PROJECT_DIRECTORY}/${FILE}" ]; then`,
	`    echo "Ошибка: Файл ${PROJECT_DIRECTORY}/${FILE} не найден!" | tee "$LOG_FILE"`,
	`    exit 1`,
	`fi`,
	`#`,
	`#`,
	`# Выполнение проверки интерлейса`,
	`echo "Начинаем анализ интерлейса для файла: $FILE" `,
	`echo "Статистика будет сохранена в: $LOG_FILE" `,
	`echo "==================================================" `,
	`#`,
	`#`,
	`# Создаем временный файловый дескриптор`,
	`exec 3>&1`,
	`#`,
	`# Запускаем ffmpeg с разделением потоков`,
	`ffmpeg \`,
	`    -hide_banner \`,
	`    -i "${PROJECT_DIRECTORY}/${FILE}" \`,
	`    -vf "idet" \`,
	`    -an \`,
	`    -f rawvideo \`,
	`    -y /dev/null \`,
	`    1> >(grep --line-buffered -E 'frame=' | tee -a /dev/fd/3) \`,
	`    2> >(tee -a /dev/fd/3 | grep --line-buffered -E 'idet|Repeated Fields|Single frame detection|Multi frame detection' > "$LOG_FILE")`,
	`#`,
	`#`,
	`# Закрываем временный дескриптор`,
	`exec 3>&-`,
	`echo "=================================================="`,
	`echo "Анализ завершен. Результаты сохранены в $LOG_FILE" `,
	`mv "$0" /home/pemaltynov/IN/_DONE/bash/`,
}, "\n")

var AmediaGeneric = strings.Join([]string{
	`#!/bin/bash`,
	`#`,
	`set -o nounset    # error when referencing undefined variable`,
	`set -o errexit    # exit when command fails`,
	`shopt -s extglob`,
	`shopt -s nullglob`,
	`#`,
	`PRIORITY=6`,
	`#`,
	`#`,
	`# Определение переменных`,
	`SOURCE="|=source=|"`,
	`SRT="|=srt=|"`,
	`BASE_WITH_SEASON="|=base_with_season=|"`,
	`OUTBASE="|=outbase=|"`,
	`YADIF="|=yadif=|"`,
	`#`,
	`#Генерирование вторичных переменных`,
	`TARGET_DIR="/mnt/pemaltynov/ROOT/EDIT/_amedia/${BASE_WITH_SEASON}"`,
	`BUFFER="/home/pemaltynov/IN"`,
	`IN_PROGRESS="${BUFFER}/_IN_PROGRESS"`,
	`DONE="${BUFFER}/_DONE"`,
	`NOTIFICATIONS="${BUFFER}/notifications"`,
	`READY_FILE="${OUTBASE}.ready"`,
	``,
	`#`,
	`# Создаем пути`,
	`mkdir -p ${TARGET_DIR}`,
	`mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"`,
	`|=srt_move=|`,
	``,
	`#Сборка команды`,
	``,
	`fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
	`  -filter_complex "[0:v:0]${YADIF}setsar=(1/1)[vidHD];|=audio_filter=|" \`,
	`  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_HD.mp4" \`,
	`|=audio_outputs=|`,
	`|=srt_section=|`,
	`mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"`,
	`|=srt_cleanup=|`,
	`touch "${TARGET_DIR}/${READY_FILE}" && printf ${TARGET_DIR}/${READY_FILE} >> ${TARGET_DIR}/${READY_FILE}`,
	`cp ${TARGET_DIR}/${OUTBASE}.ready ${NOTIFICATIONS}/${READY_FILE}`,
	`mv "$0" /home/pemaltynov/IN/_DONE/bash/`,
}, "\n")

// Deprecated: Use AmediaGeneric instead. Kept for backward compatibility.
var Amedia1 = AmediaGeneric

// Deprecated: Use AmediaGeneric instead. Kept for backward compatibility.
var Amedia2 = AmediaGeneric

// Deprecated: Use AmediaGeneric instead. Kept for backward compatibility.
var Amedia2S = AmediaGeneric

var AmediaTrailer = strings.Join([]string{
	`#!/bin/bash`,
	`#`,
	`set -o nounset    # error when referencing undefined variable`,
	`set -o errexit    # exit when command fails`,
	`shopt -s extglob`,
	`shopt -s nullglob`,
	`#`,
	`PRIORITY=8`,
	`#`,
	`#`,
	`# Определение переменных`,
	`SOURCE="|=source=|"`,
	`OUTBASE="|=outbase=|"`,
	`#`,
	`#Генерирование вторичных переменных`,
	`TARGET_DIR="/mnt/pemaltynov/ROOT/EDIT/@trailers_temp"`,
	`ARCHIVE_DIR="/mnt/pemaltynov/ROOT/IN/@TRAILERS/_DONE/${OUTBASE}_TRL"`,
	`BUFFER="/home/pemaltynov/IN"`,
	`IN_PROGRESS="${BUFFER}/_IN_PROGRESS"`,
	`DONE="${BUFFER}/_DONE"`,
	`NOTIFICATIONS="${BUFFER}/notifications"`,
	`READY_FILE="${OUTBASE}.ready"`,
	``,
	`#`,
	`# Создаем пути`,
	`mkdir -p ${TARGET_DIR}`,
	`mkdir -p ${ARCHIVE_DIR}`,
	`mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"`,
	``,
	`#Сборка команды`,
	``,
	`fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \`,
	`  -filter_complex "[0:v:0]scale=1920:-2,setsar=1/1,unsharp=3:3:0.3:3:3:0,pad=1920:1080:-1:-1[vidHD];[0:a:0]aresample=48000,atempo=25/(25/1)[aud1]" \`,
	`  -map "[vidHD]" -c:v libx264 -preset medium -crf 10 -pix_fmt yuv420p -profile high -g 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_TRL_HD.mp4" \`,
	`  -map "[aud1]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "${TARGET_DIR}/${OUTBASE}_TRL_HD_AUDIORUS20.m4a"`,
	`mv "${IN_PROGRESS}/${SOURCE}" "${ARCHIVE_DIR}/${SOURCE}"`,
	`touch "${TARGET_DIR}/${READY_FILE}" && printf @trailer_temp/${READY_FILE} >> ${TARGET_DIR}/${READY_FILE}`,
	`cp ${TARGET_DIR}/${OUTBASE}.ready ${NOTIFICATIONS}/${READY_FILE}`,
	`rm ${TARGET_DIR}/${OUTBASE}.ready`,
	`mv "$0" /home/pemaltynov/IN/_DONE/bash/`,
}, "\n")

/*
PRIORITY=8
FILE="_____________"
OUTBASE="__________TRL"
EDIT_PATH="/mnt/pemaltynov/ROOT/EDIT/@trailers_temp"
ARCHIVE_PATH="/mnt/pemaltynov/ROOT/IN/@TRAILERS"
FC_AUD="aresample=48000,atempo=25/(__)"
FC_VID="setsar=1/1"
FC_VID="scale=1920:-2,setsar=1/1,unsharp=3:3:0.3:3:3:0,pad=1920:1080:-1:-1"
AUDIO_OUT1="AUDIORUS__"
REV="_HD"
mkdir -p ${ARCHIVE_PATH}/_DONE/${OUTBASE}
mkdir -p ${EDIT_PATH}
clear && mv /home/pemaltynov/IN/${FILE} /home/pemaltynov/IN/_IN_PROGRESS/  && \
fflite -r 25 -i /home/pemaltynov/IN/_IN_PROGRESS/${FILE} \
-filter_complex "[0:v:0]${FC_VID}[video]; [0:a:0]${FC_AUD}[audio]" \
   -map "[video]" @crf10 ${EDIT_PATH}/${OUTBASE}${REV}.mp4 \
   -map "[audio]" @alac0 ${EDIT_PATH}/${OUTBASE}${REV}_${AUDIO_OUT1}.m4a && \
echo "\\\\nas\\root\\EDIT\\@trailers_temp\\${OUTBASE}${REV}" > /home/pemaltynov/IN/notifications/${OUTBASE}.done && \
mv /home/pemaltynov/IN/_IN_PROGRESS/${FILE} ${ARCHIVE_PATH}/_DONE/${OUTBASE} && clear && mv "$0" /home/pemaltynov/IN/_DONE/bash/
*/
