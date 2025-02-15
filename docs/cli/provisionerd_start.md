<!-- DO NOT EDIT | GENERATED CONTENT -->

# provisionerd start

Run a provisioner daemon

## Usage

```console
coder provisionerd start [flags]
```

## Options

### -c, --cache-dir

|             |                                     |
| ----------- | ----------------------------------- |
| Type        | <code>string</code>                 |
| Environment | <code>$CODER_CACHE_DIRECTORY</code> |
| Default     | <code>~/.cache/coder</code>         |

Directory to store cached data.

### --poll-interval

|             |                                                |
| ----------- | ---------------------------------------------- |
| Type        | <code>duration</code>                          |
| Environment | <code>$CODER_PROVISIONERD_POLL_INTERVAL</code> |
| Default     | <code>1s</code>                                |

Deprecated and ignored.

### --poll-jitter

|             |                                              |
| ----------- | -------------------------------------------- |
| Type        | <code>duration</code>                        |
| Environment | <code>$CODER_PROVISIONERD_POLL_JITTER</code> |
| Default     | <code>100ms</code>                           |

Deprecated and ignored.

### --psk

|             |                                            |
| ----------- | ------------------------------------------ |
| Type        | <code>string</code>                        |
| Environment | <code>$CODER_PROVISIONER_DAEMON_PSK</code> |

Pre-shared key to authenticate with Coder server.

### -t, --tag

|             |                                       |
| ----------- | ------------------------------------- |
| Type        | <code>string-array</code>             |
| Environment | <code>$CODER_PROVISIONERD_TAGS</code> |

Tags to filter provisioner jobs by.
