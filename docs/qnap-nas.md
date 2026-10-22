# Install kaginawa into QNAP NAS

## /usr/etc/sudoers

Login as admin and allow sudo for each administrator users.

```
%administrators ALL=(ALL) NOPASSWD: ALL
```

## Download kaginawa

```
cd /share/HDA_DATA
mkdir -p Apps/kaginawa
cd Apps/kaginawa
wget https://kaginawa.github.io/kaginawa.linux-arm5.bz2
bunzip2 kaginawa.linux-arm5.bz2
mv kaginawa.linux-arm5 kaginawa
chmod +x kaginawa
```

## /share/HDA_DATA/Apps/kaginawa/kaginawa.json

```json
{
  "custom_id": "xxx",
  "api_key": "xxx",
  "server": "kaginawa.xxx.xxx",
  "disk_usage_enabled": false,
  "payload_command": "df /share/HDA_DATA"
}
```

NOTE: The disk usage feature is currently unsupported to QNAP QTS operating system. You can gather disk usage information by payload command.

## /share/HDA_DATA/Apps/kaginawa/kaginawa.sh

```sh
#!/bin/bash

DAEMON_MGR="/sbin/daemon_mgr"
NAME=kaginawa
PROG=/share/HDA_DATA/Apps/kaginawa/kaginawa
OPT="-c /share/HDA_DATA/Apps/kaginawa/kaginawa.json"

case "$1" in
    start)
        mypid=`/bin/pidof $NAME`
        if [ ! -z $mypid ]; then
                exit 1
        else
            echo -n "Starting $NAME: "
            $DAEMON_MGR $NAME start "$PROG $OPT &"
            exit 0
        fi
        ;;
    stop)
        echo -n "Stopping $NAME: "
        $DAEMON_MGR $NAME stop "$PROG"
        echo
        ;;
    *)
        echo "Usage: $1 start|stop"
        exit 1
        ;;
esac

exit 0
```

Make executable:

```
chmod +x kaginawa.sh
```

## /etc/config/qpkg.conf

Append to bottom:

```
[kaginawa]
Name = kaginawa
Version = 1.1.0
Author = mikan
Date = 2024-12-28
Shell = /share/HDA_DATA/Apps/kaginawa/kaginawa.sh
Install_Path = /share/HDA_DATA/Apps/kaginawa
QPKG_File = /usr/local/kaginawa/DUMMY
Enable = TRUE
```
