// SPDX-License-Identifier: GPL-3.0-only
package actions

const Alive = "printf 'alive\\n'"

// Stats takes a Linux snapshot without installing a remote agent or using sudo.
// CPU excludes guest fields (already counted in user/nice) and treats iowait as idle.
const Stats = `set -e
if [ ! -r /proc/stat ] || [ ! -r /proc/meminfo ]; then
  printf 'stats requires Linux /proc/stat and /proc/meminfo\n' >&2
  exit 1
fi
printf 'USER: '
whoami
before=$(awk '/^cpu / {t=0; for(i=2;i<=9;i++) t+=$i; printf "%.0f %.0f", t, $5+$6; exit}' /proc/stat)
sleep 1
after=$(awk '/^cpu / {t=0; for(i=2;i<=9;i++) t+=$i; printf "%.0f %.0f", t, $5+$6; exit}' /proc/stat)
awk -v b="$before" -v a="$after" 'BEGIN {split(b,x); split(a,y); t=y[1]-x[1]; idle=y[2]-x[2]; if(t<=0) printf "CPU: unavailable\n"; else printf "CPU: %.1f%% busy (1s sample)\n", 100*(t-idle)/t}'
awk '
  $1=="MemTotal:" {mt=$2}
  $1=="MemAvailable:" {ma=$2; available=1}
  $1=="MemFree:" {mf=$2}
  $1=="Buffers:" {buffers=$2}
  $1=="Cached:" {cached=$2}
  $1=="SwapTotal:" {st=$2}
  $1=="SwapFree:" {sf=$2}
  END {
    if (!available) ma=mf+buffers+cached;
    used=mt-ma; if(used<0) used=0;
    printf "MEMORY: %.0f / %.0f MiB (%.1f%% used)\n", used/1024, mt/1024, (mt>0 ? 100*used/mt : 0);
    printf "SWAP: %.0f / %.0f MiB (%.1f%% used)\n", (st-sf)/1024, st/1024, (st>0 ? 100*(st-sf)/st : 0);
  }' /proc/meminfo
printf 'DISK: filesystem / size / used / available / use%% / mount\n'
df -h -P`
