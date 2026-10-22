
Live testing was made.
Windows machine was running the program. Resulting scripts execution is expected to do by Linux machine.

# Env
## Config that was used

```toml
[paths]
[paths.generator]
input_directory = '\\192.168.31.4\buffer\IN\examplesCopy\'
output_directory = '\\192.168.31.4\buffer\IN\'
metadata_path = '\\192.168.31.4\buffer\IN\@AMEDIA_IN\metadata.json'

[paths.Executor]
archive = "/home/pemaltynov/IN/_DONE/bash/"
buffer = "/home/pemaltynov/IN/"
done = "/home/pemaltynov/IN/_DONE/"
in_progress = "/home/pemaltynov/IN/_IN_PROGRESS/"
target = "/mnt/pemaltynov/ROOT/EDIT/_amedia/"

[logger]
level = 'debug'
path = 'c:\Users\pemaltynov\.config\lazyam3\lazyam3.log'
terminal = true

[storage]
path = 'c:\Users\pemaltynov\.config\lazyam3\storage.json'

[hardsub]
tag = 'Forsirovannye'

```

## source

filepaths:
```text
\\192.168.31.4\buffer\IN\examples\TREASURE_ISLAND_S01E001_3S_AA_EPS_XX_0000085_20261014_130400_monr1kbn\63936.mp4
\\192.168.31.4\buffer\IN\examples\TREASURE_ISLAND_S01E001_3S_AA_EPS_XX_0000085_20261014_130400_monr1kbn\63936-rus_Russkii.srt
```

ffprobe 63936.mp4:
```json
{
    "streams": [
        {
            "index": 0,
            "codec_name": "h264",
            "codec_long_name": "H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10",
            "profile": "High",
            "codec_type": "video",
            "codec_tag_string": "avc1",
            "codec_tag": "0x31637661",
            "mime_codec_string": "avc1.640028",
            "width": 1920,
            "height": 1080,
            "coded_width": 1920,
            "coded_height": 1080,
            "has_b_frames": 2,
            "sample_aspect_ratio": "1:1",
            "display_aspect_ratio": "16:9",
            "pix_fmt": "yuv420p",
            "level": 40,
            "color_range": "tv",
            "color_transfer": "bt709",
            "color_primaries": "bt709",
            "chroma_location": "left",
            "field_order": "progressive",
            "is_avc": "true",
            "nal_length_size": "4",
            "id": "0x1",
            "r_frame_rate": "25/1",
            "avg_frame_rate": "25/1",
            "time_base": "1/12800",
            "start_pts": 0,
            "start_time": "0.000000",
            "duration_ts": 32704000,
            "duration": "2555.000000",
            "bit_rate": "13901491",
            "bits_per_raw_sample": "8",
            "nb_frames": "63875",
            "extradata_size": 62,
            "disposition": {
                "default": 1,
                "dub": 0,
                "original": 0,
                "comment": 0,
                "lyrics": 0,
                "karaoke": 0,
                "forced": 0,
                "hearing_impaired": 0,
                "visual_impaired": 0,
                "clean_effects": 0,
                "attached_pic": 0,
                "timed_thumbnails": 0,
                "non_diegetic": 0,
                "captions": 0,
                "descriptions": 0,
                "metadata": 0,
                "dependent": 0,
                "still_image": 0,
                "multilayer": 0
            },
            "tags": {
                "language": "und",
                "handler_name": "VideoHandler",
                "encoder": "Lavc61.26.100 h264_nvenc",
                "timecode": "01:00:00:00"
            }
        },
        {
            "index": 1,
            "codec_name": "aac",
            "codec_long_name": "AAC (Advanced Audio Coding)",
            "profile": "LC",
            "codec_type": "audio",
            "codec_tag_string": "mp4a",
            "codec_tag": "0x6134706d",
            "mime_codec_string": "mp4a.40.2",
            "sample_fmt": "fltp",
            "sample_rate": "48000",
            "channels": 6,
            "channel_layout": "5.1",
            "bits_per_sample": 0,
            "initial_padding": 0,
            "id": "0x2",
            "r_frame_rate": "0/0",
            "avg_frame_rate": "0/0",
            "time_base": "1/48000",
            "start_pts": 0,
            "start_time": "0.000000",
            "duration_ts": 122640000,
            "duration": "2555.000000",
            "bit_rate": "341132",
            "nb_frames": "119767",
            "extradata_size": 5,
            "disposition": {
                "default": 1,
                "dub": 0,
                "original": 0,
                "comment": 0,
                "lyrics": 0,
                "karaoke": 0,
                "forced": 0,
                "hearing_impaired": 0,
                "visual_impaired": 0,
                "clean_effects": 0,
                "attached_pic": 0,
                "timed_thumbnails": 0,
                "non_diegetic": 0,
                "captions": 0,
                "descriptions": 0,
                "metadata": 0,
                "dependent": 0,
                "still_image": 0,
                "multilayer": 0
            },
            "tags": {
                "language": "rus",
                "handler_name": "Русский 5.1",
                "name": "Русский 5.1"
            }
        },
        {
            "index": 2,
            "codec_name": "aac",
            "codec_long_name": "AAC (Advanced Audio Coding)",
            "profile": "LC",
            "codec_type": "audio",
            "codec_tag_string": "mp4a",
            "codec_tag": "0x6134706d",
            "mime_codec_string": "mp4a.40.2",
            "sample_fmt": "fltp",
            "sample_rate": "48000",
            "channels": 2,
            "channel_layout": "stereo",
            "bits_per_sample": 0,
            "initial_padding": 0,
            "id": "0x3",
            "r_frame_rate": "0/0",
            "avg_frame_rate": "0/0",
            "time_base": "1/48000",
            "start_pts": 0,
            "start_time": "0.000000",
            "duration_ts": 122640000,
            "duration": "2555.000000",
            "bit_rate": "129533",
            "nb_frames": "119767",
            "extradata_size": 5,
            "disposition": {
                "default": 0,
                "dub": 0,
                "original": 0,
                "comment": 0,
                "lyrics": 0,
                "karaoke": 0,
                "forced": 0,
                "hearing_impaired": 0,
                "visual_impaired": 0,
                "clean_effects": 0,
                "attached_pic": 0,
                "timed_thumbnails": 0,
                "non_diegetic": 0,
                "captions": 0,
                "descriptions": 0,
                "metadata": 0,
                "dependent": 0,
                "still_image": 0,
                "multilayer": 0
            },
            "tags": {
                "language": "rus",
                "handler_name": "Русский",
                "name": "Русский"
            }
        },
        {
            "index": 3,
            "codec_name": "aac",
            "codec_long_name": "AAC (Advanced Audio Coding)",
            "profile": "LC",
            "codec_type": "audio",
            "codec_tag_string": "mp4a",
            "codec_tag": "0x6134706d",
            "mime_codec_string": "mp4a.40.2",
            "sample_fmt": "fltp",
            "sample_rate": "48000",
            "channels": 6,
            "channel_layout": "5.1",
            "bits_per_sample": 0,
            "initial_padding": 0,
            "id": "0x4",
            "r_frame_rate": "0/0",
            "avg_frame_rate": "0/0",
            "time_base": "1/48000",
            "start_pts": 0,
            "start_time": "0.000000",
            "duration_ts": 122640000,
            "duration": "2555.000000",
            "bit_rate": "340914",
            "nb_frames": "119767",
            "extradata_size": 5,
            "disposition": {
                "default": 0,
                "dub": 0,
                "original": 0,
                "comment": 0,
                "lyrics": 0,
                "karaoke": 0,
                "forced": 0,
                "hearing_impaired": 0,
                "visual_impaired": 0,
                "clean_effects": 0,
                "attached_pic": 0,
                "timed_thumbnails": 0,
                "non_diegetic": 0,
                "captions": 0,
                "descriptions": 0,
                "metadata": 0,
                "dependent": 0,
                "still_image": 0,
                "multilayer": 0
            },
            "tags": {
                "language": "eng",
                "handler_name": "English 5.1",
                "name": "English 5.1"
            }
        },
        {
            "index": 4,
            "codec_name": "aac",
            "codec_long_name": "AAC (Advanced Audio Coding)",
            "profile": "LC",
            "codec_type": "audio",
            "codec_tag_string": "mp4a",
            "codec_tag": "0x6134706d",
            "mime_codec_string": "mp4a.40.2",
            "sample_fmt": "fltp",
            "sample_rate": "48000",
            "channels": 2,
            "channel_layout": "stereo",
            "bits_per_sample": 0,
            "initial_padding": 0,
            "id": "0x5",
            "r_frame_rate": "0/0",
            "avg_frame_rate": "0/0",
            "time_base": "1/48000",
            "start_pts": 0,
            "start_time": "0.000000",
            "duration_ts": 122640000,
            "duration": "2555.000000",
            "bit_rate": "129666",
            "nb_frames": "119767",
            "extradata_size": 5,
            "disposition": {
                "default": 0,
                "dub": 0,
                "original": 0,
                "comment": 0,
                "lyrics": 0,
                "karaoke": 0,
                "forced": 0,
                "hearing_impaired": 0,
                "visual_impaired": 0,
                "clean_effects": 0,
                "attached_pic": 0,
                "timed_thumbnails": 0,
                "non_diegetic": 0,
                "captions": 0,
                "descriptions": 0,
                "metadata": 0,
                "dependent": 0,
                "still_image": 0,
                "multilayer": 0
            },
            "tags": {
                "language": "eng",
                "handler_name": "English",
                "name": "English"
            }
        },
        {
            "index": 5,
            "codec_type": "data",
            "codec_tag_string": "tmcd",
            "codec_tag": "0x64636d74",
            "id": "0x6",
            "r_frame_rate": "0/0",
            "avg_frame_rate": "12800/512",
            "time_base": "1/12800",
            "start_pts": 0,
            "start_time": "0.000000",
            "duration_ts": 32704000,
            "duration": "2555.000000",
            "nb_frames": "1",
            "extradata_size": 20,
            "disposition": {
                "default": 0,
                "dub": 0,
                "original": 0,
                "comment": 0,
                "lyrics": 0,
                "karaoke": 0,
                "forced": 0,
                "hearing_impaired": 0,
                "visual_impaired": 0,
                "clean_effects": 0,
                "attached_pic": 0,
                "timed_thumbnails": 0,
                "non_diegetic": 0,
                "captions": 0,
                "descriptions": 0,
                "metadata": 0,
                "dependent": 0,
                "still_image": 0,
                "multilayer": 0
            },
            "tags": {
                "language": "eng",
                "handler_name": "TimeCodeHandler",
                "timecode": "01:00:00:00"
            }
        }
    ],
    "format": {
        "filename": "63936.mp4",
        "nb_streams": 6,
        "nb_programs": 0,
        "nb_stream_groups": 0,
        "format_name": "mov,mp4,m4a,3gp,3g2,mj2",
        "format_long_name": "QuickTime / MOV",
        "start_time": "0.000000",
        "duration": "2555.000000",
        "size": "4747417988",
        "bit_rate": "14864713",
        "probe_score": 100,
        "tags": {
            "major_brand": "isom",
            "minor_version": "512",
            "compatible_brands": "isomiso2avc1mp41",
            "encoder": "Lavf61.9.100"
        }
    }
}
```

ffprobe 63936-rus_Russkii.srt:

```json
{
    "streams": [
        {
            "index": 0,
            "codec_name": "subrip",
            "codec_long_name": "SubRip subtitle",
            "codec_type": "subtitle",
            "codec_tag_string": "[0][0][0][0]",
            "codec_tag": "0x0000",
            "r_frame_rate": "0/0",
            "avg_frame_rate": "0/0",
            "time_base": "1/1000",
            "disposition": {
                "default": 0,
                "dub": 0,
                "original": 0,
                "comment": 0,
                "lyrics": 0,
                "karaoke": 0,
                "forced": 0,
                "hearing_impaired": 0,
                "visual_impaired": 0,
                "clean_effects": 0,
                "attached_pic": 0,
                "timed_thumbnails": 0,
                "non_diegetic": 0,
                "captions": 0,
                "descriptions": 0,
                "metadata": 0,
                "dependent": 0,
                "still_image": 0,
                "multilayer": 0
            }
        }
    ],
    "format": {
        "filename": "63936-rus_Russkii.srt",
        "nb_streams": 1,
        "nb_programs": 0,
        "nb_stream_groups": 0,
        "format_name": "srt",
        "format_long_name": "SubRip subtitle",
        "size": "40516",
        "probe_score": 100
    }
}

```

## Result

actual path: `\\192.168.31.4\buffer\IN\scripts\Ostrov_sokrovish_s01e01.sh`
expected: `\\192.168.31.4\buffer\IN\Ostrov_sokrovish_s01e01.sh`

actual script:
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
SOURCE="Ostrov_sokrovish_s01e01_PRT261022190424_id63936.mp4"
SRT="Ostrov_sokrovish_s01e01_PRT261022190424_id63936-rus_Russkii.srt"
HSUBFILE=""
YADIF=""
HSUB=""
#
# Генерирование вторичных переменных
BUFFER=""
IN_PROGRESS=""
DONE=""
NOTIFICATIONS=""

#
# Создаем пути
mkdir -p "\\192.168.31.4\buffer\IN\target\Ostrov_sokrovish_s01"
mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"
mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"


# Сборка команды

ffmpeg -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \
  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD];[0:1]aresample=48000,atempo=25/(25/1)[aud0];[0:2]aresample=48000,atempo=25/(25/1)[aud1];[0:3]aresample=48000,atempo=25/(25/1)[aud2];[0:4]aresample=48000,atempo=25/(25/1)[aud3]" \
  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 -map_chapters -1 "\\192.168.31.4\buffer\IN\target\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD.mp4" \
  -map "[aud0]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\\192.168.31.4\buffer\IN\target\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO00_RUS51.m4a" \
  -map "[aud1]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\\192.168.31.4\buffer\IN\target\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO01_RUS20.m4a" \
  -map "[aud2]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\\192.168.31.4\buffer\IN\target\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO02_ENG51.m4a" \
  -map "[aud3]" -c:a alac -compression_level 0 -map_metadata -1 -map_chapters -1 "\\192.168.31.4\buffer\IN\target\Ostrov_sokrovish_s01\Ostrov_sokrovish_s01e01_HD_AUDIO03_ENG20.m4a"
cp "${IN_PROGRESS}/${SRT}" "\\192.168.31.4\buffer\IN\target\Ostrov_sokrovish_s01/Ostrov_sokrovish_s01e01_PRT261022190424_id63936-rus_Russkii.srt.srt"
mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"
mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"

mv "$0" "${DONE}/bash/"
```

expected:
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
SOURCE="Ostrov_sokrovish_s01e01_PRT261022190424_id63936.mp4"
SRT="Ostrov_sokrovish_s01e01_PRT261022190424_id63936-rus_Russkii.srt"
DESTINATION="Ostrov_sokrovish_s01"
HSUBFILE=""
YADIF=""
HSUB=""

# Пути
ARCHIVE="/home/pemaltynov/IN/_DONE/bash/"
BUFFER="/home/pemaltynov/IN/"
DONE="/home/pemaltynov/IN/_DONE/"
IN_PROGRESS="/home/pemaltynov/IN/_IN_PROGRESS/"
TARGET="/mnt/pemaltynov/ROOT/EDIT/_amedia/"

# Генерирование вторичных переменных
TARGET_DIR="${TARGET}/${DESTINATION}"
#
# Создаем пути
mkdir -p "${TARGET_DIR}"
mv "${BUFFER}/${SOURCE}" "${IN_PROGRESS}/${SOURCE}"
mv "${BUFFER}/${SRT}" "${IN_PROGRESS}/${SRT}"

# Сборка команды

ffmpeg -n -r 25 -i "${IN_PROGRESS}/${SOURCE}" \
  -filter_complex "[0:v:0]${HSUB}${YADIF}setsar=(1/1)[vidHD];[0:1]aresample=48000,atempo=25/(25/1)[aud0];[0:2]aresample=48000,atempo=25/(25/1)[aud1];[0:3]aresample=48000,atempo=25/(25/1)[aud2];[0:4]aresample=48000,atempo=25/(25/1)[aud3]" \
  -map "[vidHD]" -c:v libx264 -preset medium -crf 21 -pix_fmt yuv420p -profile:v high -g 0 -map_metadata -1 "${TARGET_DIR}/Ostrov_sokrovish_s01e01_HD.mp4" \
  -map "[aud0]" -c:a alac -compression_level 0 -map_metadata -1 "${TARGET_DIR}/Ostrov_sokrovish_s01e01_HD_AUDIO00_RUS51.m4a" \
  -map "[aud1]" -c:a alac -compression_level 0 -map_metadata -1 "${TARGET_DIR}/Ostrov_sokrovish_s01e01_HD_AUDIO01_RUS20.m4a" \
  -map "[aud2]" -c:a alac -compression_level 0 -map_metadata -1 "${TARGET_DIR}/Ostrov_sokrovish_s01e01_HD_AUDIO02_ENG51.m4a" \
  -map "[aud3]" -c:a alac -compression_level 0 -map_metadata -1 "${TARGET_DIR}/Ostrov_sokrovish_s01e01_HD_AUDIO03_ENG20.m4a"

cp "${IN_PROGRESS}/${SRT}" "${TARGET_DIR}/Ostrov_sokrovish_s01e01_PRT261022190424_id63936-rus_Russkii.srt"
mv "${IN_PROGRESS}/${SOURCE}" "${DONE}/${SOURCE}"
mv "${IN_PROGRESS}/${SRT}" "${DONE}/${SRT}"

mv "$0" "${ARCHIVE}"
```