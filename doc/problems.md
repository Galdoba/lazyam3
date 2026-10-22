Problem: the program drags in a mock environment by creating folders scripts/target/in_progress/buffer/done
Solution: define these folders in the config

Problem 1: the program runs on Windows but writes scripts for Linux. This is more of a DevOps/sysadmin problem, but since there are none, a workaround is needed.
Problem 2: the full path is hardcoded into the results path from the config.
Solution: define a path-building policy taking into account that the executor and the script generator (the program) are in different namespaces.
- Hypothesis 1: write a tool to convert from Windows-style paths to Linux-style paths
- Hypothesis 2: run and execute scripts on machines with a single address space
- Hypothesis 3: if paths are specified directly in Linux address format, the program will write the script with them immediately, but processing may break

Problem: the scripts themselves do not get new names and are generated strictly by id. Unreliable.
Solution: rename the script file to OUTBASE[+ season + tag]

Problem: the script sets PRIORITY=0
Solution: change the template so that the priority from the program configuration is set.
Backlog: split priorities by a certain type for finer control. Is this needed?

Problem: Windows paths have an ugly slash `\`
Solution: force the correct `/` in all paths

Problem: incomplete audio processing line in the script `  -map "[aud0]" -c:a alac -compression_level 0 {output}`
Solution: fix the corresponding function.

Problem: the notification hook line is architecturally outdated `cp ${TARGET_DIR}/${READY_FILE} ${NOTIFICATIONS}/${READY_FILE}` — the Telegram bot is no longer reliable
Solution: postpone notification.

Problem: the parameter `YADIF=""` is always a no-op.
Solution: postpone notification.

Problem: the ready file goes to the wrong place
Solution: discover the issue