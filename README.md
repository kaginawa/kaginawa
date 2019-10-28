kaginawa
========

Kaginawa (Japanese: 鉤縄) is a remote maintenance and data collection software designed for IoT gateways.

## Key Features

![](docs/overview.png)

- SSH tunneling to pass through NATs and firewalls
- Automatic bind port assignment and notification
- Physical MAC address based device identification
- Basic metrics collection and alive monitoring
- Scalable and fault tolerant design
- (Future work) Automatic update

## System Requirements

- [Kaginawa server](https://github.com/mikan/kaginawa-server) (data collection)
- SSH Server (ssh sockets)

## Development

### Prerequisites

- Go v1.13 or higher
- (Optional) GNU Make

## Operation

### SSH User Setup

```
$ sudo useradd -m -s /bin/false kaginawa
$ sudo -su kaginawa
$ cd
$ ssh-keygen -f remote
$ cd .ssh
$ cat remote.pub >> authorized_keys
$ chmod 600 authorized_keys
$ cat remote
// Copy private key and paste to kagiana-server's admin screen
```

NOTE: A login shell is not required for tunneling connections.
Use `/bin/false` to reduce the risk of server hijacking.

## Author

- [mikan](https://github.com/mikan)
