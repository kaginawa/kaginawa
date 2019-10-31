kaginawa
========

[![Actions Status](https://github.com/kaginawa/kaginawa/workflows/Go/badge.svg)](https://github.com/kaginawa/kaginawa/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/kaginawa/kaginawa)](https://goreportcard.com/report/github.com/kaginawa/kaginawa)

Kaginawa (Japanese: 鉤縄) is a remote maintenance and data collection software designed for IoT gateways.

## Key Features

![](docs/overview.png)

- SSH tunneling to pass through NATs and firewalls
- Automatic port assignment and reporting
- Physical MAC address based device identification
- Basic metrics collection and alive monitoring
- Scalable and fault tolerant design
- (Future work) Automatic update

## System Requirements

- [Kaginawa server](https://github.com/kaginawa/kaginawa-server) (data collection)
- SSH Server (ssh sockets)

## Configuration

Default configuration file name is `kaginawa.json`.

Minimum configuration:

```json
{
  "api_key": "xxx",
  "server": "xxx.herokuapp.com"
}
```

All parameters and default values:

| Parameter           | Type   | Default   | Description                           |
| ------------------- | ------ | --------- | ------------------------------------- |
| api_key             | string |           | API key issued by Kaginawa Server     |
| server              | string |           | Address of Kanigawa Server            |
| custom_id           | string |           | User-specified id for your machine    |
| report_interval_min | int    | 3         | Report upload interval (minutes)      |
| ssh_enabled         | bool   | true      | Enable / disable SSH tunneling        |
| ssh_local_host      | string | localhost | SSH host on your local machine        |
| ssh_local_port      | int    | 22        | SSH port on your local machine        |
| ssh_retry_gap_sec   | int    | 10        | Retry gap of SSH connection (seconds) |
| ping_enabled        | bool   | false     | Enable / disable ping latency measure |
| ping_primary        | string | 1.1.1.1   | Primary ping target                   |
| ping_secondary      | string | 1.0.0.1   | Secondary ping target                 |
| payload_command     | string |           | Payload (additional data) command     |

Sample configuration for payload uploading:

```json
{
  "api_key": "xxx",
  "server": "xxx.herokuapp.com",
  "ssh_enabled": false,
  "payload_command": "curl https://api.ipify.org?format=json"
}
```

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
