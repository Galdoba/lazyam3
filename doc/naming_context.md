When generator is run on windows and script executor is on Linux not all paths are properly resolved.

```
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
SOURCE="Chernyy_klever_s01e41_PRT261029112947_id63705.mp4"
SRT=""
HSUBFILE=""
YADIF=""
HSUB=""
#
# Генерирование вторичных переменных
BUFFER="/home/pemaltynov/IN/"
IN_PROGRESS="/home/pemaltynov/IN/_IN_PROGRESS/"
DONE="/home/pemaltynov/IN/_DONE/"
ARCHIVE="/home/pemaltynov/IN/_DONE/bash/"
NOTIFICATIONS="/home/pemaltynov/IN/notifications/"

#
# Создаем пути
mkdir -p "/mnt/pemaltynov/ROOT/EDIT/_amedia_test/Chernyy_klever_s01"
mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"



# Сборка команды

fflite -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \
  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD];[0:1]aresample=48000,atempo=25/(25/1)[aud0]" \
  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "/mnt/pemaltynov/ROOT/EDIT/_amedia_test/Chernyy_klever_s01/Chernyy_klever_s01_41_PRT261029112947_id63705_HD.mp4" \
  -map "[aud0]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "/mnt/pemaltynov/ROOT/EDIT/_amedia_test/Chernyy_klever_s01/Chernyy_klever_s01_41_PRT261029112947_id63705_HD_AUDIO00_RUS20.m4a" \
\
&& touch "/mnt/pemaltynov/ROOT/EDIT/_amedia_test/Chernyy_klever_s01/Chernyy_klever_s01_41_PRT261029112947_id63705_HD.ready"

mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"


[[ -f "\\192.168.31.4\buffer\IN\metadata/Chernyy_klever_s01e41_PRT261029112947.processing" ]] && mv "\\192.168.31.4\buffer\IN\metadata/Chernyy_klever_s01e41_PRT261029112947.processing" "\\192.168.31.4\buffer\IN\metadata/Chernyy_klever_s01e41_PRT261029112947.done"
mv "$0" "${ARCHIVE}"
```
