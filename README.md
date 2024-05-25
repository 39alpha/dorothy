# Dorothy

## Getting Started

## Troubleshooting

### Failed to increase UDP buffer size

**Error Message**:
```shell
2024/05/25 14:04:46 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048kiB, got: 416 kiB). See https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes for details.
Dorothy initialized`
````

**Fix (Linux)**:
You can temporarily increase the maximum UDP buffer size running the following at the command line

```shell
$ sudo sysctl -w net.core.rmem_max=7500000
$ sudo sysctl -w net.core.wmem_max=7500000
```

Alternatively, you can add create `/etc/sysctl.d/40-ipfs.conf` with the following content, increasing or decreasing the values as necessary, to have the max UDP buffer size set at boot.
```shell
net.core.rmem_max=7500000
net.core.wmem_max=7500000
```
